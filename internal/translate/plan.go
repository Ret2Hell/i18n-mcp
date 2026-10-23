package translate

import (
	"context"
	"fmt"
	json "github.com/Ret2Hell/i18n-mcp/internal/jsonutil"
	"maps"
	"slices"
	"time"

	"github.com/Ret2Hell/i18n-mcp/internal/diff"
	"github.com/Ret2Hell/i18n-mcp/internal/pagination"
	"github.com/Ret2Hell/i18n-mcp/internal/security"
	"github.com/Ret2Hell/i18n-mcp/internal/state"
	"github.com/Ret2Hell/i18n-mcp/internal/validate"
)

const (
	translationBatchOperation = "i18n.translation.batch"
	translationBatchTTL       = 24 * time.Hour
)

// Plan creates a translation batch from current diff analysis.
func (s *Service) Plan(ctx context.Context, in PlanInput) (Batch, error) {
	batch, err := s.buildBatch(ctx, in)
	if err != nil {
		return Batch{}, err
	}
	batch.BatchID, err = s.signBatch(ctx, in, batch)
	if err != nil {
		return Batch{}, err
	}
	s.storeLatest(ctx, batch)
	return batch, nil
}

// ResolveBatch verifies a batch handle and reconstructs its plan from current project files.
func (s *Service) ResolveBatch(ctx context.Context, batchID string) (Batch, error) {
	if batchID == "" {
		return Batch{}, fmt.Errorf("batchId is required")
	}
	if s.StateSigner == nil {
		return Batch{}, fmt.Errorf("translation batch signer is not configured")
	}
	claims, err := s.StateSigner.Verify(batchID)
	if err != nil {
		return Batch{}, fmt.Errorf("invalid translation batch handle: %w", err)
	}
	if claims.Subject != security.SubjectFromContext(ctx) || claims.Operation != translationBatchOperation || claims.InputDigest != state.SourceHash(s.guard.Root()) {
		return Batch{}, fmt.Errorf("translation batch handle does not match this project or subject")
	}
	var in PlanInput
	if err := json.Unmarshal(claims.Data, &in); err != nil {
		return Batch{}, fmt.Errorf("translation batch handle contains invalid plan input: %w", err)
	}
	batch, err := s.buildBatch(ctx, in)
	if err != nil {
		return Batch{}, err
	}
	digest, err := batchDigest(s.guard.Root(), batch)
	if err != nil {
		return Batch{}, fmt.Errorf("digest translation batch: %w", err)
	}
	if digest != claims.PlanDigest {
		return Batch{}, fmt.Errorf("translation batch is stale; create a new plan")
	}
	batch.BatchID = batchID
	return batch, nil
}

func (s *Service) buildBatch(ctx context.Context, in PlanInput) (Batch, error) {
	limit := in.MaxItems
	if limit == 0 {
		limit = pagination.DefaultLimit
	}
	query := in
	query.Cursor = ""
	query.MaxItems = limit
	builder, err := pagination.New[Item](ctx, s.StateSigner, "i18n.translation.page", s.guard.Root(), query, pagination.Input{Limit: limit, Cursor: in.Cursor})
	if err != nil {
		return Batch{}, err
	}
	statuses := in.Statuses
	if len(statuses) == 0 {
		statuses = []diff.KeyStatus{diff.Missing, diff.Stale}
	}
	report, err := s.diff.Walk(ctx, diff.Filter{Locales: in.Locales, Namespaces: in.Namespaces, Keys: in.Keys, Statuses: statuses}, func(record diff.KeyDiff) error { return builder.Add(planItem(record)) })
	if err != nil {
		return Batch{}, err
	}
	cfg, err := s.config.Resolve(ctx)
	if err != nil {
		return Batch{}, err
	}

	meta, err := builder.Finish()
	if err != nil {
		return Batch{}, err
	}
	items := builder.Items
	batch := Batch{
		Page:            &meta,
		SourceLocale:    report.SourceLocale,
		TargetLocales:   targetLocalesFromItems(items),
		Items:           items,
		ValidationRules: defaultValidationRules(),
		ResourceLinks:   []string{"i18n://analysis/diff", "i18n://translation/plan/latest"},
		CreatedAt:       time.Now().UTC(),
	}

	if in.IncludeContext {
		styleGuide, glossaryText, glossaryRefs, contextFiles, warnings := s.loadPlanContext(cfg)
		batch.StyleGuide = styleGuide
		batch.GlossaryText = glossaryText
		batch.GlossaryReferences = glossaryRefs
		batch.ContextFiles = contextFiles
		batch.Warnings = warnings
	}
	return batch, nil
}

func (s *Service) signBatch(ctx context.Context, in PlanInput, batch Batch) (string, error) {
	if s.StateSigner == nil {
		return "", fmt.Errorf("translation batch signer is not configured")
	}
	planInput, err := json.Marshal(in)
	if err != nil {
		return "", fmt.Errorf("marshal translation plan input: %w", err)
	}
	digest, err := batchDigest(s.guard.Root(), batch)
	if err != nil {
		return "", fmt.Errorf("digest translation batch: %w", err)
	}
	return s.StateSigner.Sign(security.RequestStateClaims{
		Subject:     security.SubjectFromContext(ctx),
		Operation:   translationBatchOperation,
		InputDigest: state.SourceHash(s.guard.Root()),
		PlanDigest:  digest,
		ExpiresAt:   time.Now().Add(translationBatchTTL).Unix(),
		Data:        planInput,
	})
}

func planItem(record diff.KeyDiff) Item {
	return Item{
		ID:     itemID(record.Locale, record.Namespace, record.Key),
		Locale: record.Locale, Namespace: record.Namespace, Key: record.Key, Status: record.Status,
		SourceValue: record.SourceValue, OldValue: record.TargetValue, SourceHash: record.SourceHash, TargetHash: record.TargetHash,
		Placeholders: validate.ExtractPlaceholders(record.SourceValue), Tags: validate.ExtractTags(record.SourceValue), Notes: planNotes(record),
		SourceFilePath: record.SourceFilePath, TargetFilePath: record.TargetFilePath,
	}
}

func planNotes(record diff.KeyDiff) []string {
	var notes []string
	if record.Status == diff.Stale {
		notes = append(notes, "target exists but source value changed since it was translated")
	}
	if len(record.Validation) > 0 {
		notes = append(notes, "existing target has validation issues")
	}
	return notes
}

func targetLocalesFromItems(items []Item) []string {
	seen := map[string]struct{}{}
	for _, item := range items {
		seen[item.Locale] = struct{}{}
	}
	return slices.Sorted(maps.Keys(seen))
}

func itemID(localeCode string, namespace string, key string) string {
	return localeCode + ":" + namespace + ":" + key
}

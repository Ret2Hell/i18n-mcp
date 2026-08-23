package translate

import (
	"cmp"
	"context"
	"fmt"
	json "github.com/Ret2Hell/i18n-mcp/internal/jsonutil"
	"maps"
	"slices"
	"time"

	"github.com/Ret2Hell/i18n-mcp/internal/diff"
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
	report, err := s.diff.Analyze(ctx)
	if err != nil {
		return Batch{}, err
	}
	cfg, err := s.config.Resolve(ctx)
	if err != nil {
		return Batch{}, err
	}

	items := planItems(report, in)
	batch := Batch{
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

func planItems(report diff.Report, in PlanInput) []Item {
	statuses := planStatusSet(in.Statuses)
	locales := stringSet(in.Locales)
	namespaces := stringSet(in.Namespaces)
	keys := stringSet(in.Keys)

	var items []Item
	for _, record := range report.Items {
		if !statuses[record.Status] {
			continue
		}
		if len(locales) > 0 && !locales[record.Locale] {
			continue
		}
		if len(namespaces) > 0 && !namespaces[record.Namespace] {
			continue
		}
		if len(keys) > 0 && !keys[record.Key] {
			continue
		}
		items = append(items, Item{
			ID:             itemID(record.Locale, record.Namespace, record.Key),
			Locale:         record.Locale,
			Namespace:      record.Namespace,
			Key:            record.Key,
			Status:         record.Status,
			SourceValue:    record.SourceValue,
			OldValue:       record.TargetValue,
			SourceHash:     record.SourceHash,
			TargetHash:     record.TargetHash,
			Placeholders:   validate.ExtractPlaceholders(record.SourceValue),
			Tags:           validate.ExtractTags(record.SourceValue),
			Notes:          planNotes(record),
			SourceFilePath: record.SourceFilePath,
			TargetFilePath: record.TargetFilePath,
		})
	}
	slices.SortFunc(items, compareItem)
	if in.MaxItems > 0 && len(items) > in.MaxItems {
		items = items[:in.MaxItems]
	}
	return items
}

func planStatusSet(statuses []diff.KeyStatus) map[diff.KeyStatus]bool {
	if len(statuses) == 0 {
		return map[diff.KeyStatus]bool{diff.Missing: true, diff.Stale: true}
	}
	set := map[diff.KeyStatus]bool{}
	for _, status := range statuses {
		set[status] = true
	}
	return set
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

func compareItem(a, b Item) int {
	return cmp.Or(
		cmp.Compare(a.Locale, b.Locale),
		cmp.Compare(a.Namespace, b.Namespace),
		cmp.Compare(a.Key, b.Key),
	)
}

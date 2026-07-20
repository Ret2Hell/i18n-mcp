package translate

import (
	"cmp"
	"context"
	"maps"
	"slices"
	"time"

	"github.com/Ret2Hell/i18n-mcp/internal/diff"
	"github.com/Ret2Hell/i18n-mcp/internal/validate"
)

// Plan creates a translation batch from current diff analysis.
func (s *Service) Plan(ctx context.Context, in PlanInput) (Batch, error) {
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
		styleGuide, glossaryRefs, contextFiles, warnings := s.loadPlanContext(cfg)
		batch.StyleGuide = styleGuide
		batch.GlossaryReferences = glossaryRefs
		batch.ContextFiles = contextFiles
		batch.Warnings = warnings
	}

	batch.BatchID = buildBatchID(s.guard.Root(), batch)
	s.storeLatest(ctx, batch)
	return batch, nil
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

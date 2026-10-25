package diff

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/Ret2Hell/i18n-mcp/internal/locale"
	"github.com/Ret2Hell/i18n-mcp/internal/state"
	"github.com/Ret2Hell/i18n-mcp/internal/validate"
)

// Service analyzes differences between source and target keys.
type Service struct {
	locales      *locale.Service
	stateService *state.Service
	validator    *validate.Service
}

// NewService creates a project-scoped analyzer.
func NewService(locales *locale.Service, stateService *state.Service, validator *validate.Service) *Service {
	return &Service{locales: locales, stateService: stateService, validator: validator}
}

// Analyze is the explicit full-detail path for local exports and internal callers.
func (s *Service) Analyze(ctx context.Context) (Report, error) {
	var items []KeyDiff
	report, err := s.Walk(ctx, Filter{}, func(item KeyDiff) error { items = append(items, item); return nil })
	report.Items = items
	return report, err
}

// Walk visits sorted matching records and aggregates counts without retaining them.
// Filters are applied before validation. Summary counts describe the filtered scope.
func (s *Service) Walk(ctx context.Context, filter Filter, visit func(KeyDiff) error) (Report, error) {
	inv, err := s.locales.InventoryFiltered(ctx, filter.Locales, filter.Namespaces)
	if err != nil {
		return Report{}, err
	}
	return s.WalkInventory(ctx, inv, filter, visit)
}

// WalkInventory avoids loading a second inventory when generating reports.
func (s *Service) WalkInventory(ctx context.Context, inv locale.Inventory, filter Filter, visit func(KeyDiff) error) (Report, error) {
	if err := ctx.Err(); err != nil {
		return Report{}, err
	}
	for _, status := range filter.Statuses {
		if !slices.Contains([]KeyStatus{Current, Missing, Stale, Extra, Invalid, Unknown, SourceInvalid}, status) {
			return Report{}, fmt.Errorf("unknown diff status %q", status)
		}
	}
	stateFile, err := s.stateService.Load(ctx)
	if err != nil {
		return Report{}, err
	}
	sources, targets := indexUnits(inv)
	analysis := inventoryAnalysis{service: s, inventory: inv, state: stateFile, sources: sources, targets: targets, checkedSources: map[string][]validate.Issue{}}
	keys := matchingIdentities(inv, filter)
	report := Report{SourceLocale: inv.SourceLocale, TargetLocales: targetLocales(inv), Warnings: inventoryWarnings(inv.Warnings)}
	for _, code := range report.TargetLocales {
		if len(filter.Locales) > 0 && !slices.Contains(filter.Locales, code) {
			continue
		}
		for _, id := range keys {
			if err := ctx.Err(); err != nil {
				return Report{}, err
			}
			item, ok := analysis.classify(ctx, code, id, &report)
			if !ok {
				continue
			}
			if len(filter.Statuses) > 0 && !slices.Contains(filter.Statuses, item.Status) {
				continue
			}
			report.Summary.Add(item)
			if visit != nil {
				if err := visit(item); err != nil {
					return Report{}, err
				}
			}
		}
	}
	return report, nil
}

func indexUnits(inv locale.Inventory) (map[string]locale.Unit, map[string]locale.Unit) {
	sources := make(map[string]locale.Unit)
	targets := make(map[string]locale.Unit)
	for _, unit := range inv.Units {
		if unit.Locale == inv.SourceLocale {
			sources[sourceIdentity(unit.Namespace, unit.Key)] = unit
		} else {
			targets[targetIdentity(unit.Locale, unit.Namespace, unit.Key)] = unit
		}
	}
	return sources, targets
}
func targetLocales(inv locale.Inventory) []string {
	out := slices.Clone(inv.TargetLocales)
	if len(out) == 0 {
		for _, code := range inv.Locales {
			if code != inv.SourceLocale {
				out = append(out, code)
			}
		}
	}
	slices.Sort(out)
	return out
}
func inventoryWarnings(warnings []locale.Warning) []string {
	out := make([]string, 0, len(warnings))
	for _, warning := range warnings {
		if warning.FilePath != "" || warning.Key != "" {
			out = append(out, fmt.Sprintf("%s: %s (%s %s)", warning.Code, warning.Message, warning.FilePath, warning.Key))
		} else {
			out = append(out, warning.Code+": "+warning.Message)
		}
	}
	slices.Sort(out)
	return out
}
func sourceIdentity(namespace, key string) string { return namespace + "\x00" + key }
func targetIdentity(code, namespace, key string) string {
	return code + "\x00" + namespace + "\x00" + key
}
func lessKeyDiff(a, b KeyDiff) bool { return compareKeyDiff(a, b) < 0 }
func compareKeyDiff(a, b KeyDiff) int {
	return cmp.Or(cmp.Compare(a.Locale, b.Locale), cmp.Compare(a.Namespace, b.Namespace), cmp.Compare(a.Key, b.Key), cmp.Compare(a.Status, b.Status))
}

func (s *Service) classifyExisting(ctx context.Context, inv locale.Inventory, sourceUnit locale.Unit, targetUnit locale.Unit, stateFile state.File) KeyDiff {
	currentSourceHash := state.SourceHash(sourceUnit.Value)
	entry, ok := stateFile.Entries[state.EntryKey(targetUnit.Locale, targetUnit.Namespace, targetUnit.Key)]
	status := Current
	if !ok {
		status = Unknown
	} else if entry.TranslatedFromHash != currentSourceHash {
		status = Stale
	}

	policy := inv.Validation.ForKey(sourceUnit.Namespace, sourceUnit.Key)
	validation := s.validator.ValidatePairContext(ctx, validate.Pair{
		Options:      validate.Options{MessageFormat: policy.MessageFormat, Printf: policy.Printf},
		SourceLocale: inv.SourceLocale,
		Locale:       targetUnit.Locale,
		Namespace:    targetUnit.Namespace,
		Key:          targetUnit.Key,
		Source:       sourceUnit.Value,
		Target:       targetUnit.Value,
	})
	validationIssues := slices.Concat(validation.Issues, validation.Warnings)
	if !validation.OK {
		status = Invalid
	}

	return KeyDiff{
		Locale:             targetUnit.Locale,
		Namespace:          targetUnit.Namespace,
		Key:                targetUnit.Key,
		Status:             status,
		SourceValue:        sourceUnit.Value,
		TargetValue:        targetUnit.Value,
		SourceHash:         currentSourceHash,
		TranslatedFromHash: entry.TranslatedFromHash,
		TargetHash:         entry.TargetHash,
		SourceFilePath:     sourceUnit.FilePath,
		TargetFilePath:     targetUnit.FilePath,
		Validation:         validationIssues,
	}
}

package diff

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/Ret2Hell/i18n-mcp/internal/locale"
	"github.com/Ret2Hell/i18n-mcp/internal/state"
	"github.com/Ret2Hell/i18n-mcp/internal/validate"
)

type Service struct {
	locales      *locale.Service
	stateService *state.Service
	validator    *validate.Service
}

func NewService(locales *locale.Service, stateService *state.Service, validator *validate.Service) *Service {
	return &Service{locales: locales, stateService: stateService, validator: validator}
}

func (s *Service) Analyze(ctx context.Context) (Report, error) {
	inv, err := s.locales.Inventory(ctx)
	if err != nil {
		return Report{}, err
	}

	stateFile, err := s.stateService.Load(ctx)
	if err != nil {
		return Report{}, err
	}

	sourceByKey, targetByKey := indexUnits(inv)
	targetLocaleCodes := targetLocales(inv)
	report := Report{
		SourceLocale:  inv.SourceLocale,
		TargetLocales: slices.Clone(inv.TargetLocales),
		Warnings:      inventoryWarnings(inv.Warnings),
	}

	for _, sourceUnit := range sortedUnits(sourceByKey) {
		for _, localeCode := range targetLocaleCodes {
			key := targetIdentity(localeCode, sourceUnit.Namespace, sourceUnit.Key)
			targetUnit, ok := targetByKey[key]
			if !ok {
				report.Items = append(report.Items, KeyDiff{
					Locale:         localeCode,
					Namespace:      sourceUnit.Namespace,
					Key:            sourceUnit.Key,
					Status:         Missing,
					SourceValue:    sourceUnit.Value,
					SourceFilePath: sourceUnit.FilePath,
				})
				continue
			}

			report.Items = append(report.Items, s.classifyExisting(inv.SourceLocale, sourceUnit, targetUnit, stateFile))
		}
	}

	targetSet := targetLocaleSet(targetLocaleCodes)
	for _, targetUnit := range sortedUnits(targetByKey) {
		if _, ok := targetSet[targetUnit.Locale]; !ok {
			continue
		}
		if _, ok := sourceByKey[sourceIdentity(targetUnit.Namespace, targetUnit.Key)]; ok {
			continue
		}
		report.Items = append(report.Items, KeyDiff{
			Locale:         targetUnit.Locale,
			Namespace:      targetUnit.Namespace,
			Key:            targetUnit.Key,
			Status:         Extra,
			TargetValue:    targetUnit.Value,
			TargetFilePath: targetUnit.FilePath,
		})
	}

	sortReport(&report)
	report.Summary = Summarize(report.Items)
	return report, nil
}

func indexUnits(inv locale.Inventory) (map[string]locale.Unit, map[string]locale.Unit) {
	sourceByKey := make(map[string]locale.Unit, len(inv.Units))
	targetByKey := make(map[string]locale.Unit, len(inv.Units))
	for _, unit := range inv.Units {
		if unit.Locale == inv.SourceLocale {
			sourceByKey[sourceIdentity(unit.Namespace, unit.Key)] = unit
			continue
		}
		targetByKey[targetIdentity(unit.Locale, unit.Namespace, unit.Key)] = unit
	}
	return sourceByKey, targetByKey
}

func targetLocales(inv locale.Inventory) []string {
	if len(inv.TargetLocales) > 0 {
		out := slices.Clone(inv.TargetLocales)
		slices.Sort(out)
		return out
	}
	var out []string
	for _, localeCode := range inv.Locales {
		if localeCode != inv.SourceLocale {
			out = append(out, localeCode)
		}
	}
	slices.Sort(out)
	return out
}

func targetLocaleSet(locales []string) map[string]struct{} {
	set := make(map[string]struct{}, len(locales))
	for _, localeCode := range locales {
		set[localeCode] = struct{}{}
	}
	return set
}

func sortedUnits(units map[string]locale.Unit) []locale.Unit {
	return slices.SortedFunc(maps.Values(units), compareUnit)
}

func compareUnit(a, b locale.Unit) int {
	return cmp.Or(
		cmp.Compare(a.Locale, b.Locale),
		cmp.Compare(a.Namespace, b.Namespace),
		cmp.Compare(a.Key, b.Key),
	)
}

func inventoryWarnings(warnings []locale.Warning) []string {
	out := make([]string, 0, len(warnings))
	for _, warning := range warnings {
		if warning.FilePath != "" || warning.Key != "" {
			out = append(out, fmt.Sprintf("%s: %s (%s %s)", warning.Code, warning.Message, warning.FilePath, warning.Key))
			continue
		}
		out = append(out, warning.Code+": "+warning.Message)
	}
	slices.Sort(out)
	return out
}

func sourceIdentity(namespace string, key string) string {
	return namespace + "\x00" + key
}

func targetIdentity(localeCode string, namespace string, key string) string {
	return localeCode + "\x00" + namespace + "\x00" + key
}

func sortReport(report *Report) {
	slices.Sort(report.TargetLocales)
	slices.Sort(report.Warnings)
	slices.SortFunc(report.Items, compareKeyDiff)
}

func lessKeyDiff(a, b KeyDiff) bool {
	return compareKeyDiff(a, b) < 0
}

func compareKeyDiff(a, b KeyDiff) int {
	return cmp.Or(
		cmp.Compare(a.Locale, b.Locale),
		cmp.Compare(a.Namespace, b.Namespace),
		cmp.Compare(a.Key, b.Key),
		cmp.Compare(a.Status, b.Status),
	)
}

func (s *Service) classifyExisting(sourceLocale string, sourceUnit locale.Unit, targetUnit locale.Unit, stateFile state.File) KeyDiff {
	currentSourceHash := state.SourceHash(sourceUnit.Value)
	entry, ok := stateFile.Entries[state.EntryKey(targetUnit.Locale, targetUnit.Namespace, targetUnit.Key)]
	status := Current
	if !ok {
		status = Unknown
	} else if entry.TranslatedFromHash != currentSourceHash {
		status = Stale
	}

	validation := s.validator.ValidatePair(validate.Pair{
		SourceLocale: sourceLocale,
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

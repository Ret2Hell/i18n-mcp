package diff

import (
	"context"
	"fmt"
	"sort"

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
	report := Report{
		SourceLocale:  inv.SourceLocale,
		TargetLocales: append([]string(nil), inv.TargetLocales...),
		Warnings:      inventoryWarnings(inv.Warnings),
	}

	for _, sourceUnit := range sortedUnits(sourceByKey) {
		for _, localeCode := range targetLocales(inv) {
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

	targetSet := targetLocaleSet(inv)
	for _, targetUnit := range sortedUnits(targetByKey) {
		if !targetSet[targetUnit.Locale] {
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
	return report, nil
}

func indexUnits(inv locale.Inventory) (map[string]locale.Unit, map[string]locale.Unit) {
	sourceByKey := map[string]locale.Unit{}
	targetByKey := map[string]locale.Unit{}
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
		out := append([]string(nil), inv.TargetLocales...)
		sort.Strings(out)
		return out
	}
	var out []string
	for _, localeCode := range inv.Locales {
		if localeCode != inv.SourceLocale {
			out = append(out, localeCode)
		}
	}
	sort.Strings(out)
	return out
}

func targetLocaleSet(inv locale.Inventory) map[string]bool {
	set := map[string]bool{}
	for _, localeCode := range targetLocales(inv) {
		set[localeCode] = true
	}
	return set
}

func sortedUnits(units map[string]locale.Unit) []locale.Unit {
	out := make([]locale.Unit, 0, len(units))
	for _, unit := range units {
		out = append(out, unit)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Locale != out[j].Locale {
			return out[i].Locale < out[j].Locale
		}
		if out[i].Namespace != out[j].Namespace {
			return out[i].Namespace < out[j].Namespace
		}
		return out[i].Key < out[j].Key
	})
	return out
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
	sort.Strings(out)
	return out
}

func sourceIdentity(namespace string, key string) string {
	return namespace + "\x00" + key
}

func targetIdentity(localeCode string, namespace string, key string) string {
	return localeCode + "\x00" + namespace + "\x00" + key
}

func sortReport(report *Report) {
	sort.Strings(report.TargetLocales)
	sort.Strings(report.Warnings)
	sort.Slice(report.Items, func(i, j int) bool {
		return lessKeyDiff(report.Items[i], report.Items[j])
	})
}

func lessKeyDiff(a, b KeyDiff) bool {
	if a.Locale != b.Locale {
		return a.Locale < b.Locale
	}
	if a.Namespace != b.Namespace {
		return a.Namespace < b.Namespace
	}
	if a.Key != b.Key {
		return a.Key < b.Key
	}
	return a.Status < b.Status
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
	validationIssues := append([]validate.Issue{}, validation.Issues...)
	validationIssues = append(validationIssues, validation.Warnings...)
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

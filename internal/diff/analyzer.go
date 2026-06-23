package diff

import (
	"context"
	"fmt"
	"sort"

	"github.com/Ret2Hell/i18n-mcp/internal/locale"
)

type Service struct {
	locales *locale.Service
}

func NewService(locales *locale.Service) *Service {
	return &Service{locales: locales}
}

func (s *Service) Analyze(ctx context.Context) (Report, error) {
	inv, err := s.locales.Inventory(ctx)
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
			if _, ok := targetByKey[key]; ok {
				continue
			}
			report.Items = append(report.Items, KeyDiff{
				Locale:         localeCode,
				Namespace:      sourceUnit.Namespace,
				Key:            sourceUnit.Key,
				Status:         Missing,
				SourceValue:    sourceUnit.Value,
				SourceFilePath: sourceUnit.FilePath,
			})
		}
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

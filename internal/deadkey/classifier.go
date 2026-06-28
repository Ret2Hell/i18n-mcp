package deadkey

import (
	"cmp"
	"context"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/Ret2Hell/i18n-mcp/internal/config"
	"github.com/Ret2Hell/i18n-mcp/internal/locale"
	"github.com/Ret2Hell/i18n-mcp/internal/scanner"
)

func (s *Service) Report(ctx context.Context, in ReportInput) (Report, error) {
	cfg, err := s.config.Resolve(ctx)
	if err != nil {
		return Report{}, err
	}
	inv, err := s.locales.Inventory(ctx)
	if err != nil {
		return Report{}, err
	}
	usage, err := s.usageReport(ctx, in.RefreshUsage)
	if err != nil {
		return Report{}, err
	}

	usageByKey := buildUsageIndex(usage)
	dynamicHints := usage.DynamicHints
	filters := reportFilters(in)
	report := Report{SourceLocale: inv.SourceLocale, Usage: usage, GeneratedAt: time.Now().UTC()}
	for _, unit := range inv.Units {
		if unit.Locale != inv.SourceLocale {
			continue
		}
		if !filters.include(unit.Namespace, unit.Key) {
			continue
		}
		item := classifyUnit(cfg, unit, usageByKey, dynamicHints)
		if item.Status == StatusUsed && !in.IncludeUsed {
			continue
		}
		report.Items = append(report.Items, item)
	}
	sortItems(report.Items)
	report.Summary = summarize(report.Items)
	s.storeLatest(report)
	return report, nil
}

func (s *Service) usageReport(ctx context.Context, refresh bool) (scanner.Report, error) {
	if !refresh {
		if report, ok := s.scanner.Latest(); ok {
			return report, nil
		}
	}
	return s.scanner.Scan(ctx, scanner.ScanInput{})
}

type usageIndex map[string][]scanner.Evidence

func buildUsageIndex(report scanner.Report) usageIndex {
	index := usageIndex{}
	for _, usage := range report.Usages {
		for _, ev := range usage.Evidence {
			index[scanner.UsageIdentity(ev.Namespace, ev.Key)] = append(index[scanner.UsageIdentity(ev.Namespace, ev.Key)], ev)
			if ev.Namespace == "" {
				index[scanner.UsageIdentity("", ev.Key)] = append(index[scanner.UsageIdentity("", ev.Key)], ev)
			}
		}
	}
	return index
}

func classifyUnit(cfg config.Resolved, unit locale.Unit, usageByKey usageIndex, dynamicHints []scanner.DynamicHint) Item {
	fullKey := scanner.FullKey(unit.Namespace, unit.Key)
	item := Item{Namespace: unit.Namespace, Key: unit.Key, FullKey: fullKey, SourceValue: unit.Value, SourceFilePath: unit.FilePath}
	if matchesAnyPattern(cfg.KeptKeyPatterns, unit.Namespace, unit.Key) {
		item.Status = StatusKept
		item.Confidence = scanner.ConfidenceExact
		item.Reasons = append(item.Reasons, "matched keptKeyPatterns")
		return item
	}
	if matchesAnyPattern(cfg.IgnoredKeyPatterns, unit.Namespace, unit.Key) {
		item.Status = StatusIgnored
		item.Confidence = scanner.ConfidenceExact
		item.Reasons = append(item.Reasons, "matched ignoredKeyPatterns")
		return item
	}
	item.Evidence = matchingEvidence(usageByKey, unit.Namespace, unit.Key)
	if len(item.Evidence) > 0 {
		item.Status = StatusUsed
		item.Confidence = highestEvidenceConfidence(item.Evidence)
		return item
	}
	item.DynamicHints = matchingDynamicHints(cfg, dynamicHints, unit.Namespace, unit.Key)
	if len(item.DynamicHints) > 0 || matchesAnyPattern(cfg.DynamicKeyHints, unit.Namespace, unit.Key) {
		item.Status = StatusMaybeDynamic
		item.Confidence = scanner.ConfidenceLow
		item.Reasons = append(item.Reasons, "matched dynamic usage hint")
		return item
	}
	item.Status = StatusProbablyUnused
	item.Confidence = scanner.ConfidenceMedium
	item.Reasons = append(item.Reasons, "no static usage evidence or dynamic hint matched")
	return item
}

func matchingEvidence(index usageIndex, namespace string, key string) []scanner.Evidence {
	var out []scanner.Evidence
	out = append(out, index[scanner.UsageIdentity(namespace, key)]...)
	out = append(out, index[scanner.UsageIdentity("", key)]...)
	out = append(out, index[scanner.UsageIdentity("", scanner.FullKey(namespace, key))]...)
	slices.SortFunc(out, compareEvidence)
	return out
}

func matchingDynamicHints(cfg config.Resolved, hints []scanner.DynamicHint, namespace string, key string) []scanner.DynamicHint {
	var out []scanner.DynamicHint
	for _, hint := range hints {
		if hint.KeyPattern == "" {
			continue
		}
		if hint.KeyPattern == "*" || matchPattern(hint.KeyPattern, scanner.FullKey(namespace, key)) || matchPattern(hint.KeyPattern, key) {
			out = append(out, hint)
		}
	}
	for _, pattern := range cfg.DynamicKeyHints {
		if matchesPattern(pattern, namespace, key) {
			out = append(out, scanner.DynamicHint{KeyPattern: pattern, Pattern: "config.dynamicKeyHints", Confidence: scanner.ConfidenceMedium, Message: "configured dynamic key hint matched"})
		}
	}
	return out
}

func matchesAnyPattern(patterns []string, namespace string, key string) bool {
	for _, pattern := range patterns {
		if matchesPattern(pattern, namespace, key) {
			return true
		}
	}
	return false
}

func matchesPattern(pattern string, namespace string, key string) bool {
	return matchPattern(pattern, scanner.FullKey(namespace, key)) || matchPattern(pattern, key)
}

func matchPattern(pattern string, value string) bool {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return false
	}
	matched, err := path.Match(pattern, value)
	if err == nil && matched {
		return true
	}
	return pattern == value
}

func highestEvidenceConfidence(evidence []scanner.Evidence) scanner.Confidence {
	best := scanner.ConfidenceLow
	for _, ev := range evidence {
		switch ev.Confidence {
		case scanner.ConfidenceExact:
			return scanner.ConfidenceExact
		case scanner.ConfidenceHigh:
			best = scanner.ConfidenceHigh
		case scanner.ConfidenceMedium:
			if best == scanner.ConfidenceLow {
				best = scanner.ConfidenceMedium
			}
		}
	}
	return best
}

type filters struct {
	namespaces map[string]bool
	keys       map[string]bool
}

func reportFilters(in ReportInput) filters {
	return filters{namespaces: stringSet(in.Namespaces), keys: stringSet(in.Keys)}
}

func (f filters) include(namespace string, key string) bool {
	if len(f.namespaces) > 0 && !f.namespaces[namespace] {
		return false
	}
	if len(f.keys) > 0 && !f.keys[key] && !f.keys[scanner.FullKey(namespace, key)] {
		return false
	}
	return true
}

func stringSet(values []string) map[string]bool {
	set := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			set[value] = true
		}
	}
	return set
}

func sortItems(items []Item) {
	slices.SortFunc(items, func(a, b Item) int {
		return cmp.Or(cmp.Compare(a.Namespace, b.Namespace), cmp.Compare(a.Key, b.Key))
	})
}

func compareEvidence(a scanner.Evidence, b scanner.Evidence) int {
	return cmp.Or(
		cmp.Compare(a.FilePath, b.FilePath),
		cmp.Compare(a.Line, b.Line),
		cmp.Compare(a.Column, b.Column),
		cmp.Compare(a.Pattern, b.Pattern),
	)
}

func summarize(items []Item) Summary {
	var summary Summary
	for _, item := range items {
		summary.Total++
		switch item.Status {
		case StatusUsed:
			summary.Used++
		case StatusProbablyUnused:
			summary.ProbablyUnused++
		case StatusMaybeDynamic:
			summary.MaybeDynamic++
		case StatusIgnored:
			summary.Ignored++
		case StatusKept:
			summary.Kept++
		}
	}
	return summary
}

func cloneReport(report Report) Report {
	report.Items = slices.Clone(report.Items)
	report.Warnings = slices.Clone(report.Warnings)
	report.Usage.Files = slices.Clone(report.Usage.Files)
	report.Usage.Usages = slices.Clone(report.Usage.Usages)
	report.Usage.DynamicHints = slices.Clone(report.Usage.DynamicHints)
	report.Usage.Warnings = slices.Clone(report.Usage.Warnings)
	for i := range report.Items {
		report.Items[i].Evidence = slices.Clone(report.Items[i].Evidence)
		report.Items[i].DynamicHints = slices.Clone(report.Items[i].DynamicHints)
		report.Items[i].Reasons = slices.Clone(report.Items[i].Reasons)
	}
	for i := range report.Usage.Usages {
		report.Usage.Usages[i].Evidence = slices.Clone(report.Usage.Usages[i].Evidence)
	}
	return report
}

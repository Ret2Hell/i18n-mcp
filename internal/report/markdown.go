package report

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/Ret2Hell/i18n-mcp/internal/deadkey"
	"github.com/Ret2Hell/i18n-mcp/internal/diff"
)

func RenderMarkdown(report Report) (string, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "# i18n Audit Report\n\n")
	fmt.Fprintf(&b, "Generated: %s\n\n", report.GeneratedAt.Format("2006-01-02T15:04:05Z07:00"))
	renderSummary(&b, report)
	renderInventory(&b, report)
	renderDiffSection(&b, "Missing Keys", report.Diff.Items, diff.Missing)
	renderDiffSection(&b, "Stale Keys", report.Diff.Items, diff.Stale)
	renderDiffSection(&b, "Invalid Translations", report.Diff.Items, diff.Invalid)
	renderDiffSection(&b, "Extra Keys", report.Diff.Items, diff.Extra)
	renderDeadKeys(&b, report)
	renderChangedFiles(&b)
	renderWarnings(&b, report.Warnings)
	return b.String(), nil
}

func renderSummary(b *strings.Builder, report Report) {
	fmt.Fprintf(b, "## Summary\n\n")
	fmt.Fprintf(b, "| Metric | Count |\n")
	fmt.Fprintf(b, "| --- | ---: |\n")
	fmt.Fprintf(b, "| Locales | %d |\n", report.Summary.Locales)
	fmt.Fprintf(b, "| Namespaces | %d |\n", report.Summary.Namespaces)
	fmt.Fprintf(b, "| Source keys | %d |\n", report.Summary.SourceKeys)
	fmt.Fprintf(b, "| Missing | %d |\n", report.Summary.Missing)
	fmt.Fprintf(b, "| Stale | %d |\n", report.Summary.Stale)
	fmt.Fprintf(b, "| Invalid | %d |\n", report.Summary.Invalid)
	fmt.Fprintf(b, "| Extra | %d |\n", report.Summary.Extra)
	fmt.Fprintf(b, "| Probably unused | %d |\n", report.Summary.ProbablyUnused)
	fmt.Fprintf(b, "| Maybe dynamic | %d |\n", report.Summary.MaybeDynamic)
	fmt.Fprintf(b, "| Warnings | %d |\n\n", report.Summary.Warnings)
}

func renderInventory(b *strings.Builder, report Report) {
	fmt.Fprintf(b, "## Locale Inventory\n\n")
	fmt.Fprintf(b, "Source locale: `%s`\n\n", report.Inventory.SourceLocale)
	fmt.Fprintf(b, "Target locales: `%s`\n\n", strings.Join(report.Inventory.TargetLocales, "`, `"))
	fmt.Fprintf(b, "| Locale | Keys |\n")
	fmt.Fprintf(b, "| --- | ---: |\n")
	for _, localeCode := range slices.Sorted(maps.Keys(report.Inventory.CountsByLocale)) {
		fmt.Fprintf(b, "| `%s` | %d |\n", localeCode, report.Inventory.CountsByLocale[localeCode])
	}
	fmt.Fprintf(b, "\n")
}

func renderDiffSection(b *strings.Builder, title string, items []diff.KeyDiff, status diff.KeyStatus) {
	fmt.Fprintf(b, "## %s\n\n", title)
	filtered := filterDiffItems(items, status)
	if len(filtered) == 0 {
		fmt.Fprintf(b, "None.\n\n")
		return
	}
	fmt.Fprintf(b, "| Locale | Namespace | Key | Detail |\n")
	fmt.Fprintf(b, "| --- | --- | --- | --- |\n")
	for _, item := range filtered {
		detail := item.TargetValue
		if detail == "" {
			detail = item.SourceValue
		}
		fmt.Fprintf(b, "| `%s` | `%s` | `%s` | %s |\n", item.Locale, item.Namespace, item.Key, markdownCell(truncate(detail, 80)))
	}
	fmt.Fprintf(b, "\n")
}

func filterDiffItems(items []diff.KeyDiff, status diff.KeyStatus) []diff.KeyDiff {
	var out []diff.KeyDiff
	for _, item := range items {
		if item.Status == status {
			out = append(out, item)
		}
	}
	slices.SortFunc(out, func(a, b diff.KeyDiff) int {
		return cmp.Or(
			cmp.Compare(a.Locale, b.Locale),
			cmp.Compare(a.Namespace, b.Namespace),
			cmp.Compare(a.Key, b.Key),
		)
	})
	return out
}

func renderDeadKeys(b *strings.Builder, report Report) {
	fmt.Fprintf(b, "## Dead-Key Candidates\n\n")
	items := filterDeadKeys(report.DeadKeys.Items)
	if len(items) == 0 {
		fmt.Fprintf(b, "None.\n\n")
		return
	}
	fmt.Fprintf(b, "| Namespace | Key | Status | Confidence | Reasons |\n")
	fmt.Fprintf(b, "| --- | --- | --- | --- | --- |\n")
	for _, item := range items {
		fmt.Fprintf(b, "| `%s` | `%s` | `%s` | `%s` | %s |\n", item.Namespace, item.Key, item.Status, item.Confidence, markdownCell(strings.Join(item.Reasons, "; ")))
	}
	fmt.Fprintf(b, "\n")
}

func filterDeadKeys(items []deadkey.Item) []deadkey.Item {
	var out []deadkey.Item
	for _, item := range items {
		if item.Status == deadkey.StatusProbablyUnused || item.Status == deadkey.StatusMaybeDynamic {
			out = append(out, item)
		}
	}
	slices.SortFunc(out, func(a, b deadkey.Item) int {
		return cmp.Or(
			cmp.Compare(a.Namespace, b.Namespace),
			cmp.Compare(a.Key, b.Key),
			cmp.Compare(a.Status, b.Status),
		)
	})
	return out
}

func renderChangedFiles(b *strings.Builder) {
	fmt.Fprintf(b, "## Changed Files\n\n")
	fmt.Fprintf(b, "No file changes are produced by report generation.\n\n")
}

func renderWarnings(b *strings.Builder, warnings []string) {
	fmt.Fprintf(b, "## Warnings\n\n")
	if len(warnings) == 0 {
		fmt.Fprintf(b, "None.\n")
		return
	}
	for _, warning := range warnings {
		fmt.Fprintf(b, "- %s\n", warning)
	}
}

func markdownCell(value string) string {
	value = strings.ReplaceAll(value, "|", "\\|")
	value = strings.ReplaceAll(value, "\n", " ")
	if value == "" {
		return ""
	}
	return value
}

func truncate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	if limit <= 3 {
		return value[:limit]
	}
	return value[:limit-3] + "..."
}

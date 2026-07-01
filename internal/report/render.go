package report

import (
	"bytes"
	"encoding/json"
	"fmt"
)

func RenderJSON(report Report) (string, error) {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func RenderMarkdown(report Report) (string, error) {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "# i18n Audit Report\n\n")
	fmt.Fprintf(&buf, "Generated: %s\n\n", report.GeneratedAt.Format("2006-01-02T15:04:05Z07:00"))
	fmt.Fprintf(&buf, "Project root: `%s`\n\n", report.ProjectRoot)
	fmt.Fprintf(&buf, "## Summary\n\n")
	fmt.Fprintf(&buf, "| Metric | Count |\n| --- | ---: |\n")
	fmt.Fprintf(&buf, "| Locales | %d |\n", report.Summary.Locales)
	fmt.Fprintf(&buf, "| Namespaces | %d |\n", report.Summary.Namespaces)
	fmt.Fprintf(&buf, "| Source keys | %d |\n", report.Summary.SourceKeys)
	fmt.Fprintf(&buf, "| Target keys | %d |\n", report.Summary.TargetKeys)
	fmt.Fprintf(&buf, "| Missing | %d |\n", report.Summary.Missing)
	fmt.Fprintf(&buf, "| Stale | %d |\n", report.Summary.Stale)
	fmt.Fprintf(&buf, "| Invalid | %d |\n", report.Summary.Invalid)
	fmt.Fprintf(&buf, "| Extra | %d |\n", report.Summary.Extra)
	fmt.Fprintf(&buf, "| Unknown | %d |\n", report.Summary.Unknown)
	fmt.Fprintf(&buf, "| Probably unused | %d |\n", report.Summary.ProbablyUnused)
	fmt.Fprintf(&buf, "| Maybe dynamic | %d |\n", report.Summary.MaybeDynamic)
	fmt.Fprintf(&buf, "| Warnings | %d |\n\n", report.Summary.Warnings)
	if len(report.Warnings) > 0 {
		fmt.Fprintf(&buf, "## Warnings\n\n")
		for _, warning := range report.Warnings {
			fmt.Fprintf(&buf, "- %s\n", warning)
		}
		fmt.Fprintf(&buf, "\n")
	}
	fmt.Fprintf(&buf, "## Inventory\n\n")
	fmt.Fprintf(&buf, "- Source locale: `%s`\n", report.Inventory.SourceLocale)
	fmt.Fprintf(&buf, "- Target locales: %d\n", len(report.Inventory.TargetLocales))
	fmt.Fprintf(&buf, "- Locale files: %d\n\n", len(report.Inventory.Files))
	fmt.Fprintf(&buf, "## Analysis\n\n")
	fmt.Fprintf(&buf, "- Diff items: %d\n", len(report.Diff.Items))
	fmt.Fprintf(&buf, "- Usage entries: %d\n", len(report.Usage.Usages))
	fmt.Fprintf(&buf, "- Dead-key items: %d\n", len(report.DeadKeys.Items))
	return buf.String(), nil
}

package scanner

import "slices"

func cloneReport(report Report) Report {
	report.Files = slices.Clone(report.Files)
	report.Usages = slices.Clone(report.Usages)
	report.DynamicHints = slices.Clone(report.DynamicHints)
	report.Warnings = slices.Clone(report.Warnings)
	for i := range report.Usages {
		report.Usages[i].Evidence = slices.Clone(report.Usages[i].Evidence)
	}
	return report
}

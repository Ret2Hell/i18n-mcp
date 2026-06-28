package scanner

func cloneReport(report Report) Report {
	report.Files = append([]SourceFile(nil), report.Files...)
	report.Usages = append([]Usage(nil), report.Usages...)
	report.DynamicHints = append([]DynamicHint(nil), report.DynamicHints...)
	report.Warnings = append([]string(nil), report.Warnings...)
	for i := range report.Usages {
		report.Usages[i].Evidence = append([]Evidence(nil), report.Usages[i].Evidence...)
	}
	return report
}

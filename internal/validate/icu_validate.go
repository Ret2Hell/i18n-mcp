package validate

func compareICUArguments(source string, target string) []Issue {
	if !LooksLikeICU(source) && !LooksLikeICU(target) {
		return nil
	}
	issues := compareTokenCounts(
		icuArgumentCounts(source),
		icuArgumentCounts(target),
		"ICU argument",
		"icu_argument_missing",
		"icu_argument_extra",
		"icu_argument_count_changed",
		SeverityError,
	)
	for i := range issues {
		if issues[i].Code == "icu_argument_extra" {
			issues[i].Severity = SeverityWarning
		}
	}
	return issues
}

func icuArgumentCounts(s string) map[string]int {
	counts := map[string]int{}
	for _, arg := range ExtractICUArguments(s) {
		counts[arg]++
	}
	return counts
}

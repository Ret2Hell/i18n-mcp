package validate

import (
	"fmt"
	"sort"
)

func comparePlaceholders(source string, target string) []Issue {
	return compareTokenCounts(
		placeholderCounts(source),
		placeholderCounts(target),
		"placeholder",
		"placeholder_missing",
		"placeholder_extra",
		"placeholder_count_changed",
		SeverityError,
	)
}

func compareTokenCounts(source map[string]int, target map[string]int, label string, missingCode string, extraCode string, countChangedCode string, severity Severity) []Issue {
	keys := make([]string, 0, len(source)+len(target))
	seen := map[string]bool{}
	for key := range source {
		seen[key] = true
		keys = append(keys, key)
	}
	for key := range target {
		if seen[key] {
			continue
		}
		seen[key] = true
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var issues []Issue
	for _, key := range keys {
		sourceCount := source[key]
		targetCount := target[key]
		switch {
		case sourceCount > 0 && targetCount == 0:
			issues = append(issues, Issue{
				Code:     missingCode,
				Message:  fmt.Sprintf("target is missing %s %q", label, key),
				Severity: severity,
			})
		case sourceCount == 0 && targetCount > 0:
			issues = append(issues, Issue{
				Code:     extraCode,
				Message:  fmt.Sprintf("target has extra %s %q", label, key),
				Severity: severity,
			})
		case sourceCount != targetCount:
			issues = append(issues, Issue{
				Code:     countChangedCode,
				Message:  fmt.Sprintf("target changes %s %q count from %d to %d", label, key, sourceCount, targetCount),
				Severity: severity,
			})
		}
	}
	return issues
}

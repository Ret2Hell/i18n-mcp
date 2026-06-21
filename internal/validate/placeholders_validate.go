package validate

import (
	"fmt"
	"maps"
	"slices"
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
	keys := make(map[string]int, len(source)+len(target))
	maps.Copy(keys, source)
	maps.Copy(keys, target)

	var issues []Issue
	for _, key := range slices.Sorted(maps.Keys(keys)) {
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

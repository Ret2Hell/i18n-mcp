package validate

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
)

var icuHeadPattern = regexp.MustCompile(`\{\s*([A-Za-z_][A-Za-z0-9_]*)\s*,\s*(plural|select|selectordinal|number|date|time)\b`)

func LooksLikeICU(s string) bool {
	return icuHeadPattern.MatchString(s)
}

func ExtractICUArguments(s string) []string {
	matches := icuHeadPattern.FindAllStringSubmatch(s, -1)
	seen := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		seen[match[1]] = struct{}{}
	}
	return slices.Sorted(maps.Keys(seen))
}

func HasBalancedBraces(s string) bool {
	depth := 0
	for _, r := range s {
		switch r {
		case '{':
			depth++
		case '}':
			depth--
			if depth < 0 {
				return false
			}
		}
	}
	return depth == 0
}

func validateICUShape(source string, target string) []Issue {
	sourceICU := LooksLikeICU(source)
	targetICU := LooksLikeICU(target)
	var issues []Issue

	if sourceICU && !targetICU {
		issues = append(issues, Issue{
			Code:     "icu_missing",
			Message:  "source uses ICU formatting but target does not look like ICU",
			Severity: SeverityError,
		})
	}
	if !sourceICU && targetICU {
		issues = append(issues, Issue{
			Code:     "icu_extra",
			Message:  "target looks like ICU but source does not",
			Severity: SeverityWarning,
		})
	}
	if sourceICU && !HasBalancedBraces(source) {
		issues = append(issues, Issue{
			Code:     "icu_source_unbalanced_braces",
			Message:  "source ICU message has unbalanced braces",
			Severity: SeverityError,
		})
	}
	if targetICU && !HasBalancedBraces(target) {
		issues = append(issues, Issue{
			Code:     "icu_target_unbalanced_braces",
			Message:  "target ICU message has unbalanced braces",
			Severity: SeverityError,
		})
	}
	if sourceICU && targetICU && len(ExtractICUArguments(source)) == 0 {
		issues = append(issues, Issue{
			Code:     "icu_no_arguments",
			Message:  fmt.Sprintf("ICU source was detected but no argument names were extracted from %q", source),
			Severity: SeverityWarning,
		})
	}
	return issues
}

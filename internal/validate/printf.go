package validate

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Supported portable printf conversions, positional arguments and dynamic width/
// precision. Escaped percents are consumed left-to-right, not guessed from prose.
var printfPattern = regexp.MustCompile(`^%([0-9]+\$)?[-+#0 ']*(?:[0-9]+|(\*)([0-9]+\$)?)?(?:\.(?:[0-9]+|(\*)([0-9]+\$)?))?(hh|ll|[hlLjzt])?([sdiuoxXfFeEgGc])`)

type printfToken struct{ raw, position, width, widthPosition, precision, precisionPosition, length, conversion string }

func printfTokens(s string) ([]printfToken, error) {
	var out []printfToken
	for i := 0; i < len(s); i++ {
		if s[i] != '%' {
			continue
		}
		if i+1 < len(s) && s[i+1] == '%' {
			i++
			continue
		}
		match := printfPattern.FindStringSubmatch(s[i:])
		if match == nil {
			return nil, fmt.Errorf("invalid or unsupported printf conversion at byte %d; literal percent signs must be escaped as %%%%", i)
		}
		for _, position := range []string{match[1], match[3], match[5]} {
			if position != "" {
				index, err := strconv.Atoi(strings.TrimSuffix(position, "$"))
				if err != nil || index < 1 {
					return nil, fmt.Errorf("invalid printf argument position %q", position)
				}
			}
		}
		modifier, conversion := match[6], match[7]
		if (strings.ContainsAny(conversion, "sc") && modifier != "" && modifier != "l") || (strings.ContainsAny(conversion, "diuoxX") && modifier == "L") || (strings.ContainsAny(conversion, "fFeEgG") && modifier != "" && modifier != "l" && modifier != "L") {
			return nil, fmt.Errorf("unsupported printf length modifier %q for %s", modifier, conversion)
		}
		out = append(out, printfToken{match[0], match[1], match[2], match[3], match[4], match[5], match[6], match[7]})
		i += len(match[0]) - 1
	}
	return out, nil
}

func printfCounts(tokens []printfToken) map[string]int {
	counts := map[string]int{}
	next := 1
	add := func(position, kind string) {
		index := next
		if position != "" {
			parsed, err := strconv.Atoi(strings.TrimSuffix(position, "$"))
			if err == nil {
				index = parsed
			}
		}
		counts[fmt.Sprintf("%%%d$%s", index, kind)]++
		next = index + 1
	}
	for _, token := range tokens {
		if token.width != "" {
			add(token.widthPosition, ":integer")
		}
		if token.precision != "" {
			add(token.precisionPosition, ":integer")
		}
		kind := "integer"
		switch token.conversion {
		case "s":
			kind = "string"
		case "u", "o", "x", "X":
			kind = "unsigned"
		case "c":
			kind = "character"
		case "f", "F", "e", "E", "g", "G":
			kind = "float"
		}
		add(token.position, token.length+":"+kind)
	}
	return counts
}

func comparePrintf(source, target string) []Issue {
	sourceTokens, sourceErr := printfTokens(source)
	targetTokens, targetErr := printfTokens(target)
	var issues []Issue
	if sourceErr != nil {
		issues = append(issues, Issue{Code: "printf_source_invalid", Message: sourceErr.Error(), Severity: SeverityError})
	}
	if targetErr != nil {
		issues = append(issues, Issue{Code: "printf_target_invalid", Message: targetErr.Error(), Severity: SeverityError})
	}
	if len(issues) > 0 {
		return issues
	}
	return compareTokenCounts(printfCounts(sourceTokens), printfCounts(targetTokens), "printf argument", "printf_argument_missing", "printf_argument_extra", "printf_argument_count_changed", SeverityError)
}

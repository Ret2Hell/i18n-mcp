package validate

import "fmt"

func compareTags(source string, target string) []Issue {
	issues := compareTokenCounts(
		tagCounts(ExtractTagTokens(source)),
		tagCounts(ExtractTagTokens(target)),
		"tag",
		"tag_missing",
		"tag_extra",
		"tag_count_changed",
		SeverityError,
	)
	issues = append(issues, validateTagBalance("source", ExtractTagTokens(source))...)
	issues = append(issues, validateTagBalance("target", ExtractTagTokens(target))...)
	return issues
}

func tagCounts(tokens []TagToken) map[string]int {
	counts := map[string]int{}
	for _, token := range tokens {
		counts[token.Normalized]++
	}
	return counts
}

func validateTagBalance(label string, tokens []TagToken) []Issue {
	var issues []Issue
	var stack []TagToken
	for _, token := range tokens {
		if token.SelfClosing {
			continue
		}
		if !token.Closing {
			stack = append(stack, token)
			continue
		}
		if len(stack) == 0 {
			issues = append(issues, Issue{
				Code:     "tag_unopened",
				Message:  fmt.Sprintf("%s has closing tag %s without a matching opening tag", label, token.Normalized),
				Severity: SeverityError,
			})
			continue
		}
		top := stack[len(stack)-1]
		if top.Name != token.Name {
			issues = append(issues, Issue{
				Code:     "tag_mismatched",
				Message:  fmt.Sprintf("%s closes %s while %s is still open", label, token.Normalized, top.Normalized),
				Severity: SeverityError,
			})
			continue
		}
		stack = stack[:len(stack)-1]
	}
	for i := len(stack) - 1; i >= 0; i-- {
		issues = append(issues, Issue{
			Code:     "tag_unclosed",
			Message:  fmt.Sprintf("%s has unclosed tag %s", label, stack[i].Normalized),
			Severity: SeverityError,
		})
	}
	return issues
}

package validate

import "strings"

func (s *Service) ValidatePair(pair Pair) Result {
	issues := validatePairIssues(pair)
	return classify(pair, issues)
}

func (s *Service) ValidateStrings(source string, target string) Result {
	return s.ValidatePair(Pair{Source: source, Target: target})
}

func validatePairIssues(pair Pair) []Issue {
	var issues []Issue
	if pair.Source != "" && strings.TrimSpace(pair.Target) == "" {
		issues = append(issues, Issue{
			Code:     "target_empty",
			Message:  "target translation is empty while source is not empty",
			Severity: SeverityError,
		})
	}
	if pair.Source != "" && pair.Target == pair.Source && pair.Locale != "" && pair.SourceLocale != "" && pair.Locale != pair.SourceLocale {
		issues = append(issues, Issue{
			Code:     "target_untranslated",
			Message:  "target translation is identical to source",
			Severity: SeverityWarning,
		})
	}

	issues = append(issues, comparePlaceholders(pair.Source, pair.Target)...)
	issues = append(issues, compareTags(pair.Source, pair.Target)...)
	issues = append(issues, validateICUShape(pair.Source, pair.Target)...)
	issues = append(issues, compareICUArguments(pair.Source, pair.Target)...)
	issues = append(issues, compareMarkdownLinks(pair.Source, pair.Target)...)
	return issues
}

func classify(pair Pair, issues []Issue) Result {
	var result Result
	for _, issue := range issues {
		issue.Locale = pair.Locale
		issue.Namespace = pair.Namespace
		issue.Key = pair.Key
		switch issue.Severity {
		case SeverityWarning:
			result.Warnings = append(result.Warnings, issue)
		default:
			issue.Severity = SeverityError
			result.Issues = append(result.Issues, issue)
		}
	}
	result.OK = len(result.Issues) == 0
	return result
}

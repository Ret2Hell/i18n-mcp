package validate

import (
	"context"
	"slices"
	"strings"
)

// ValidatePair validates a source and target translation pair.
func (s *Service) ValidatePair(pair Pair) Result {
	return s.ValidatePairContext(context.Background(), pair)
}

// ValidatePairContext validates a pair with cancellation-aware ICU parsing.
func (s *Service) ValidatePairContext(ctx context.Context, pair Pair) Result {
	issues := validatePairIssues(ctx, pair)
	return classify(pair, issues)
}

// ValidateStrings validates source and target translation strings.
func (s *Service) ValidateStrings(source string, target string) Result {
	return s.ValidatePair(Pair{Source: source, Target: target})
}

func validatePairIssues(ctx context.Context, pair Pair) []Issue {
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

	var messageIssues []Issue
	if LooksLikeICU(pair.Source) || LooksLikeICU(pair.Target) {
		messageIssues = compareICUMessages(ctx, pair.Source, pair.Target)
	} else {
		messageIssues = comparePlaceholders(pair.Source, pair.Target)
	}
	return slices.Concat(
		issues,
		messageIssues,
		compareTags(pair.Source, pair.Target),
		compareMarkdownLinks(pair.Source, pair.Target),
	)
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

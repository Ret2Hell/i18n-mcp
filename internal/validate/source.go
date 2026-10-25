package validate

import "context"

// CheckSource checks the selected source syntax without attributing failures to
// a target locale. Raw is an explicit opt-out, not a failed-parser fallback.
func CheckSource(ctx context.Context, value string, options Options) []Issue {
	if options.MessageFormat == "raw" {
		return nil
	}
	if _, err := ExtractPlaceholdersFor(ctx, value, options); err != nil {
		return []Issue{{Scope: "source", Code: "source_format_invalid", Message: err.Error(), Severity: SeverityError}}
	}
	issues := validateTagBalance("source", ExtractTagTokens(value))
	for i := range issues {
		issues[i].Scope = "source"
	}
	return issues
}

package validate

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractPlaceholders(t *testing.T) {
	got := ExtractPlaceholders("Hi {name}, {{ user }}, %{count}, %02d, %%s")
	require.Equal(t, []string{"%02d", "%{count}", "{name}", "{{user}}"}, got)
}

func TestPlaceholderParityValidation(t *testing.T) {
	result := NewService().ValidateStrings("Hello {name}, you have %{count}", "Bonjour {name}")

	require.False(t, result.OK)
	requireIssueCode(t, result.Issues, "placeholder_missing")
}

func TestPlaceholderCountChanged(t *testing.T) {
	result := NewService().ValidateStrings("{name} invited {name}", "{name} a invite")

	require.False(t, result.OK)
	requireIssueCode(t, result.Issues, "placeholder_count_changed")
}

func TestExtractTags(t *testing.T) {
	got := ExtractTags("Click <Link href=\"/x\"><strong>here</strong></Link><br />")
	require.Equal(t, []string{"</link>", "</strong>", "<br/>", "<link>", "<strong>"}, got)
}

func TestTagParityValidation(t *testing.T) {
	result := NewService().ValidateStrings("Click <strong>here</strong>", "Cliquez ici")

	require.False(t, result.OK)
	requireIssueCode(t, result.Issues, "tag_missing")
}

func TestTagMismatchValidation(t *testing.T) {
	result := NewService().ValidateStrings("Click <strong>here</strong>", "Cliquez <strong>ici</em>")

	require.False(t, result.OK)
	requireIssueCode(t, result.Issues, "tag_mismatched")
}

func TestMarkdownLinkValidationWarns(t *testing.T) {
	result := NewService().ValidateStrings("Read [docs](/docs)", "Lire la documentation")

	require.True(t, result.OK)
	requireIssueCode(t, result.Warnings, "markdown_link_count_changed")
}

func TestICUShapeValidation(t *testing.T) {
	result := NewService().ValidateStrings(
		"{count, plural, one {# item} other {# items}}",
		"{count, plural, one {# element} other {# elements}",
	)

	require.False(t, result.OK)
	requireIssueCode(t, result.Issues, "icu_target_unbalanced_braces")
}

func TestICUArgumentParityValidation(t *testing.T) {
	result := NewService().ValidateStrings(
		"{count, plural, one {# item} other {# items}}",
		"{total, plural, one {# element} other {# elements}}",
	)

	require.False(t, result.OK)
	requireIssueCode(t, result.Issues, "icu_argument_missing")
	requireIssueCode(t, result.Warnings, "icu_argument_extra")
}

func TestEmptyTargetIsBlocking(t *testing.T) {
	result := NewService().ValidateStrings("Save", "")

	require.False(t, result.OK)
	requireIssueCode(t, result.Issues, "target_empty")
}

func TestUntranslatedTargetIsWarning(t *testing.T) {
	result := NewService().ValidatePair(Pair{
		SourceLocale: "en",
		Locale:       "fr",
		Namespace:    "common",
		Key:          "save",
		Source:       "Save",
		Target:       "Save",
	})

	require.True(t, result.OK)
	requireIssueCode(t, result.Warnings, "target_untranslated")
	require.Equal(t, "fr", result.Warnings[0].Locale)
	require.Equal(t, "common", result.Warnings[0].Namespace)
	require.Equal(t, "save", result.Warnings[0].Key)
}

func TestValidPair(t *testing.T) {
	result := NewService().ValidateStrings(
		"Hello {name}, read <link>[docs](/docs)</link>",
		"Bonjour {name}, lisez <link>[docs](/docs)</link>",
	)

	require.True(t, result.OK)
	require.Empty(t, result.Issues)
	require.Empty(t, result.Warnings)
}

func requireIssueCode(t *testing.T, issues []Issue, code string) {
	t.Helper()
	for _, issue := range issues {
		if issue.Code == code {
			return
		}
	}
	require.Failf(t, "missing issue", "expected issue code %q in %#v", code, issues)
}

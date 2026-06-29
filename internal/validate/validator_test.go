package validate

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractPlaceholders(t *testing.T) {
	got := ExtractPlaceholders("Hi {name}, {{ user }}, %{count}, %02d, %%s")
	require.Equal(t, []string{"%02d", "%{count}", "{name}", "{{user}}"}, got)
}

func TestCollectPlaceholderSpansRecordsExactOffsets(t *testing.T) {
	spans := collectPlaceholderSpans("x {name} %{count}")

	require.Equal(t, []placeholderSpan{
		{start: 2, end: 8, value: "{name}"},
		{start: 9, end: 17, value: "%{count}"},
	}, spans)
}

func TestExtractPlaceholdersBoundariesAndOverlaps(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{
			name: "escaped percent placeholder ignored",
			in:   "Discount %%{count} and real %{total}",
			want: []string{"%{total}"},
		},
		{
			name: "escaped percent placeholder ignored at start",
			in:   "%%{count} and real %{total}",
			want: []string{"%{total}"},
		},
		{
			name: "percent placeholder masks brace placeholder",
			in:   "%{count} {count}",
			want: []string{"%{count}", "{count}"},
		},
		{
			name: "printf placeholder beside brace placeholder",
			in:   "%02d{name}",
			want: []string{"%02d", "{name}"},
		},
		{
			name: "double brace normalized and not also single brace",
			in:   "{{ user }}",
			want: []string{"{{user}}"},
		},
		{
			name: "duplicates deduplicated",
			in:   "{name} {name} %{count} %{count}",
			want: []string{"%{count}", "{name}"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, ExtractPlaceholders(tt.in))
		})
	}
}

func TestExtractTags(t *testing.T) {
	got := ExtractTags("Click <Link href=\"/x\"><strong>here</strong></Link><br />")
	require.Equal(t, []string{"</link>", "</strong>", "<br/>", "<link>", "<strong>"}, got)
}

func TestExtractMarkdownLinks(t *testing.T) {
	links := ExtractMarkdownLinks("Read [docs](/docs) and [API](https://example.test/api).")

	require.Equal(t, []MarkdownLink{
		{Raw: "[docs](/docs)", Text: "docs", Destination: "/docs"},
		{Raw: "[API](https://example.test/api)", Text: "API", Destination: "https://example.test/api"},
	}, links)
}

func TestExtractMarkdownLinksIgnoresMalformedLinks(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{name: "empty destination", in: "Read [docs]()"},
		{name: "empty text", in: "Read [](/docs)"},
		{name: "newline in text", in: "Read [do\ncs](/docs)"},
		{name: "newline in destination", in: "Read [docs](/do\ncs)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Empty(t, ExtractMarkdownLinks(tt.in))
		})
	}
}

func TestValidationIssueAndWarningCodes(t *testing.T) {
	tests := []struct {
		name         string
		source       string
		target       string
		wantOK       bool
		issueCodes   []string
		warningCodes []string
	}{
		{
			name:       "placeholder missing",
			source:     "Hello {name}, you have %{count}",
			target:     "Bonjour {name}",
			issueCodes: []string{"placeholder_missing"},
		},
		{
			name:       "placeholder count changed",
			source:     "{name} invited {name}",
			target:     "{name} a invite",
			issueCodes: []string{"placeholder_count_changed"},
		},
		{
			name:       "tag missing",
			source:     "Click <strong>here</strong>",
			target:     "Cliquez ici",
			issueCodes: []string{"tag_missing"},
		},
		{
			name:       "tag mismatched",
			source:     "Click <strong>here</strong>",
			target:     "Cliquez <strong>ici</em>",
			issueCodes: []string{"tag_mismatched"},
		},
		{
			name:         "markdown link warning",
			source:       "Read [docs](/docs)",
			target:       "Lire la documentation",
			wantOK:       true,
			warningCodes: []string{"markdown_link_count_changed"},
		},
		{
			name:       "ICU target unbalanced braces",
			source:     "{count, plural, one {# item} other {# items}}",
			target:     "{count, plural, one {# element} other {# elements}",
			issueCodes: []string{"icu_target_unbalanced_braces"},
		},
		{
			name:         "ICU argument parity",
			source:       "{count, plural, one {# item} other {# items}}",
			target:       "{total, plural, one {# element} other {# elements}}",
			issueCodes:   []string{"icu_argument_missing"},
			warningCodes: []string{"icu_argument_extra"},
		},
		{
			name:       "only source looks like ICU",
			source:     "{count, plural, one {# item} other {# items}}",
			target:     "Aucun element",
			issueCodes: []string{"icu_argument_missing"},
		},
		{
			name:         "only target looks like ICU",
			source:       "No items",
			target:       "{count, plural, one {# element} other {# elements}}",
			wantOK:       true,
			warningCodes: []string{"icu_argument_extra"},
		},
		{
			name:       "empty target",
			source:     "Save",
			target:     "",
			issueCodes: []string{"target_empty"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewService().ValidateStrings(tt.source, tt.target)

			require.Equal(t, tt.wantOK, result.OK)
			for _, code := range tt.issueCodes {
				requireIssueCode(t, result.Issues, code)
			}
			for _, code := range tt.warningCodes {
				requireIssueCode(t, result.Warnings, code)
			}
		})
	}
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
	if slices.ContainsFunc(issues, func(issue Issue) bool { return issue.Code == code }) {
		return
	}
	require.Failf(t, "missing issue", "expected issue code %q in %#v", code, issues)
}

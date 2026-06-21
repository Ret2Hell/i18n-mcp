package validate

import (
	"regexp"
	"sort"
	"strings"
)

type placeholderPattern struct {
	re        *regexp.Regexp
	normalize func(string) string
}

type placeholderSpan struct {
	start int
	end   int
	value string
}

var placeholderPatterns = []placeholderPattern{
	{
		re:        regexp.MustCompile(`\{\{\s*[A-Za-z_][A-Za-z0-9_]*\s*\}\}`),
		normalize: normalizeDoubleBracePlaceholder,
	},
	{
		re:        regexp.MustCompile(`%\{[A-Za-z_][A-Za-z0-9_]*\}`),
		normalize: identityPlaceholder,
	},
	{
		re:        regexp.MustCompile(`%([0-9]+\$)?[-+#0 ]*(\*|[0-9]+)?(\.(\*|[0-9]+))?[sdif]`),
		normalize: identityPlaceholder,
	},
	{
		re:        regexp.MustCompile(`\{[A-Za-z_][A-Za-z0-9_]*\}`),
		normalize: identityPlaceholder,
	},
}

func ExtractPlaceholders(s string) []string {
	seen := map[string]bool{}
	var out []string
	for _, span := range collectPlaceholderSpans(s) {
		if seen[span.value] {
			continue
		}
		seen[span.value] = true
		out = append(out, span.value)
	}
	sort.Strings(out)
	return out
}

func collectPlaceholderSpans(s string) []placeholderSpan {
	var spans []placeholderSpan
	occupied := make([]bool, len(s))
	for _, pattern := range placeholderPatterns {
		for _, loc := range pattern.re.FindAllStringIndex(s, -1) {
			if loc == nil || overlapsOccupied(occupied, loc[0], loc[1]) {
				continue
			}
			match := s[loc[0]:loc[1]]
			if strings.HasPrefix(match, "%") && loc[0] > 0 && s[loc[0]-1] == '%' {
				continue
			}
			for i := loc[0]; i < loc[1]; i++ {
				occupied[i] = true
			}
			spans = append(spans, placeholderSpan{start: loc[0], end: loc[1], value: pattern.normalize(match)})
		}
	}
	sort.Slice(spans, func(i, j int) bool {
		if spans[i].start != spans[j].start {
			return spans[i].start < spans[j].start
		}
		return spans[i].end < spans[j].end
	})
	return spans
}

func placeholderCounts(s string) map[string]int {
	counts := map[string]int{}
	for _, span := range collectPlaceholderSpans(s) {
		counts[span.value]++
	}
	return counts
}

func overlapsOccupied(occupied []bool, start int, end int) bool {
	for i := start; i < end; i++ {
		if occupied[i] {
			return true
		}
	}
	return false
}

func normalizeDoubleBracePlaceholder(match string) string {
	inner := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(match, "{{"), "}}"))
	return "{{" + inner + "}}"
}

func identityPlaceholder(match string) string {
	return match
}

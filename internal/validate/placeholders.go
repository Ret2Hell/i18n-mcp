package validate

import (
	"cmp"
	"maps"
	"regexp"
	"slices"
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
	spans := collectPlaceholderSpans(s)
	seen := make(map[string]struct{}, len(spans))
	for _, span := range spans {
		seen[span.value] = struct{}{}
	}
	return slices.Sorted(maps.Keys(seen))
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
			_, hasPercentPrefix := strings.CutPrefix(match, "%")
			if hasPercentPrefix && loc[0] > 0 && s[loc[0]-1] == '%' {
				continue
			}
			for i := loc[0]; i < loc[1]; i++ {
				occupied[i] = true
			}
			spans = append(spans, placeholderSpan{start: loc[0], end: loc[1], value: pattern.normalize(match)})
		}
	}
	slices.SortFunc(spans, func(a, b placeholderSpan) int {
		if byStart := cmp.Compare(a.start, b.start); byStart != 0 {
			return byStart
		}
		return cmp.Compare(a.end, b.end)
	})
	return spans
}

func placeholderCounts(s string) map[string]int {
	spans := collectPlaceholderSpans(s)
	counts := make(map[string]int, len(spans))
	for _, span := range spans {
		counts[span.value]++
	}
	return counts
}

func overlapsOccupied(occupied []bool, start int, end int) bool {
	return slices.Contains(occupied[start:end], true)
}

func normalizeDoubleBracePlaceholder(match string) string {
	inner, _ := strings.CutPrefix(match, "{{")
	inner, _ = strings.CutSuffix(inner, "}}")
	inner = strings.TrimSpace(inner)
	return "{{" + inner + "}}"
}

func identityPlaceholder(match string) string {
	return match
}

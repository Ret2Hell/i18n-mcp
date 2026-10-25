package validate

import (
	"cmp"
	"context"
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
		re:        regexp.MustCompile(`\{[A-Za-z_][A-Za-z0-9_]*\}`),
		normalize: identityPlaceholder,
	},
}

// ExtractPlaceholders returns unique placeholder tokens found in s.
func ExtractPlaceholders(s string) []string {
	tokens, _ := ExtractPlaceholdersFor(context.Background(), s, Options{})
	return tokens
}

// ExtractPlaceholdersFor uses the same syntax policy as validation, including ICU parsing.
func ExtractPlaceholdersFor(ctx context.Context, s string, options Options) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if options.MessageFormat == "raw" {
		return nil, nil
	}
	var tokens []string
	format := options.MessageFormat
	if format == "icu" || ((format == "" || format == "auto") && LooksLikeICU(s)) {
		nodes, err := parseICU(ctx, s)
		if err != nil {
			return nil, err
		}
		for name := range maps.Keys(icuArguments(nodes)) {
			tokens = append(tokens, "{"+name+"}")
		}
	} else {
		for _, span := range collectInterpolationSpans(s, options) {
			tokens = append(tokens, span.value)
		}
	}
	if options.Printf {
		printf, err := printfTokens(s)
		if err != nil {
			return nil, err
		}
		for _, token := range printf {
			tokens = append(tokens, token.raw)
		}
	}
	slices.Sort(tokens)
	tokens = slices.Compact(tokens)
	return tokens, nil
}

func collectPlaceholderSpans(s string) []placeholderSpan {
	return collectInterpolationSpans(s, Options{})
}

func collectInterpolationSpans(s string, options Options) []placeholderSpan {
	var spans []placeholderSpan
	occupied := make([]bool, len(s))
	patterns := placeholderPatterns
	if options.MessageFormat == "i18next" {
		patterns = patterns[:1]
	}
	for _, pattern := range patterns {
		for _, loc := range pattern.re.FindAllStringIndex(s, -1) {
			if loc == nil || overlapsOccupied(occupied, loc[0], loc[1]) {
				continue
			}
			match := s[loc[0]:loc[1]]
			_, hasPercentPrefix := strings.CutPrefix(match, "%")
			if hasPercentPrefix && loc[0] > 0 && s[loc[0]-1] == '%' {
				markOccupied(occupied, loc[0], loc[1])
				continue
			}
			markOccupied(occupied, loc[0], loc[1])
			spans = append(spans, placeholderSpan{start: loc[0], end: loc[1], value: pattern.normalize(match)})
		}
	}
	slices.SortFunc(spans, func(a, b placeholderSpan) int {
		return cmp.Or(
			cmp.Compare(a.start, b.start),
			cmp.Compare(a.end, b.end),
		)
	})
	return spans
}

func placeholderCounts(s string, options Options) map[string]int {
	spans := collectInterpolationSpans(s, options)
	counts := make(map[string]int, len(spans))
	for _, span := range spans {
		counts[span.value]++
	}
	return counts
}

func overlapsOccupied(occupied []bool, start int, end int) bool {
	return slices.Contains(occupied[start:end], true)
}

func markOccupied(occupied []bool, start int, end int) {
	for i := range end - start {
		occupied[start+i] = true
	}
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

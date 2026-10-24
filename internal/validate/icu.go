package validate

import (
	"context"
	"maps"
	"regexp"
	"slices"
)

// This pattern routes legacy auto-format callers to the parser. It is not used
// to extract arguments or validate ICU syntax.
var icuHeadPattern = regexp.MustCompile(`\{\s*[^{}\s,]+\s*,`)

// LooksLikeICU reports whether a message contains an ICU formatted argument.
func LooksLikeICU(s string) bool { return icuHeadPattern.MatchString(s) }

// ExtractICUArguments returns real argument names, including nested/simple ones.
// Malformed messages return no arguments; validation reports the parse error.
func ExtractICUArguments(s string) []string {
	nodes, err := parseICU(context.Background(), s)
	if err != nil {
		return nil
	}
	return slices.Sorted(maps.Keys(icuArguments(nodes)))
}

// HasBalancedBraces is retained for callers that only need a lexical brace check.
// Actual message validation uses parseICU, including apostrophe quoting.
func HasBalancedBraces(s string) bool {
	depth := 0
	for _, r := range s {
		switch r {
		case '{':
			depth++
		case '}':
			depth--
			if depth < 0 {
				return false
			}
		}
	}
	return depth == 0
}

func icuArguments(nodes []icuNode) map[string][]icuNode {
	result := map[string][]icuNode{}
	var walk func([]icuNode)
	walk = func(nodes []icuNode) {
		for _, node := range nodes {
			if node.name == "" {
				continue
			}
			result[node.name] = append(result[node.name], node)
			for _, selector := range slices.Sorted(maps.Keys(node.branches)) {
				walk(node.branches[selector])
			}
		}
	}
	walk(nodes)
	return result
}

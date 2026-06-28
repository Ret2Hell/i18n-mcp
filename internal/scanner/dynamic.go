package scanner

import (
	"regexp"
	"strings"
)

var dynamicCallPatterns = []literalPattern{
	{name: "t-call-dynamic", re: regexp.MustCompile(`(^|[^A-Za-z0-9_$.])t\s*\(\s*([A-Za-z_$][A-Za-z0-9_$]*|` + "`" + `[^` + "`" + `]*\$\{[^` + "`" + `]+\}[^` + "`" + `]*` + "`" + `)`), keyGroup: 2},
	{name: "i18n.t-dynamic", re: regexp.MustCompile(`\bi18n\.t\s*\(\s*([A-Za-z_$][A-Za-z0-9_$]*|` + "`" + `[^` + "`" + `]*\$\{[^` + "`" + `]+\}[^` + "`" + `]*` + "`" + `)`), keyGroup: 1},
	{name: "jsx-i18nkey-dynamic", re: regexp.MustCompile(`\bi18nKey\s*=\s*\{\s*([^"'{}][^}]*)\}`), keyGroup: 1},
}

func scanDynamicHints(path string, data []byte) []DynamicHint {
	var hints []DynamicHint
	for _, pattern := range dynamicCallPatterns {
		matches := pattern.re.FindAllSubmatchIndex(data, -1)
		for _, match := range matches {
			groupStart := pattern.keyGroup * 2
			if groupStart+1 >= len(match) || match[groupStart] < 0 {
				continue
			}
			expression := strings.TrimSpace(string(data[match[groupStart]:match[groupStart+1]]))
			line, column, snippet := location(data, match[0])
			hints = append(hints, DynamicHint{
				KeyPattern: inferDynamicKeyPattern(expression),
				FilePath:   path,
				Line:       line,
				Column:     column,
				Snippet:    snippet,
				Pattern:    pattern.name,
				Confidence: ConfidenceLow,
				Message:    "dynamic translation key construction detected",
			})
		}
	}
	return hints
}

func inferDynamicKeyPattern(expression string) string {
	if strings.HasPrefix(expression, "`") && strings.Contains(expression, "${") {
		trimmed := strings.Trim(expression, "`")
		prefix, _, ok := strings.Cut(trimmed, "${")
		if ok && prefix != "" {
			return prefix + "*"
		}
	}
	return "*"
}

package scanner

import (
	"bytes"
	"regexp"
	"strings"
)

type literalPattern struct {
	name     string
	re       *regexp.Regexp
	keyGroup int
}

var literalPatterns = []literalPattern{
	{name: "t-call-double", re: regexp.MustCompile(`(^|[^A-Za-z0-9_$.])t\s*\(\s*"([^"]+)"`), keyGroup: 2},
	{name: "t-call-single", re: regexp.MustCompile(`(^|[^A-Za-z0-9_$.])t\s*\(\s*'([^']+)'`), keyGroup: 2},
	{name: "i18n.t-double", re: regexp.MustCompile(`\bi18n\.t\s*\(\s*"([^"]+)"`), keyGroup: 1},
	{name: "i18n.t-single", re: regexp.MustCompile(`\bi18n\.t\s*\(\s*'([^']+)'`), keyGroup: 1},
}

var jsxI18nKeyPatterns = []literalPattern{
	{name: "jsx-i18nkey-double", re: regexp.MustCompile(`\bi18nKey\s*=\s*"([^"]+)"`), keyGroup: 1},
	{name: "jsx-i18nkey-single", re: regexp.MustCompile(`\bi18nKey\s*=\s*'([^']+)'`), keyGroup: 1},
	{name: "jsx-i18nkey-brace-double", re: regexp.MustCompile(`\bi18nKey\s*=\s*\{\s*"([^"]+)"\s*\}`), keyGroup: 1},
	{name: "jsx-i18nkey-brace-single", re: regexp.MustCompile(`\bi18nKey\s*=\s*\{\s*'([^']+)'\s*\}`), keyGroup: 1},
}

func scanLiteralCalls(path string, data []byte) []Evidence {
	return scanLiteralPatterns(path, data, literalPatterns, ConfidenceMedium)
}

func scanJSXI18nKeys(path string, data []byte) []Evidence {
	return scanLiteralPatterns(path, data, jsxI18nKeyPatterns, ConfidenceHigh)
}

func scanLiteralPatterns(path string, data []byte, patterns []literalPattern, confidence Confidence) []Evidence {
	var evidence []Evidence
	for _, pattern := range patterns {
		matches := pattern.re.FindAllSubmatchIndex(data, -1)
		for _, match := range matches {
			groupStart := pattern.keyGroup * 2
			if groupStart+1 >= len(match) || match[groupStart] < 0 {
				continue
			}
			key := CleanLiteralKey(string(data[match[groupStart]:match[groupStart+1]]))
			if key == "" {
				continue
			}
			line, column, snippet := location(data, match[0])
			evidence = append(evidence, Evidence{
				Key:        key,
				FullKey:    FullKey("", key),
				FilePath:   path,
				Line:       line,
				Column:     column,
				Snippet:    snippet,
				Pattern:    pattern.name,
				Confidence: confidence,
			})
		}
	}
	return evidence
}

func location(data []byte, offset int) (int, int, string) {
	if offset < 0 {
		offset = 0
	}
	if offset > len(data) {
		offset = len(data)
	}
	line := bytes.Count(data[:offset], []byte("\n")) + 1
	lineStart := bytes.LastIndexByte(data[:offset], '\n') + 1
	column := offset - lineStart + 1
	lineEnd := bytes.IndexByte(data[offset:], '\n')
	if lineEnd < 0 {
		lineEnd = len(data)
	} else {
		lineEnd = offset + lineEnd
	}
	snippet := strings.TrimSpace(string(data[lineStart:lineEnd]))
	if len(snippet) > 240 {
		snippet = snippet[:240]
	}
	return line, column, snippet
}

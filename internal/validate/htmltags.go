package validate

import (
	"regexp"
	"sort"
	"strings"
)

type TagToken struct {
	Raw         string `json:"raw"`
	Name        string `json:"name"`
	Closing     bool   `json:"closing,omitzero"`
	SelfClosing bool   `json:"selfClosing,omitzero"`
	Normalized  string `json:"normalized"`
}

var (
	tagPattern     = regexp.MustCompile(`</?\s*[A-Za-z][A-Za-z0-9_-]*(?:\s+[^<>]*)?/?>`)
	tagNamePattern = regexp.MustCompile(`[A-Za-z][A-Za-z0-9_-]*`)
)

func ExtractTags(s string) []string {
	seen := map[string]bool{}
	var out []string
	for _, token := range ExtractTagTokens(s) {
		if seen[token.Normalized] {
			continue
		}
		seen[token.Normalized] = true
		out = append(out, token.Normalized)
	}
	sort.Strings(out)
	return out
}

func ExtractTagTokens(s string) []TagToken {
	matches := tagPattern.FindAllString(s, -1)
	tokens := make([]TagToken, 0, len(matches))
	for _, match := range matches {
		token, ok := normalizeTag(match)
		if ok {
			tokens = append(tokens, token)
		}
	}
	return tokens
}

func normalizeTag(raw string) (TagToken, bool) {
	trimmed := strings.TrimSpace(raw)
	name := tagNamePattern.FindString(trimmed)
	if name == "" {
		return TagToken{}, false
	}
	name = strings.ToLower(name)
	closing := strings.HasPrefix(trimmed, "</")
	selfClosing := !closing && strings.HasSuffix(strings.TrimSpace(strings.TrimSuffix(trimmed, ">")), "/")

	normalized := "<" + name + ">"
	if closing {
		normalized = "</" + name + ">"
	} else if selfClosing {
		normalized = "<" + name + "/>"
	}

	return TagToken{
		Raw:         raw,
		Name:        name,
		Closing:     closing,
		SelfClosing: selfClosing,
		Normalized:  normalized,
	}, true
}

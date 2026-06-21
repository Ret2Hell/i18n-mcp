package validate

import (
	"maps"
	"regexp"
	"slices"
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
	tokens := ExtractTagTokens(s)
	seen := make(map[string]struct{}, len(tokens))
	for _, token := range tokens {
		seen[token.Normalized] = struct{}{}
	}
	return slices.Sorted(maps.Keys(seen))
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
	_, closing := strings.CutPrefix(trimmed, "</")
	withoutEnd, _ := strings.CutSuffix(trimmed, ">")
	_, selfClosing := strings.CutSuffix(strings.TrimSpace(withoutEnd), "/")
	selfClosing = !closing && selfClosing

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

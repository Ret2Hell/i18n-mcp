package scanner

import (
	"fmt"
	"regexp"
)

type namespaceBinding struct {
	Variable  string
	Namespace string
	Offset    int
}

var namespaceBindingPattern = regexp.MustCompile(`\b(?:const|let|var)\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*=\s*(?:await\s+)?(?:useTranslations|getTranslations)\s*\(\s*["']([^"']+)["']\s*\)`)

func scanNamespaceUsages(path string, data []byte) []Evidence {
	bindings := findNamespaceBindings(data)
	var evidence []Evidence
	for _, binding := range bindings {
		pattern := regexp.MustCompile(fmt.Sprintf(`\b%s\s*\(\s*(?:"([^"]+)"|'([^']+)')`, regexp.QuoteMeta(binding.Variable)))
		matches := pattern.FindAllSubmatchIndex(data[binding.Offset:], -1)
		for _, match := range matches {
			absoluteStart := binding.Offset + match[0]
			keyStart, keyEnd := firstPresentGroup(match, binding.Offset)
			if keyStart < 0 {
				continue
			}
			key := CleanLiteralKey(string(data[keyStart:keyEnd]))
			if key == "" {
				continue
			}
			line, column, snippet := location(data, absoluteStart)
			evidence = append(evidence, Evidence{
				Namespace:  binding.Namespace,
				Key:        key,
				FullKey:    FullKey(binding.Namespace, key),
				FilePath:   path,
				Line:       line,
				Column:     column,
				Snippet:    snippet,
				Pattern:    "namespace-bound-call",
				Confidence: ConfidenceHigh,
			})
		}
	}
	return evidence
}

func findNamespaceBindings(data []byte) []namespaceBinding {
	matches := namespaceBindingPattern.FindAllSubmatchIndex(data, -1)
	bindings := make([]namespaceBinding, 0, len(matches))
	for _, match := range matches {
		if len(match) < 6 || match[2] < 0 || match[4] < 0 {
			continue
		}
		bindings = append(bindings, namespaceBinding{
			Variable:  string(data[match[2]:match[3]]),
			Namespace: string(data[match[4]:match[5]]),
			Offset:    match[1],
		})
	}
	return bindings
}

func firstPresentGroup(match []int, baseOffset int) (int, int) {
	for i := 2; i+1 < len(match); i += 2 {
		if match[i] >= 0 {
			return baseOffset + match[i], baseOffset + match[i+1]
		}
	}
	return -1, -1
}

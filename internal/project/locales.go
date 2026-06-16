package project

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
	"github.com/bmatcuk/doublestar/v4"
)

type LocaleLayout struct {
	Pattern    string                `json:"pattern"`
	Files      []LocaleFileCandidate `json:"files"`
	TotalKeys  int                   `json:"totalKeys"`
	Namespaces []string              `json:"namespaces,omitzero"`
}

type LocaleFileCandidate struct {
	Locale     string `json:"locale"`
	Namespace  string `json:"namespace,omitzero"`
	Path       string `json:"path"`
	Pattern    string `json:"pattern"`
	StringKeys int    `json:"stringKeys"`
	Bytes      int    `json:"bytes"`
}

var commonLocalePatterns = []string{
	"messages/{locale}.json",
	"messages/{locale}/{namespace}.json",
	"locales/{locale}.json",
	"locales/{locale}/{namespace}.json",
	"src/messages/{locale}.json",
	"src/messages/{locale}/{namespace}.json",
	"src/locales/{locale}.json",
	"src/locales/{locale}/{namespace}.json",
}

func detectLocaleLayouts(ctx context.Context, guard *fsutil.Guard) ([]LocaleLayout, []string) {
	var layouts []LocaleLayout
	var warnings []string
	for _, pattern := range commonLocalePatterns {
		select {
		case <-ctx.Done():
			warnings = append(warnings, ctx.Err().Error())
			return layouts, warnings
		default:
		}

		layout, layoutWarnings := detectLocaleLayout(guard, pattern)
		warnings = append(warnings, layoutWarnings...)
		if len(layout.Files) > 0 {
			layouts = append(layouts, layout)
		}
	}
	sort.Slice(layouts, func(i, j int) bool {
		if len(layouts[i].Files) != len(layouts[j].Files) {
			return len(layouts[i].Files) > len(layouts[j].Files)
		}
		if layouts[i].TotalKeys != layouts[j].TotalKeys {
			return layouts[i].TotalKeys > layouts[j].TotalKeys
		}
		return layouts[i].Pattern < layouts[j].Pattern
	})
	sort.Strings(warnings)
	return layouts, warnings
}

func detectLocaleLayout(guard *fsutil.Guard, pattern string) (LocaleLayout, []string) {
	globPattern := patternToGlob(pattern)
	matches, err := doublestar.Glob(os.DirFS(guard.Root()), globPattern)
	if err != nil {
		return LocaleLayout{Pattern: pattern}, []string{err.Error()}
	}
	sort.Strings(matches)

	layout := LocaleLayout{Pattern: pattern}
	seenNamespaces := map[string]bool{}
	var warnings []string
	for _, match := range matches {
		candidate, ok := matchLocalePattern(pattern, filepath.ToSlash(match))
		if !ok {
			continue
		}
		resolved, err := guard.Resolve(match)
		if err != nil {
			warnings = append(warnings, err.Error())
			continue
		}
		keys, bytes, err := countJSONStrings(resolved)
		if err != nil {
			warnings = append(warnings, err.Error())
			continue
		}
		candidate.Path = resolved
		candidate.Pattern = pattern
		candidate.StringKeys = keys
		candidate.Bytes = bytes
		layout.TotalKeys += keys
		layout.Files = append(layout.Files, candidate)
		if candidate.Namespace != "" && !seenNamespaces[candidate.Namespace] {
			seenNamespaces[candidate.Namespace] = true
			layout.Namespaces = append(layout.Namespaces, candidate.Namespace)
		}
	}
	sort.Strings(layout.Namespaces)
	return layout, warnings
}

func patternToGlob(pattern string) string {
	pattern = strings.ReplaceAll(pattern, "{locale}", "*")
	pattern = strings.ReplaceAll(pattern, "{namespace}", "*")
	return filepath.ToSlash(pattern)
}

func matchLocalePattern(pattern string, path string) (LocaleFileCandidate, bool) {
	patternParts := strings.Split(filepath.ToSlash(pattern), "/")
	pathParts := strings.Split(filepath.ToSlash(path), "/")
	if len(patternParts) != len(pathParts) {
		return LocaleFileCandidate{}, false
	}

	var candidate LocaleFileCandidate
	for i := range patternParts {
		pp := patternParts[i]
		actual := pathParts[i]
		switch pp {
		case "{locale}":
			candidate.Locale = actual
		case "{namespace}.json":
			candidate.Namespace = strings.TrimSuffix(actual, ".json")
		case "{namespace}":
			candidate.Namespace = actual
		case "{locale}.json":
			candidate.Locale = strings.TrimSuffix(actual, ".json")
		default:
			if pp != actual {
				return LocaleFileCandidate{}, false
			}
		}
	}
	return candidate, candidate.Locale != ""
}

func countJSONStrings(path string) (int, int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, 0, err
	}
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return 0, len(data), err
	}
	return countStrings(value), len(data), nil
}

func countStrings(value any) int {
	switch v := value.(type) {
	case string:
		return 1
	case []any:
		count := 0
		for _, item := range v {
			count += countStrings(item)
		}
		return count
	case map[string]any:
		count := 0
		for _, item := range v {
			count += countStrings(item)
		}
		return count
	default:
		return 0
	}
}

func uniqueLocales(layouts []LocaleLayout) []string {
	seen := map[string]bool{}
	for _, layout := range layouts {
		for _, file := range layout.Files {
			seen[file.Locale] = true
		}
	}
	locales := make([]string, 0, len(seen))
	for locale := range seen {
		locales = append(locales, locale)
	}
	sort.Strings(locales)
	return locales
}

func bestSourceCandidates(locales []string) []string {
	preferred := []string{"en", "en-US", "en-GB"}
	var out []string
	seen := map[string]bool{}
	for _, locale := range preferred {
		if slices.Contains(locales, locale) {
			out = append(out, locale)
			seen[locale] = true
		}
	}
	for _, locale := range locales {
		if strings.HasPrefix(locale, "en-") && !seen[locale] {
			out = append(out, locale)
			seen[locale] = true
		}
	}
	for _, locale := range locales {
		if !seen[locale] {
			out = append(out, locale)
		}
	}
	return out
}

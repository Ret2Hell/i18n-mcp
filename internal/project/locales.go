package project

import (
	"cmp"
	"context"
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
	"github.com/bmatcuk/doublestar/v4"
)

// LocaleLayout describes a detected locale file layout.
type LocaleLayout struct {
	Pattern    string                `json:"pattern"`
	Files      []LocaleFileCandidate `json:"files"`
	TotalKeys  int                   `json:"totalKeys"`
	Namespaces []string              `json:"namespaces,omitzero"`
}

// LocaleFileCandidate describes a detected locale file.
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
	slices.SortFunc(layouts, func(a, b LocaleLayout) int {
		return cmp.Or(
			cmp.Compare(len(b.Files), len(a.Files)),
			cmp.Compare(b.TotalKeys, a.TotalKeys),
			cmp.Compare(a.Pattern, b.Pattern),
		)
	})
	slices.Sort(warnings)
	return layouts, warnings
}

func detectLocaleLayout(guard *fsutil.Guard, pattern string) (LocaleLayout, []string) {
	globPattern := patternToGlob(pattern)
	matches, err := doublestar.Glob(os.DirFS(guard.Root()), globPattern)
	if err != nil {
		return LocaleLayout{Pattern: pattern}, []string{err.Error()}
	}
	slices.Sort(matches)

	layout := LocaleLayout{Pattern: pattern}
	seenNamespaces := map[string]struct{}{}
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
		if _, ok := seenNamespaces[candidate.Namespace]; candidate.Namespace != "" && !ok {
			seenNamespaces[candidate.Namespace] = struct{}{}
			layout.Namespaces = append(layout.Namespaces, candidate.Namespace)
		}
	}
	slices.Sort(layout.Namespaces)
	return layout, warnings
}

func patternToGlob(pattern string) string {
	pattern = strings.ReplaceAll(pattern, "{locale}", "*")
	pattern = strings.ReplaceAll(pattern, "{namespace}", "*")
	return filepath.ToSlash(pattern)
}

func matchLocalePattern(pattern string, path string) (LocaleFileCandidate, bool) {
	patternParts := slices.Collect(strings.SplitSeq(filepath.ToSlash(pattern), "/"))
	pathParts := slices.Collect(strings.SplitSeq(filepath.ToSlash(path), "/"))
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
			candidate.Namespace, _ = strings.CutSuffix(actual, ".json")
		case "{namespace}":
			candidate.Namespace = actual
		case "{locale}.json":
			candidate.Locale, _ = strings.CutSuffix(actual, ".json")
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
	seen := map[string]struct{}{}
	for _, layout := range layouts {
		for _, file := range layout.Files {
			seen[file.Locale] = struct{}{}
		}
	}
	return slices.Sorted(maps.Keys(seen))
}

func bestSourceCandidates(locales []string) []string {
	preferred := []string{"en", "en-US", "en-GB"}
	var out []string
	seen := map[string]struct{}{}
	for _, locale := range preferred {
		if slices.Contains(locales, locale) {
			out = append(out, locale)
			seen[locale] = struct{}{}
		}
	}
	for _, locale := range locales {
		_, alreadySeen := seen[locale]
		if _, ok := strings.CutPrefix(locale, "en-"); ok && !alreadySeen {
			out = append(out, locale)
			seen[locale] = struct{}{}
		}
	}
	for _, locale := range locales {
		if _, ok := seen[locale]; !ok {
			out = append(out, locale)
		}
	}
	return out
}

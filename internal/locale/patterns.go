package locale

import (
	"cmp"
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
	"github.com/bmatcuk/doublestar/v4"
)

// ExpandPattern returns project-relative files matching a locale pattern.
func ExpandPattern(ctx context.Context, guard *fsutil.Guard, pattern string) ([]string, error) {
	_ = ctx
	if err := validatePattern(pattern); err != nil {
		return nil, err
	}

	glob := patternToGlob(pattern)
	matches, err := doublestar.Glob(
		os.DirFS(guard.Root()),
		glob,
		doublestar.WithFilesOnly(),
		doublestar.WithNoFollow(),
	)
	if err != nil {
		return nil, fmt.Errorf("expand locale pattern %q: %w", pattern, err)
	}

	out := make([]string, 0, len(matches))
	for _, match := range matches {
		rel, _ := strings.CutPrefix(filepath.ToSlash(match), "./")
		if rel == "" || rel == "." {
			continue
		}

		candidate, err := guard.Resolve(rel)
		if err != nil {
			return nil, err
		}
		info, err := os.Lstat(candidate)
		if err != nil {
			return nil, fmt.Errorf("stat locale file %s: %w", rel, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("locale file symlink is not supported: %s", rel)
		}

		if _, err := guard.ResolveExisting(rel); err != nil {
			return nil, err
		}
		out = append(out, rel)
	}

	return uniqueSorted(out), nil
}

func validatePattern(pattern string) error {
	if strings.TrimSpace(pattern) == "" {
		return fmt.Errorf("locale file pattern is empty")
	}
	_, startsWithSlash := strings.CutPrefix(pattern, "/")
	if filepath.IsAbs(pattern) || startsWithSlash {
		return fmt.Errorf("locale file pattern must be project-relative: %s", pattern)
	}
	if strings.Contains(pattern, "\\") {
		return fmt.Errorf("locale file pattern must use slash separators: %s", pattern)
	}
	clean := path.Clean(pattern)
	_, climbsOut := strings.CutPrefix(clean, "../")
	if clean == "." || clean == ".." || climbsOut || strings.Contains(clean, "/../") {
		return fmt.Errorf("locale file pattern must not contain traversal: %s", pattern)
	}
	for part := range strings.SplitSeq(pattern, "/") {
		if part == ".." {
			return fmt.Errorf("locale file pattern must not contain traversal: %s", pattern)
		}
	}
	if !strings.Contains(pattern, "{locale}") {
		return fmt.Errorf("locale file pattern must contain {locale}: %s", pattern)
	}
	if strings.Count(pattern, "{locale}") != 1 {
		return fmt.Errorf("locale file pattern must contain exactly one {locale}: %s", pattern)
	}
	if strings.Count(pattern, "{namespace}") > 1 {
		return fmt.Errorf("locale file pattern must contain at most one {namespace}: %s", pattern)
	}
	if _, ok := strings.CutSuffix(pattern, ".json"); !ok {
		return fmt.Errorf("locale file pattern must point at JSON files: %s", pattern)
	}
	return nil
}

func patternToGlob(pattern string) string {
	glob := strings.ReplaceAll(pattern, "{locale}", "*")
	glob = strings.ReplaceAll(glob, "{namespace}", "*")
	return glob
}

func uniqueSorted(values []string) []string {
	slices.Sort(values)
	return slices.Compact(values)
}

// DiscoverPattern returns locale file references matching a pattern.
func DiscoverPattern(ctx context.Context, guard *fsutil.Guard, pattern string) ([]FileRef, error) {
	paths, err := ExpandPattern(ctx, guard, pattern)
	if err != nil {
		return nil, err
	}

	refs := make([]FileRef, 0, len(paths))
	for _, relPath := range paths {
		ref, ok, err := MatchPattern(pattern, relPath)
		if err != nil {
			return nil, err
		}
		if ok {
			refs = append(refs, ref)
		}
	}
	slices.SortFunc(refs, compareFileRef)
	return refs, nil
}

// MatchPattern extracts locale metadata from relPath using pattern.
func MatchPattern(pattern string, relPath string) (FileRef, bool, error) {
	if err := validatePattern(pattern); err != nil {
		return FileRef{}, false, err
	}

	re, err := patternRegexp(pattern)
	if err != nil {
		return FileRef{}, false, err
	}
	relPath, _ = strings.CutPrefix(filepath.ToSlash(relPath), "./")
	matches := re.FindStringSubmatch(relPath)
	if matches == nil {
		return FileRef{}, false, nil
	}

	ref := FileRef{Path: relPath, Pattern: pattern}
	for i, name := range re.SubexpNames() {
		if i == 0 || name == "" {
			continue
		}
		switch name {
		case "locale":
			ref.Locale = matches[i]
		case "namespace":
			ref.Namespace = matches[i]
		}
	}
	if ref.Locale == "" {
		return FileRef{}, false, nil
	}
	return ref, true, nil
}

func patternRegexp(pattern string) (*regexp.Regexp, error) {
	var b strings.Builder
	b.WriteString("^")

	for len(pattern) > 0 {
		if rest, ok := strings.CutPrefix(pattern, "{locale}"); ok {
			b.WriteString("(?P<locale>[^/]+)")
			pattern = rest
			continue
		}
		if rest, ok := strings.CutPrefix(pattern, "{namespace}"); ok {
			b.WriteString("(?P<namespace>[^/]+)")
			pattern = rest
			continue
		}
		r, size := utf8.DecodeRuneInString(pattern)
		b.WriteString(regexp.QuoteMeta(string(r)))
		pattern = pattern[size:]
	}

	b.WriteString("$")
	return regexp.Compile(b.String())
}

func compareFileRef(a, b FileRef) int {
	return cmp.Or(
		cmp.Compare(a.Locale, b.Locale),
		cmp.Compare(a.Namespace, b.Namespace),
		cmp.Compare(a.Path, b.Path),
		cmp.Compare(a.Pattern, b.Pattern),
	)
}

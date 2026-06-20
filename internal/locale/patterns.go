package locale

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
	"github.com/bmatcuk/doublestar/v4"
)

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
		rel := strings.TrimPrefix(filepath.ToSlash(match), "./")
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
	if filepath.IsAbs(pattern) || strings.HasPrefix(pattern, "/") {
		return fmt.Errorf("locale file pattern must be project-relative: %s", pattern)
	}
	if strings.Contains(pattern, "\\") {
		return fmt.Errorf("locale file pattern must use slash separators: %s", pattern)
	}
	clean := path.Clean(pattern)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || strings.Contains(clean, "/../") {
		return fmt.Errorf("locale file pattern must not contain traversal: %s", pattern)
	}
	if slices.Contains(strings.Split(pattern, "/"), "..") {
		return fmt.Errorf("locale file pattern must not contain traversal: %s", pattern)
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
	if !strings.HasSuffix(pattern, ".json") {
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
	sort.Strings(values)
	out := values[:0]
	for _, value := range values {
		if len(out) == 0 || out[len(out)-1] != value {
			out = append(out, value)
		}
	}
	return out
}

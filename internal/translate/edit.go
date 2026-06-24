package translate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/Ret2Hell/i18n-mcp/internal/config"
	"github.com/Ret2Hell/i18n-mcp/internal/locale"
)

type FileEdit struct {
	Path         string                 `json:"path"`
	Before       []byte                 `json:"-"`
	After        []byte                 `json:"-"`
	Translations []ValidatedTranslation `json:"translations"`
}

func (s *Service) BuildEdits(ctx context.Context, accepted []ValidatedTranslation) ([]FileEdit, error) {
	if len(accepted) == 0 {
		return nil, nil
	}
	cfg, err := s.config.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	inv, err := s.locales.Inventory(ctx)
	if err != nil {
		return nil, err
	}

	byPath := map[string][]ValidatedTranslation{}
	for _, tr := range accepted {
		path, err := s.targetPathFor(cfg, inv, tr.Locale, tr.Namespace)
		if err != nil {
			return nil, err
		}
		byPath[path] = append(byPath[path], tr)
	}

	paths := make([]string, 0, len(byPath))
	for path := range byPath {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	var edits []FileEdit
	for _, path := range paths {
		translations := byPath[path]
		sort.Slice(translations, func(i, j int) bool {
			return proposalIdentity(translations[i].Locale, translations[i].Namespace, translations[i].Key) < proposalIdentity(translations[j].Locale, translations[j].Namespace, translations[j].Key)
		})

		before, value, err := s.readEditableJSON(path)
		if err != nil {
			return nil, err
		}
		for _, tr := range translations {
			if err := setNestedString(value, strings.Split(tr.Key, "."), tr.Value); err != nil {
				return nil, fmt.Errorf("set %s in %s: %w", tr.Key, path, err)
			}
		}
		after, err := renderEditableJSON(value, cfg.Format)
		if err != nil {
			return nil, err
		}
		edits = append(edits, FileEdit{Path: path, Before: before, After: after, Translations: translations})
	}
	return edits, nil
}

func (s *Service) readEditableJSON(relPath string) ([]byte, map[string]any, error) {
	resolved, err := s.guard.Resolve(relPath)
	if err != nil {
		return nil, nil, err
	}
	data, err := os.ReadFile(resolved)
	if errors.Is(err, os.ErrNotExist) {
		return []byte("{}\n"), map[string]any{}, nil
	}
	if err != nil {
		return nil, nil, err
	}
	var value map[string]any
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, nil, fmt.Errorf("parse %s: %w", relPath, err)
	}
	return data, value, nil
}

func setNestedString(value map[string]any, path []string, text string) error {
	if len(path) == 0 {
		return fmt.Errorf("empty JSON path")
	}
	current := value
	for _, part := range path[:len(path)-1] {
		next, ok := current[part]
		if !ok {
			child := map[string]any{}
			current[part] = child
			current = child
			continue
		}
		child, ok := next.(map[string]any)
		if !ok {
			return fmt.Errorf("path segment %q is not an object", part)
		}
		current = child
	}
	current[path[len(path)-1]] = text
	return nil
}

func renderEditableJSON(value map[string]any, format config.FormatConfig) ([]byte, error) {
	indent := format.Indent
	if indent <= 0 {
		indent = 2
	}
	data, err := json.MarshalIndent(value, "", strings.Repeat(" ", indent))
	if err != nil {
		return nil, err
	}
	if format.TrailingNewline {
		data = append(data, '\n')
	}
	return data, nil
}

func (s *Service) targetPathFor(cfg config.Resolved, inv locale.Inventory, localeCode string, namespace string) (string, error) {
	for _, file := range inv.Files {
		if file.Locale == localeCode && file.Namespace == namespace {
			return file.Path, nil
		}
	}
	for _, pattern := range cfg.LocaleFiles {
		path, ok := buildPathFromPattern(pattern, localeCode, namespace, cfg.DefaultNamespace)
		if ok {
			return path, nil
		}
	}
	return "", fmt.Errorf("no locale file path for %s/%s", localeCode, namespace)
}

func buildPathFromPattern(pattern string, localeCode string, namespace string, defaultNamespace string) (string, bool) {
	if !strings.Contains(pattern, "{locale}") {
		return "", false
	}
	if strings.Contains(pattern, "{namespace}") {
		if namespace == "" {
			return "", false
		}
		path := strings.ReplaceAll(pattern, "{locale}", localeCode)
		path = strings.ReplaceAll(path, "{namespace}", namespace)
		return path, true
	}
	if namespace != "" && defaultNamespace != "" && namespace != defaultNamespace {
		return "", false
	}
	return strings.ReplaceAll(pattern, "{locale}", localeCode), true
}

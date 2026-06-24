package translate

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/Ret2Hell/i18n-mcp/internal/config"
	"github.com/Ret2Hell/i18n-mcp/internal/jsonedit"
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

		before, doc, err := s.readEditableJSON(path, cfg)
		if err != nil {
			return nil, err
		}
		for _, tr := range translations {
			if err := doc.SetString(strings.Split(tr.Key, "."), tr.Value); err != nil {
				return nil, fmt.Errorf("set %s in %s: %w", tr.Key, path, err)
			}
		}
		after, err := doc.Render()
		if err != nil {
			return nil, err
		}
		edits = append(edits, FileEdit{Path: path, Before: before, After: after, Translations: translations})
	}
	return edits, nil
}

func (s *Service) readEditableJSON(relPath string, cfg config.Resolved) ([]byte, *jsonedit.Document, error) {
	resolved, err := s.guard.Resolve(relPath)
	if err != nil {
		return nil, nil, err
	}
	data, err := os.ReadFile(resolved)
	if errors.Is(err, os.ErrNotExist) {
		doc := jsonedit.NewObject(cfg.Format.Indent, cfg.Format.SortKeys)
		doc.Format.TrailingNewline = cfg.Format.TrailingNewline
		return []byte("{}\n"), doc, nil
	}
	if err != nil {
		return nil, nil, err
	}
	doc, err := jsonedit.Parse(data, cfg.Format.Indent, cfg.Format.SortKeys)
	if err != nil {
		return nil, nil, fmt.Errorf("parse %s: %w", relPath, err)
	}
	doc.Format.SortKeys = cfg.Format.SortKeys
	doc.Format.TrailingNewline = cfg.Format.TrailingNewline
	return data, doc, nil
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

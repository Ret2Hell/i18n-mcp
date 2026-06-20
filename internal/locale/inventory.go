package locale

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/Ret2Hell/i18n-mcp/internal/config"
	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
)

type Service struct {
	guard  *fsutil.Guard
	config *config.Service
}

func NewService(guard *fsutil.Guard, configService *config.Service) *Service {
	return &Service{guard: guard, config: configService}
}

func (s *Service) Inventory(ctx context.Context) (Inventory, error) {
	cfg, err := s.config.Resolve(ctx)
	if err != nil {
		return Inventory{}, err
	}
	validation := s.config.Validate(ctx, cfg)
	if !validation.Valid {
		return Inventory{}, configValidationError(validation)
	}
	return s.InventoryForConfig(ctx, cfg)
}

func (s *Service) InventoryForConfig(ctx context.Context, cfg config.Resolved) (Inventory, error) {
	inv := Inventory{
		SourceLocale:      cfg.SourceLocale,
		TargetLocales:     slices.Clone(cfg.TargetLocales),
		CountsByLocale:    map[string]int{},
		CountsByNamespace: map[string]int{},
	}

	seenPath := map[string]bool{}
	seenLocale := map[string]bool{}
	seenNamespace := map[string]bool{}
	filesByLocaleNamespace := map[string][]string{}

	for _, pattern := range cfg.LocaleFiles {
		refs, err := DiscoverPattern(ctx, s.guard, pattern)
		if err != nil {
			return Inventory{}, err
		}
		for _, ref := range refs {
			if seenPath[ref.Path] {
				continue
			}
			seenPath[ref.Path] = true
			ref.Namespace = cmp.Or(ref.Namespace, cfg.DefaultNamespace, "common")

			doc, err := ParseJSONFile(ctx, s.guard, ref.Path)
			if err != nil {
				return Inventory{}, err
			}
			flat := Flatten(ref, doc.Value)

			inv.Files = append(inv.Files, FileSummary{
				FileRef:    ref,
				StringKeys: len(flat.Units),
				Bytes:      doc.Bytes,
			})
			inv.Units = append(inv.Units, flat.Units...)
			inv.Warnings = append(inv.Warnings, flat.Warnings...)
			inv.CountsByLocale[ref.Locale] += len(flat.Units)
			inv.CountsByNamespace[ref.Namespace] += len(flat.Units)

			seenLocale[ref.Locale] = true
			seenNamespace[ref.Namespace] = true
			dupKey := ref.Locale + "\x00" + ref.Namespace
			filesByLocaleNamespace[dupKey] = append(filesByLocaleNamespace[dupKey], ref.Path)
		}
	}

	inv.Locales = slices.Sorted(maps.Keys(seenLocale))
	inv.Namespaces = slices.Sorted(maps.Keys(seenNamespace))

	if len(inv.TargetLocales) == 0 {
		for _, locale := range inv.Locales {
			if locale != inv.SourceLocale {
				inv.TargetLocales = append(inv.TargetLocales, locale)
			}
		}
	}

	for key, paths := range filesByLocaleNamespace {
		if len(paths) < 2 {
			continue
		}
		slices.Sort(paths)
		localeCode, namespace, _ := strings.Cut(key, "\x00")
		issue := DuplicateNamespaceIssue{Locale: localeCode, Namespace: namespace, Paths: slices.Clone(paths)}
		inv.Duplicates = append(inv.Duplicates, issue)
		inv.Warnings = append(inv.Warnings, Warning{
			Code:      "duplicate_namespace",
			Message:   "multiple locale files map to the same locale and namespace",
			Locale:    localeCode,
			Namespace: namespace,
		})
	}

	sortInventory(&inv)
	return inv, nil
}

func configValidationError(validation config.ValidationResult) error {
	parts := make([]string, 0, len(validation.Errors))
	for _, diagnostic := range validation.Errors {
		if diagnostic.Field != "" {
			parts = append(parts, diagnostic.Field+": "+diagnostic.Message)
			continue
		}
		parts = append(parts, diagnostic.Message)
	}
	return fmt.Errorf("invalid i18n config: %s", strings.Join(parts, "; "))
}

func sortInventory(inv *Inventory) {
	slices.Sort(inv.TargetLocales)
	slices.SortFunc(inv.Files, func(a, b FileSummary) int { return compareLess(a.FileRef, b.FileRef, lessFileRef) })
	slices.SortFunc(inv.Units, func(a, b Unit) int { return compareLess(a, b, lessUnit) })
	slices.SortFunc(inv.Warnings, func(a, b Warning) int { return compareLess(a, b, lessWarning) })
	slices.SortFunc(inv.Duplicates, func(a, b DuplicateNamespaceIssue) int {
		if a.Locale != b.Locale {
			return strings.Compare(a.Locale, b.Locale)
		}
		return strings.Compare(a.Namespace, b.Namespace)
	})
}

func lessUnit(a, b Unit) bool {
	if a.Locale != b.Locale {
		return a.Locale < b.Locale
	}
	if a.Namespace != b.Namespace {
		return a.Namespace < b.Namespace
	}
	if a.Key != b.Key {
		return a.Key < b.Key
	}
	return a.FilePath < b.FilePath
}

func lessWarning(a, b Warning) bool {
	if a.Locale != b.Locale {
		return a.Locale < b.Locale
	}
	if a.Namespace != b.Namespace {
		return a.Namespace < b.Namespace
	}
	if a.FilePath != b.FilePath {
		return a.FilePath < b.FilePath
	}
	if a.Key != b.Key {
		return a.Key < b.Key
	}
	return a.Code < b.Code
}

var ErrNamespaceNotFound = errors.New("locale namespace not found")

func (s *Service) Namespace(ctx context.Context, localeCode string, namespace string) (NamespaceContent, error) {
	inv, err := s.Inventory(ctx)
	if err != nil {
		return NamespaceContent{}, err
	}

	out := NamespaceContent{Locale: localeCode, Namespace: namespace}
	for _, file := range inv.Files {
		if file.Locale != localeCode || file.Namespace != namespace {
			continue
		}
		out.Files = append(out.Files, file)
		doc, err := ParseJSONFile(ctx, s.guard, file.Path)
		if err != nil {
			return NamespaceContent{}, err
		}
		out.RawFiles = append(out.RawFiles, RawFileContent{Path: file.Path, JSON: doc.Raw})
	}
	for _, unit := range inv.Units {
		if unit.Locale == localeCode && unit.Namespace == namespace {
			out.Units = append(out.Units, unit)
		}
	}
	for _, warning := range inv.Warnings {
		if warning.Locale == localeCode && warning.Namespace == namespace {
			out.Warnings = append(out.Warnings, warning)
		}
	}

	if len(out.Files) == 0 {
		return NamespaceContent{}, fmt.Errorf("%w: %s/%s", ErrNamespaceNotFound, localeCode, namespace)
	}
	return out, nil
}

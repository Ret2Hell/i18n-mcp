package locale

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Ret2Hell/i18n-mcp/internal/config"
	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
	"github.com/stretchr/testify/require"
)

func TestInventoryFlatMessages(t *testing.T) {
	inv := inventoryFixture(t, "next-intl-basic")

	require.Equal(t, "en", inv.SourceLocale)
	require.Equal(t, []string{"en", "fr"}, inv.Locales)
	require.Equal(t, []string{"fr"}, inv.TargetLocales)
	require.Equal(t, []string{"common"}, inv.Namespaces)
	require.Equal(t, 2, inv.CountsByLocale["en"])
	require.Equal(t, 2, inv.CountsByLocale["fr"])
	require.Empty(t, inv.Duplicates)
	require.Empty(t, inv.Warnings)
}

func TestInventoryNamespacedLocales(t *testing.T) {
	inv := inventoryFixture(t, "i18next-namespaces")

	require.Equal(t, "en", inv.SourceLocale)
	require.Equal(t, []string{"de", "en"}, inv.Locales)
	require.Equal(t, []string{"de"}, inv.TargetLocales)
	require.Equal(t, []string{"common"}, inv.Namespaces)
	require.Equal(t, 2, inv.CountsByLocale["en"])
	require.Equal(t, 2, inv.CountsByLocale["de"])
	require.Len(t, inv.Files, 2)
}

func TestInventoryUnsupportedValuesAndDuplicateNamespace(t *testing.T) {
	inv := inventoryFixture(t, "locale-inventory-edge")

	require.NotEmpty(t, inv.Duplicates)
	requireWarningCode(t, inv.Warnings, "duplicate_namespace")
	requireWarningCode(t, inv.Warnings, "unsupported_array")
	requireWarningCode(t, inv.Warnings, "unsupported_non_string")
	requireWarningCode(t, inv.Warnings, "unsupported_rich_object")
	require.Equal(t, 2, inv.CountsByLocale["en"])
}

func TestInventoryForConfigDeduplicatesRepeatedPatterns(t *testing.T) {
	service := localeServiceForFixture(t, "next-intl-basic")
	inv, err := service.InventoryForConfig(t.Context(), config.Resolved{File: config.File{
		SourceLocale:     "en",
		LocaleFiles:      []string{"messages/{locale}.json", "messages/{locale}.json"},
		DefaultNamespace: "common",
	}})
	require.NoError(t, err)

	require.Len(t, inv.Files, 2)
	require.Len(t, inv.Units, 4)
	require.Equal(t, []string{"en", "fr"}, inv.Locales)
	require.Equal(t, []string{"common"}, inv.Namespaces)
	require.Equal(t, []string{"fr"}, inv.TargetLocales)
	require.Equal(t, 2, inv.CountsByLocale["en"])
	require.Equal(t, 2, inv.CountsByLocale["fr"])
	require.Equal(t, 4, inv.CountsByNamespace["common"])
}

func TestConfigValidationErrorIncludesFieldedAndFieldlessDiagnostics(t *testing.T) {
	err := configValidationError(config.ValidationResult{Errors: []config.Diagnostic{
		{Field: "sourceLocale", Message: "sourceLocale is required"},
		{Message: "fieldless diagnostic"},
	}})

	require.EqualError(t, err, "invalid i18n config: sourceLocale: sourceLocale is required; fieldless diagnostic")
}

func TestNamespaceContent(t *testing.T) {
	service := localeServiceForFixture(t, "i18next-namespaces")
	content, err := service.Namespace(t.Context(), "en", "common")
	require.NoError(t, err)

	require.Equal(t, "en", content.Locale)
	require.Equal(t, "common", content.Namespace)
	require.Len(t, content.Files, 1)
	require.Equal(t, "en", content.Files[0].Locale)
	require.Equal(t, "common", content.Files[0].Namespace)
	require.Len(t, content.RawFiles, 1)
	require.Len(t, content.Units, 2)
	for _, unit := range content.Units {
		require.Equal(t, "en", unit.Locale)
		require.Equal(t, "common", unit.Namespace)
	}
	require.Empty(t, content.Warnings)
}

func TestNamespaceContentFiltersWarnings(t *testing.T) {
	service := localeServiceForFixture(t, "locale-inventory-edge")
	content, err := service.Namespace(t.Context(), "en", "common")
	require.NoError(t, err)

	require.Equal(t, "en", content.Locale)
	require.Equal(t, "common", content.Namespace)
	require.Len(t, content.Files, 2)
	require.Len(t, content.Units, 2)
	require.NotEmpty(t, content.Warnings)
	for _, warning := range content.Warnings {
		require.Equal(t, "en", warning.Locale)
		require.Equal(t, "common", warning.Namespace)
	}
	requireWarningCode(t, content.Warnings, "duplicate_namespace")
	requireWarningCode(t, content.Warnings, "unsupported_array")
	requireWarningCode(t, content.Warnings, "unsupported_non_string")
	requireWarningCode(t, content.Warnings, "unsupported_rich_object")
}

func TestNamespaceContentExcludesWarningsFromOtherNamespaces(t *testing.T) {
	root := t.TempDir()
	writeLocaleTestFile(t, root, ".i18n-mcp.json", `{
		"sourceLocale":"en",
		"localeFiles":["locales/{locale}/{namespace}.json"],
		"defaultNamespace":"common"
	}`)
	writeLocaleTestFile(t, root, "locales/en/common.json", `{"hello":"Hello"}`)
	writeLocaleTestFile(t, root, "locales/en/admin.json", `{"items":["a"]}`)
	writeLocaleTestFile(t, root, "locales/fr/common.json", `{"items":["b"]}`)

	guard, err := fsutil.NewGuard(root)
	require.NoError(t, err)
	service := NewService(guard, config.NewService(guard, ""))

	content, err := service.Namespace(t.Context(), "en", "common")
	require.NoError(t, err)
	require.Len(t, content.Files, 1)
	require.Len(t, content.Units, 1)
	require.Empty(t, content.Warnings)
}

func TestNamespaceContentNotFound(t *testing.T) {
	service := localeServiceForFixture(t, "i18next-namespaces")
	_, err := service.Namespace(t.Context(), "fr", "common")
	require.ErrorIs(t, err, ErrNamespaceNotFound)
	require.ErrorContains(t, err, "fr/common")
}

func TestLessUnit(t *testing.T) {
	base := Unit{Locale: "en", Namespace: "common", Key: "home.title", FilePath: "messages/en.json"}

	tests := []struct {
		name string
		a    Unit
		b    Unit
		want int
	}{
		{name: "equal", a: base, b: base, want: 0},
		{name: "locale less", a: Unit{Locale: "de"}, b: base, want: -1},
		{name: "namespace less", a: Unit{Locale: "en", Namespace: "admin"}, b: base, want: -1},
		{name: "key less", a: Unit{Locale: "en", Namespace: "common", Key: "a"}, b: base, want: -1},
		{name: "file less", a: Unit{Locale: "en", Namespace: "common", Key: "home.title", FilePath: "a.json"}, b: base, want: -1},
		{name: "locale greater", a: base, b: Unit{Locale: "de"}, want: 1},
		{name: "namespace greater", a: base, b: Unit{Locale: "en", Namespace: "admin"}, want: 1},
		{name: "key greater", a: base, b: Unit{Locale: "en", Namespace: "common", Key: "a"}, want: 1},
		{name: "file greater", a: base, b: Unit{Locale: "en", Namespace: "common", Key: "home.title", FilePath: "a.json"}, want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, compareUnit(tt.a, tt.b))
		})
	}
}

func TestLessWarning(t *testing.T) {
	base := Warning{Locale: "en", Namespace: "common", FilePath: "messages/en.json", Key: "home.title", Code: "unsupported_array"}

	tests := []struct {
		name string
		a    Warning
		b    Warning
		want int
	}{
		{name: "equal", a: base, b: base, want: 0},
		{name: "locale less", a: Warning{Locale: "de"}, b: base, want: -1},
		{name: "namespace less", a: Warning{Locale: "en", Namespace: "admin"}, b: base, want: -1},
		{name: "file less", a: Warning{Locale: "en", Namespace: "common", FilePath: "a.json"}, b: base, want: -1},
		{name: "key less", a: Warning{Locale: "en", Namespace: "common", FilePath: "messages/en.json", Key: "a"}, b: base, want: -1},
		{name: "code less", a: Warning{Locale: "en", Namespace: "common", FilePath: "messages/en.json", Key: "home.title", Code: "a"}, b: base, want: -1},
		{name: "locale greater", a: base, b: Warning{Locale: "de"}, want: 1},
		{name: "namespace greater", a: base, b: Warning{Locale: "en", Namespace: "admin"}, want: 1},
		{name: "file greater", a: base, b: Warning{Locale: "en", Namespace: "common", FilePath: "a.json"}, want: 1},
		{name: "key greater", a: base, b: Warning{Locale: "en", Namespace: "common", FilePath: "messages/en.json", Key: "a"}, want: 1},
		{name: "code greater", a: base, b: Warning{Locale: "en", Namespace: "common", FilePath: "messages/en.json", Key: "home.title", Code: "a"}, want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, compareWarning(tt.a, tt.b))
		})
	}
}

func inventoryFixture(t *testing.T, name string) Inventory {
	t.Helper()
	service := localeServiceForFixture(t, name)
	inv, err := service.Inventory(t.Context())
	require.NoError(t, err)
	return inv
}

func localeServiceForFixture(t *testing.T, name string) *Service {
	t.Helper()
	guard, err := fsutil.NewGuard(fixturePath(t, name))
	require.NoError(t, err)
	configService := config.NewService(guard, "")
	return NewService(guard, configService)
}

func fixturePath(t *testing.T, name string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Join(filepath.Dir(file), "..", "..", "testdata", "fixtures", name)
}

func writeLocaleTestFile(t *testing.T, root string, name string, data string) {
	t.Helper()
	path := filepath.Join(root, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(data), 0o644))
}

func requireWarningCode(t *testing.T, warnings []Warning, code string) {
	t.Helper()
	for _, warning := range warnings {
		if warning.Code == code {
			return
		}
	}
	require.Failf(t, "missing warning", "expected warning code %q in %#v", code, warnings)
}

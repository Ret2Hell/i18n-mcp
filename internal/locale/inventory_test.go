package locale

import (
	"context"
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

func TestNamespaceContent(t *testing.T) {
	service := localeServiceForFixture(t, "i18next-namespaces")
	content, err := service.Namespace(context.Background(), "en", "common")
	require.NoError(t, err)

	require.Equal(t, "en", content.Locale)
	require.Equal(t, "common", content.Namespace)
	require.Len(t, content.Files, 1)
	require.Len(t, content.RawFiles, 1)
	require.Len(t, content.Units, 2)
}

func inventoryFixture(t *testing.T, name string) Inventory {
	t.Helper()
	service := localeServiceForFixture(t, name)
	inv, err := service.Inventory(context.Background())
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

func requireWarningCode(t *testing.T, warnings []Warning, code string) {
	t.Helper()
	for _, warning := range warnings {
		if warning.Code == code {
			return
		}
	}
	require.Failf(t, "missing warning", "expected warning code %q in %#v", code, warnings)
}

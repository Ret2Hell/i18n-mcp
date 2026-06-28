package scanner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Ret2Hell/i18n-mcp/internal/config"
	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
	"github.com/stretchr/testify/require"
)

func TestScannerFindsLiteralAndJSXKeys(t *testing.T) {
	root := t.TempDir()
	writeScannerFile(t, root, "app/page.tsx", `
export function Page() {
  t("home.title")
  i18n.t('shared.cancel')
  return <Trans i18nKey="home.body" />
}
`)
	service := newScannerService(t, root)
	report, err := service.Scan(t.Context(), ScanInput{})
	require.NoError(t, err)
	requireUsage(t, report, "", "home.title")
	requireUsage(t, report, "", "shared.cancel")
	requireUsage(t, report, "", "home.body")
}

func TestScannerFindsNamespaceBoundKeys(t *testing.T) {
	root := t.TempDir()
	writeScannerFile(t, root, "app/login.tsx", `
export function Login() {
  const t = useTranslations("auth")
  return t("login.title")
}
`)
	service := newScannerService(t, root)
	report, err := service.Scan(t.Context(), ScanInput{})
	require.NoError(t, err)
	requireUsage(t, report, "auth", "login.title")
}

func TestScannerEmitsDynamicHints(t *testing.T) {
	root := t.TempDir()
	writeScannerFile(t, root, "app/routes.tsx", "t(`routes.${slug}`)")
	service := newScannerService(t, root)
	report, err := service.Scan(t.Context(), ScanInput{})
	require.NoError(t, err)
	require.NotEmpty(t, report.DynamicHints)
	require.Equal(t, "routes.*", report.DynamicHints[0].KeyPattern)
}

func TestScannerDiscoveryIgnoresBuildAndVendorDirs(t *testing.T) {
	root := t.TempDir()
	writeScannerFile(t, root, "app/page.tsx", `t("used")`)
	for _, dir := range []string{"node_modules", ".next", "dist", "build", ".git", "coverage"} {
		writeScannerFile(t, root, filepath.Join(dir, "ignored.tsx"), `t("ignored")`)
	}

	report, err := newScannerService(t, root).Scan(t.Context(), ScanInput{})
	require.NoError(t, err)
	require.Len(t, report.Files, 1)
	require.Equal(t, "app/page.tsx", report.Files[0].Path)
	requireUsage(t, report, "", "used")
}

func TestScannerDiscoverySkipsSymlinks(t *testing.T) {
	root := t.TempDir()
	writeScannerFile(t, root, "real/page.tsx", `t("real")`)
	writeScannerFile(t, root, "outside.txt", `t("linked-file")`)
	require.NoError(t, os.Symlink(filepath.Join(root, "outside.txt"), filepath.Join(root, "linked.tsx")))
	require.NoError(t, os.Symlink(filepath.Join(root, "real"), filepath.Join(root, "linked-dir")))

	report, err := newScannerService(t, root).Scan(t.Context(), ScanInput{})
	require.NoError(t, err)
	requireUsage(t, report, "", "real")
	for _, usage := range report.Usages {
		require.NotEqual(t, "linked-file", usage.Key)
	}
}

func newScannerService(t *testing.T, root string) *Service {
	t.Helper()
	guard, err := fsutil.NewGuard(root)
	require.NoError(t, err)
	return NewService(guard, config.NewService(guard, ""))
}

func writeScannerFile(t *testing.T, root string, relPath string, contents string) {
	t.Helper()
	abs := filepath.Join(root, relPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(abs), 0o700))
	require.NoError(t, os.WriteFile(abs, []byte(contents), 0o600))
}

func requireUsage(t *testing.T, report Report, namespace string, key string) {
	t.Helper()
	for _, usage := range report.Usages {
		if usage.Namespace == namespace && usage.Key == key {
			return
		}
	}
	require.Failf(t, "missing usage", "namespace=%q key=%q report=%#v", namespace, key, report.Usages)
}

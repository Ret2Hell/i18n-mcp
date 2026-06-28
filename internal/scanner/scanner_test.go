package scanner

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"slices"
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
	homeBody := requireUsage(t, report, "", "home.body")
	require.Equal(t, "jsx_i18n_key", homeBody.Kind)
	require.Equal(t, ConfidenceHigh, homeBody.Evidence[0].Confidence)
	require.Equal(t, "jsx-i18nkey-double", homeBody.Evidence[0].Pattern)
}

func TestScannerFindsJSXI18nKeyVariants(t *testing.T) {
	root := t.TempDir()
	writeScannerFile(t, root, "app/page.tsx", `
<Trans i18nKey="home.double" />
<Trans i18nKey='home.single' />
<Trans i18nKey={"home.braceDouble"} />
<Trans i18nKey={'home.braceSingle'} />
`)
	report, err := newScannerService(t, root).Scan(t.Context(), ScanInput{})
	require.NoError(t, err)

	cases := map[string]string{
		"home.double":      "jsx-i18nkey-double",
		"home.single":      "jsx-i18nkey-single",
		"home.braceDouble": "jsx-i18nkey-brace-double",
		"home.braceSingle": "jsx-i18nkey-brace-single",
	}
	for key, pattern := range cases {
		usage := requireUsage(t, report, "", key)
		require.Equal(t, "jsx_i18n_key", usage.Kind)
		require.Equal(t, pattern, usage.Evidence[0].Pattern)
		require.Equal(t, ConfidenceHigh, usage.Evidence[0].Confidence)
	}
}

func TestScanLiteralPatternsSkipsInvalidCaptureGroups(t *testing.T) {
	patterns := []literalPattern{
		{name: "missing-group", re: literalPatterns[0].re, keyGroup: 9},
		{name: "valid", re: literalPatterns[0].re, keyGroup: 2},
	}

	evidence := scanLiteralPatterns("app/page.tsx", []byte("  "+`t("valid.key")`), patterns, ConfidenceMedium)

	require.Len(t, evidence, 1)
	require.Equal(t, "valid.key", evidence[0].Key)
	require.Equal(t, "valid", evidence[0].Pattern)
	require.Equal(t, 1, evidence[0].Line)
	require.Equal(t, 2, evidence[0].Column)
}

func TestScanLiteralPatternsSkipsUnmatchedOptionalCaptureGroup(t *testing.T) {
	pattern := literalPattern{name: "optional", re: regexp.MustCompile(`x(?:"([^"]+)")?`), keyGroup: 1}

	evidence := scanLiteralPatterns("app/page.tsx", []byte(`x x"present"`), []literalPattern{pattern}, ConfidenceMedium)

	require.Len(t, evidence, 1)
	require.Equal(t, "present", evidence[0].Key)
	require.Equal(t, 3, evidence[0].Column)
}

func TestLocationBoundaries(t *testing.T) {
	data := []byte("first\n  second line\nthird")

	line, column, snippet := location(data, -5)
	require.Equal(t, 1, line)
	require.Equal(t, 1, column)
	require.Equal(t, "first", snippet)

	line, column, snippet = location(data, 5)
	require.Equal(t, 1, line)
	require.Equal(t, 6, column)
	require.Equal(t, "first", snippet)

	line, column, snippet = location(data, 8)
	require.Equal(t, 2, line)
	require.Equal(t, 3, column)
	require.Equal(t, "second line", snippet)

	line, column, snippet = location(data, len(data)+10)
	require.Equal(t, 3, line)
	require.Equal(t, 6, column)
	require.Equal(t, "third", snippet)
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

func TestScannerDiscoversRequestedFiles(t *testing.T) {
	root := t.TempDir()
	writeScannerFile(t, root, "app/b.tsx", `t("b")`)
	writeScannerFile(t, root, "app/a.ts", `t("a")`)
	writeScannerFile(t, root, "app/ignored.css", `t("ignored")`)
	require.NoError(t, os.Mkdir(filepath.Join(root, "app/dir.tsx"), 0o700))
	require.NoError(t, os.Symlink(filepath.Join(root, "app/a.ts"), filepath.Join(root, "app/link.ts")))

	report, err := newScannerService(t, root).Scan(t.Context(), ScanInput{Files: []string{
		"app/b.tsx",
		"app/ignored.css",
		"app/dir.tsx",
		"app/link.ts",
		"app/a.ts",
	}})

	require.NoError(t, err)
	require.Equal(t, []SourceFile{
		{Path: "app/a.ts", Bytes: len(`t("a")`)},
		{Path: "app/b.tsx", Bytes: len(`t("b")`)},
	}, report.Files)
	requireUsage(t, report, "", "a")
	requireUsage(t, report, "", "b")
}

func TestScannerDiscoverRequestedHonorsContextCancellation(t *testing.T) {
	root := t.TempDir()
	writeScannerFile(t, root, "app/a.ts", `t("a")`)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := newScannerService(t, root).DiscoverSourceFiles(ctx, []string{"app/a.ts"})

	require.ErrorIs(t, err, context.Canceled)
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

func requireUsage(t *testing.T, report Report, namespace string, key string) Usage {
	t.Helper()
	if i := slices.IndexFunc(report.Usages, func(usage Usage) bool {
		return usage.Namespace == namespace && usage.Key == key
	}); i >= 0 {
		return report.Usages[i]
	}
	require.Failf(t, "missing usage", "namespace=%q key=%q report=%#v", namespace, key, report.Usages)
	return Usage{}
}

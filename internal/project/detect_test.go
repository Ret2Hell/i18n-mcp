package project

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/Ret2Hell/i18n-mcp/internal/config"
	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
	"github.com/stretchr/testify/require"
)

func TestDetectNextIntlFlatMessages(t *testing.T) {
	root := fixturePath(t, "next-intl-basic")
	d := detectFixture(t, "next-intl-basic")

	require.Equal(t, root, d.ProjectRoot)
	require.Equal(t, filepath.Base(root), d.Root.Name)
	require.Empty(t, d.Warnings)

	require.True(t, d.NextJS.LooksLikeNextJS)
	require.True(t, d.NextJS.PackageJSON)
	require.Equal(t, filepath.Join(root, "package.json"), d.NextJS.PackageJSONPath)
	require.Equal(t, "15.0.0", d.NextJS.NextDependency)
	require.Equal(t, "19.0.0", d.NextJS.ReactDependency)
	require.Empty(t, d.NextJS.ReactDOMDependency)
	require.True(t, d.NextJS.AppDir)
	require.True(t, d.NextJS.AppRouter)
	require.False(t, d.NextJS.PagesDir)
	require.False(t, d.NextJS.PagesRouter)
	require.False(t, d.NextJS.SrcDir)
	require.Empty(t, d.NextJS.NextConfigFiles)
	require.Empty(t, d.NextJS.UnsupportedNextConfigFiles)
	require.Empty(t, d.NextJS.NextScripts)

	require.Equal(t, "next-intl", d.DetectedLibrary)
	require.Equal(t, []LibraryHint{{Name: "next-intl", Version: "4.0.0", Source: "package.json", Confidence: "high"}}, d.Libraries)

	require.Len(t, d.Layouts, 1)
	layout := d.Layouts[0]
	require.Equal(t, "messages/{locale}.json", layout.Pattern)
	require.Equal(t, 4, layout.TotalKeys)
	require.Empty(t, layout.Namespaces)
	require.Len(t, layout.Files, 2)

	require.Equal(t, []string{"en", "fr"}, sortedLocalesFromFiles(d.LocaleFiles))
	assertLocaleFile(t, d.LocaleFiles, "en", "", "messages/{locale}.json", filepath.Join(root, "messages", "en.json"), 2)
	assertLocaleFile(t, d.LocaleFiles, "fr", "", "messages/{locale}.json", filepath.Join(root, "messages", "fr.json"), 2)

	require.Equal(t, []string{"en", "fr"}, d.SourceCandidates)
	require.Equal(t, "en", d.SourceCandidates[0])
	require.Equal(t, []string{"fr"}, d.TargetLocales)
	assertProposedConfig(t, d.ProposedConfig, config.File{
		SourceLocale:         "en",
		TargetLocales:        []string{"fr"},
		LocaleFiles:          []string{"messages/{locale}.json"},
		DefaultNamespace:     "common",
		TranslationFunctions: []string{"t"},
		NamespaceFunctions:   []string{"useTranslations", "getTranslations"},
	})
}

func TestDetectI18nextNamespacedLocales(t *testing.T) {
	root := fixturePath(t, "i18next-namespaces")
	d := detectFixture(t, "i18next-namespaces")

	require.Equal(t, root, d.ProjectRoot)
	require.Empty(t, d.Warnings)

	require.True(t, d.NextJS.LooksLikeNextJS)
	require.Equal(t, "15.0.0", d.NextJS.NextDependency)
	require.False(t, d.NextJS.AppRouter)
	require.False(t, d.NextJS.AppDir)
	require.True(t, d.NextJS.PagesRouter)
	require.True(t, d.NextJS.PagesDir)
	require.Empty(t, d.NextJS.NextConfigFiles)

	require.Equal(t, "react-i18next", d.DetectedLibrary)
	require.Equal(t, []LibraryHint{
		{Name: "react-i18next", Version: "15.0.0", Source: "package.json", Confidence: "high"},
		{Name: "i18next", Version: "23.0.0", Source: "package.json", Confidence: "high"},
	}, d.Libraries)

	require.Len(t, d.Layouts, 1)
	layout := d.Layouts[0]
	require.Equal(t, "locales/{locale}/{namespace}.json", layout.Pattern)
	require.Equal(t, []string{"common"}, layout.Namespaces)
	require.Equal(t, 4, layout.TotalKeys)
	require.Len(t, layout.Files, 2)

	require.Equal(t, []string{"de", "en"}, sortedLocalesFromFiles(d.LocaleFiles))
	assertLocaleFile(t, d.LocaleFiles, "en", "common", "locales/{locale}/{namespace}.json", filepath.Join(root, "locales", "en", "common.json"), 2)
	assertLocaleFile(t, d.LocaleFiles, "de", "common", "locales/{locale}/{namespace}.json", filepath.Join(root, "locales", "de", "common.json"), 2)

	require.Equal(t, []string{"en", "de"}, d.SourceCandidates)
	require.Equal(t, []string{"de"}, d.TargetLocales)
	assertProposedConfig(t, d.ProposedConfig, config.File{
		SourceLocale:         "en",
		TargetLocales:        []string{"de"},
		LocaleFiles:          []string{"locales/{locale}/{namespace}.json"},
		DefaultNamespace:     "common",
		TranslationFunctions: []string{"t", "i18n.t"},
		NamespaceFunctions:   []string{"useTranslation"},
	})
}

func TestDetectDoesNotRequireConfig(t *testing.T) {
	root := fixturePath(t, "no-config-flat")
	_, err := os.Stat(filepath.Join(root, config.DefaultConfigFile))
	require.ErrorIs(t, err, os.ErrNotExist)

	d := detectFixture(t, "no-config-flat")

	require.Equal(t, root, d.ProjectRoot)
	require.True(t, d.NextJS.LooksLikeNextJS)
	require.Equal(t, "15.0.0", d.NextJS.NextDependency)
	require.Equal(t, []string{filepath.Join(root, "next.config.mjs")}, d.NextJS.NextConfigFiles)
	require.False(t, d.NextJS.AppRouter)
	require.False(t, d.NextJS.PagesRouter)
	require.Empty(t, d.DetectedLibrary)
	require.Empty(t, d.Libraries)

	require.Len(t, d.Layouts, 1)
	require.Equal(t, "messages/{locale}.json", d.Layouts[0].Pattern)
	require.Equal(t, 2, d.Layouts[0].TotalKeys)
	require.Equal(t, []string{"en", "es"}, sortedLocalesFromFiles(d.LocaleFiles))
	assertLocaleFile(t, d.LocaleFiles, "en", "", "messages/{locale}.json", filepath.Join(root, "messages", "en.json"), 1)
	assertLocaleFile(t, d.LocaleFiles, "es", "", "messages/{locale}.json", filepath.Join(root, "messages", "es.json"), 1)

	require.Equal(t, []string{"en", "es"}, d.SourceCandidates)
	require.Equal(t, "en", d.SourceCandidates[0])
	require.Equal(t, []string{"es"}, d.TargetLocales)
	assertProposedConfig(t, d.ProposedConfig, config.File{
		SourceLocale:         "en",
		TargetLocales:        []string{"es"},
		LocaleFiles:          []string{"messages/{locale}.json"},
		DefaultNamespace:     "common",
		TranslationFunctions: []string{"t", "i18n.t"},
		NamespaceFunctions:   []string{"useTranslations", "getTranslations"},
	})
	require.NotEmpty(t, d.Warnings)
	require.Contains(t, d.Warnings, "no supported i18n library dependency was detected")
}

func TestDetectNextIntlSrcAppCurrentConventions(t *testing.T) {
	root := fixturePath(t, "next-intl-src-app")
	d := detectFixture(t, "next-intl-src-app")

	require.Equal(t, root, d.ProjectRoot)
	require.Empty(t, d.Warnings)

	require.True(t, d.NextJS.LooksLikeNextJS)
	require.True(t, d.NextJS.PackageJSON)
	require.Equal(t, "16.2.0", d.NextJS.NextDependency)
	require.Equal(t, "19.2.0", d.NextJS.ReactDependency)
	require.Equal(t, "19.2.0", d.NextJS.ReactDOMDependency)
	require.Equal(t, []string{"build", "dev", "start"}, d.NextJS.NextScripts)
	require.Equal(t, []string{filepath.Join(root, "next.config.ts")}, d.NextJS.NextConfigFiles)
	require.True(t, d.NextJS.NextEnvDTS)
	require.True(t, d.NextJS.SrcDir)
	require.True(t, d.NextJS.AppDir)
	require.True(t, d.NextJS.AppRouter)
	require.False(t, d.NextJS.PagesDir)
	require.False(t, d.NextJS.PagesRouter)
	require.Equal(t, []string{filepath.Join(root, "src", "proxy.ts")}, d.NextJS.ProxyFiles)
	require.Equal(t, []string{filepath.Join(root, "src", "instrumentation.ts")}, d.NextJS.InstrumentationFiles)

	require.Equal(t, "next-intl", d.DetectedLibrary)
	require.Equal(t, []LibraryHint{{Name: "next-intl", Version: "4.1.0", Source: "package.json", Confidence: "high"}}, d.Libraries)
	require.Len(t, d.Layouts, 1)
	require.Equal(t, "src/messages/{locale}.json", d.Layouts[0].Pattern)
	require.Equal(t, 6, d.Layouts[0].TotalKeys)
	require.Equal(t, []string{"en-US", "ja"}, sortedLocalesFromFiles(d.LocaleFiles))
	assertLocaleFile(t, d.LocaleFiles, "en-US", "", "src/messages/{locale}.json", filepath.Join(root, "src", "messages", "en-US.json"), 3)
	assertLocaleFile(t, d.LocaleFiles, "ja", "", "src/messages/{locale}.json", filepath.Join(root, "src", "messages", "ja.json"), 3)
	require.Equal(t, []string{"en-US", "ja"}, d.SourceCandidates)
	require.Equal(t, []string{"ja"}, d.TargetLocales)
	assertProposedConfig(t, d.ProposedConfig, config.File{
		SourceLocale:         "en-US",
		TargetLocales:        []string{"ja"},
		LocaleFiles:          []string{"src/messages/{locale}.json"},
		DefaultNamespace:     "common",
		TranslationFunctions: []string{"t"},
		NamespaceFunctions:   []string{"useTranslations", "getTranslations"},
	})
}

func TestDetectNextI18nextPagesFlatLocales(t *testing.T) {
	root := fixturePath(t, "next-i18next-pages")
	d := detectFixture(t, "next-i18next-pages")

	require.Equal(t, root, d.ProjectRoot)
	require.Empty(t, d.Warnings)
	require.True(t, d.NextJS.LooksLikeNextJS)
	require.True(t, d.NextJS.PagesRouter)
	require.False(t, d.NextJS.AppRouter)
	require.Equal(t, "15.2.1", d.NextJS.NextDependency)
	require.Equal(t, "19.0.0", d.NextJS.ReactDOMDependency)

	require.Equal(t, "next-i18next", d.DetectedLibrary)
	require.Equal(t, []LibraryHint{
		{Name: "next-i18next", Version: "15.4.0", Source: "package.json", Confidence: "high"},
		{Name: "i18next", Version: "23.12.0", Source: "package.json", Confidence: "high"},
	}, d.Libraries)
	require.Len(t, d.Layouts, 1)
	require.Equal(t, "locales/{locale}.json", d.Layouts[0].Pattern)
	require.Equal(t, 4, d.Layouts[0].TotalKeys)
	require.Equal(t, []string{"en", "nl"}, sortedLocalesFromFiles(d.LocaleFiles))
	assertLocaleFile(t, d.LocaleFiles, "en", "", "locales/{locale}.json", filepath.Join(root, "locales", "en.json"), 2)
	assertLocaleFile(t, d.LocaleFiles, "nl", "", "locales/{locale}.json", filepath.Join(root, "locales", "nl.json"), 2)
	require.Equal(t, []string{"en", "nl"}, d.SourceCandidates)
	require.Equal(t, []string{"nl"}, d.TargetLocales)
	assertProposedConfig(t, d.ProposedConfig, config.File{
		SourceLocale:         "en",
		TargetLocales:        []string{"nl"},
		LocaleFiles:          []string{"locales/{locale}.json"},
		DefaultNamespace:     "common",
		TranslationFunctions: []string{"t", "i18n.t"},
		NamespaceFunctions:   []string{"useTranslation"},
	})
}

func TestDetectNextTranslateNamespacedLocales(t *testing.T) {
	root := fixturePath(t, "next-translate-pages")
	d := detectFixture(t, "next-translate-pages")

	require.Equal(t, root, d.ProjectRoot)
	require.Empty(t, d.Warnings)
	require.True(t, d.NextJS.LooksLikeNextJS)
	require.True(t, d.NextJS.PagesRouter)
	require.False(t, d.NextJS.AppRouter)
	require.Equal(t, "15.4.0", d.NextJS.NextDependency)

	require.Equal(t, "next-translate", d.DetectedLibrary)
	require.Equal(t, []LibraryHint{{Name: "next-translate", Version: "2.6.2", Source: "package.json", Confidence: "high"}}, d.Libraries)
	require.Len(t, d.Layouts, 1)
	require.Equal(t, "locales/{locale}/{namespace}.json", d.Layouts[0].Pattern)
	require.Equal(t, []string{"common"}, d.Layouts[0].Namespaces)
	require.Equal(t, 4, d.Layouts[0].TotalKeys)
	require.Equal(t, []string{"en", "pt-BR"}, sortedLocalesFromFiles(d.LocaleFiles))
	assertLocaleFile(t, d.LocaleFiles, "en", "common", "locales/{locale}/{namespace}.json", filepath.Join(root, "locales", "en", "common.json"), 2)
	assertLocaleFile(t, d.LocaleFiles, "pt-BR", "common", "locales/{locale}/{namespace}.json", filepath.Join(root, "locales", "pt-BR", "common.json"), 2)
	require.Equal(t, []string{"en", "pt-BR"}, d.SourceCandidates)
	require.Equal(t, []string{"pt-BR"}, d.TargetLocales)
	assertProposedConfig(t, d.ProposedConfig, config.File{
		SourceLocale:         "en",
		TargetLocales:        []string{"pt-BR"},
		LocaleFiles:          []string{"locales/{locale}/{namespace}.json"},
		DefaultNamespace:     "common",
		TranslationFunctions: []string{"t"},
		NamespaceFunctions:   []string{"useTranslation"},
	})
}

func TestDetectWorkspaceScriptsOnlyApp(t *testing.T) {
	root := fixturePath(t, "workspace-scripts-only")
	d := detectFixture(t, "workspace-scripts-only")

	require.True(t, d.NextJS.LooksLikeNextJS)
	require.Empty(t, d.NextJS.NextDependency)
	require.Equal(t, []string{"build", "dev", "preview"}, d.NextJS.NextScripts)
	require.True(t, d.NextJS.AppRouter)
	require.Empty(t, d.DetectedLibrary)
	require.Contains(t, d.Warnings, "no supported i18n library dependency was detected")
	require.Equal(t, []string{"en", "it"}, sortedLocalesFromFiles(d.LocaleFiles))
	assertLocaleFile(t, d.LocaleFiles, "en", "", "messages/{locale}.json", filepath.Join(root, "messages", "en.json"), 1)
	assertLocaleFile(t, d.LocaleFiles, "it", "", "messages/{locale}.json", filepath.Join(root, "messages", "it.json"), 1)
	require.Equal(t, []string{"en", "it"}, d.SourceCandidates)
	require.Equal(t, []string{"it"}, d.TargetLocales)
	require.Equal(t, []string{"messages/{locale}.json"}, d.ProposedConfig.LocaleFiles)
}

func TestDetectUnsupportedConfigOnlyDoesNotLookLikeNextJS(t *testing.T) {
	root := fixturePath(t, "unsupported-config-only")
	d := detectFixture(t, "unsupported-config-only")

	require.False(t, d.NextJS.LooksLikeNextJS)
	require.Empty(t, d.NextJS.NextConfigFiles)
	require.Equal(t, []string{filepath.Join(root, "next.config.cjs")}, d.NextJS.UnsupportedNextConfigFiles)
	require.Empty(t, d.Layouts)
	require.Empty(t, d.LocaleFiles)
	require.Empty(t, d.SourceCandidates)
	require.Empty(t, d.TargetLocales)
	require.Equal(t, "en", d.ProposedConfig.SourceLocale)
	require.Equal(t, []string{"messages/{locale}.json", "locales/{locale}.json"}, d.ProposedConfig.LocaleFiles)
	require.Contains(t, d.Warnings, "project root does not look like a Next.js app")
	require.Contains(t, d.Warnings, "no supported i18n library dependency was detected")
	require.Contains(t, d.Warnings, "no common JSON locale layout was detected")
	require.Contains(t, d.Warnings, "could not infer source locale")
	require.Contains(t, d.Warnings, "could not infer target locales")
}

func TestDetectNextAppWithoutLocaleFiles(t *testing.T) {
	d := detectFixture(t, "no-locale-next-app")

	require.True(t, d.NextJS.LooksLikeNextJS)
	require.True(t, d.NextJS.AppRouter)
	require.Equal(t, "15.5.0", d.NextJS.NextDependency)
	require.Empty(t, d.Layouts)
	require.Empty(t, d.LocaleFiles)
	require.Empty(t, d.SourceCandidates)
	require.Empty(t, d.TargetLocales)
	require.Equal(t, []string{"messages/{locale}.json", "locales/{locale}.json"}, d.ProposedConfig.LocaleFiles)
	require.Contains(t, d.Warnings, "no supported i18n library dependency was detected")
	require.Contains(t, d.Warnings, "no common JSON locale layout was detected")
	require.Contains(t, d.Warnings, "could not infer source locale")
	require.Contains(t, d.Warnings, "could not infer target locales")
}

func TestDetectInvalidLocaleJSONWarnings(t *testing.T) {
	root := fixturePath(t, "invalid-json-locale")
	d := detectFixture(t, "invalid-json-locale")

	require.True(t, d.NextJS.LooksLikeNextJS)
	require.Equal(t, "next-intl", d.DetectedLibrary)
	require.Len(t, d.Layouts, 1)
	require.Equal(t, "messages/{locale}.json", d.Layouts[0].Pattern)
	require.Equal(t, []string{"fr"}, sortedLocalesFromFiles(d.LocaleFiles))
	assertLocaleFile(t, d.LocaleFiles, "fr", "", "messages/{locale}.json", filepath.Join(root, "messages", "fr.json"), 1)
	require.Equal(t, []string{"fr"}, d.SourceCandidates)
	require.Empty(t, d.TargetLocales)
	require.Equal(t, "fr", d.ProposedConfig.SourceLocale)
	require.Equal(t, []string{"messages/{locale}.json"}, d.ProposedConfig.LocaleFiles)
	require.Contains(t, d.Warnings, "could not infer target locales")
	requireWarningContains(t, d.Warnings, "unexpected end of JSON input")
}

func TestDetectChoosesMostSpecificLayoutWhenMultipleLayoutsExist(t *testing.T) {
	root := fixturePath(t, "multiple-layouts")
	d := detectFixture(t, "multiple-layouts")

	require.Empty(t, d.Warnings)
	require.True(t, d.NextJS.LooksLikeNextJS)
	require.Equal(t, "next-intl", d.DetectedLibrary)
	require.Len(t, d.Layouts, 2)
	require.Equal(t, "locales/{locale}/{namespace}.json", d.Layouts[0].Pattern)
	require.Equal(t, []string{"common", "dashboard"}, d.Layouts[0].Namespaces)
	require.Equal(t, 6, d.Layouts[0].TotalKeys)
	require.Equal(t, "messages/{locale}.json", d.Layouts[1].Pattern)
	require.Equal(t, 2, d.Layouts[1].TotalKeys)
	require.Len(t, d.LocaleFiles, 6)
	require.Equal(t, []string{"en", "fr"}, sortedLocalesFromFiles(d.LocaleFiles))
	assertLocaleFile(t, d.LocaleFiles, "en", "common", "locales/{locale}/{namespace}.json", filepath.Join(root, "locales", "en", "common.json"), 2)
	assertLocaleFile(t, d.LocaleFiles, "fr", "common", "locales/{locale}/{namespace}.json", filepath.Join(root, "locales", "fr", "common.json"), 2)
	assertLocaleFile(t, d.LocaleFiles, "en", "dashboard", "locales/{locale}/{namespace}.json", filepath.Join(root, "locales", "en", "dashboard.json"), 1)
	assertLocaleFile(t, d.LocaleFiles, "fr", "dashboard", "locales/{locale}/{namespace}.json", filepath.Join(root, "locales", "fr", "dashboard.json"), 1)
	assertLocaleFile(t, d.LocaleFiles, "en", "", "messages/{locale}.json", filepath.Join(root, "messages", "en.json"), 1)
	assertLocaleFile(t, d.LocaleFiles, "fr", "", "messages/{locale}.json", filepath.Join(root, "messages", "fr.json"), 1)
	require.Equal(t, []string{"en", "fr"}, d.SourceCandidates)
	require.Equal(t, []string{"fr"}, d.TargetLocales)
	require.Equal(t, []string{"locales/{locale}/{namespace}.json"}, d.ProposedConfig.LocaleFiles)
}

func TestDetectNextJSCurrentConventions(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "package.json", `{
		"scripts": {"dev": "next dev", "build": "next build --webpack", "test": "vitest"},
		"dependencies": {"next": "^16.2.0", "react": "^19.2.0", "react-dom": "^19.2.0"}
	}`)
	writeFile(t, root, "next.config.ts", "export default {}\n")
	writeFile(t, root, "next.config.cjs", "module.exports = {}\n")
	writeFile(t, root, "next.config.cts", "export default {}\n")
	writeFile(t, root, "next-env.d.ts", "/// <reference types=\"next\" />\n")
	writeFile(t, root, filepath.Join("src", "app", "page.tsx"), "export default function Page() { return null }\n")
	writeFile(t, root, filepath.Join("src", "proxy.ts"), "export function proxy() {}\n")
	writeFile(t, root, filepath.Join("src", "instrumentation.ts"), "export function register() {}\n")

	d := detectRoot(t, root)

	require.True(t, d.NextJS.LooksLikeNextJS)
	require.True(t, d.NextJS.PackageJSON)
	require.Equal(t, filepath.Join(root, "package.json"), d.NextJS.PackageJSONPath)
	require.Equal(t, "^16.2.0", d.NextJS.NextDependency)
	require.Equal(t, "^19.2.0", d.NextJS.ReactDependency)
	require.Equal(t, "^19.2.0", d.NextJS.ReactDOMDependency)
	require.Equal(t, []string{"build", "dev"}, d.NextJS.NextScripts)
	require.Equal(t, []string{filepath.Join(root, "next.config.ts")}, d.NextJS.NextConfigFiles)
	require.ElementsMatch(t, []string{filepath.Join(root, "next.config.cjs"), filepath.Join(root, "next.config.cts")}, d.NextJS.UnsupportedNextConfigFiles)
	require.True(t, d.NextJS.NextEnvDTS)
	require.Equal(t, []string{filepath.Join(root, "src", "proxy.ts")}, d.NextJS.ProxyFiles)
	require.Equal(t, []string{filepath.Join(root, "src", "instrumentation.ts")}, d.NextJS.InstrumentationFiles)
	require.True(t, d.NextJS.SrcDir)
	require.True(t, d.NextJS.AppDir)
	require.True(t, d.NextJS.AppRouter)
	require.False(t, d.NextJS.PagesDir)
	require.False(t, d.NextJS.PagesRouter)
}

func TestDetectNextJSRequiresRouterPathsToBeDirectories(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "app", "not a directory\n")
	writeFile(t, root, "pages", "not a directory\n")

	d := detectRoot(t, root)

	require.False(t, d.NextJS.AppDir)
	require.False(t, d.NextJS.PagesDir)
	require.False(t, d.NextJS.LooksLikeNextJS)
	require.Contains(t, d.Warnings, "project root does not look like a Next.js app")
}

func detectFixture(t *testing.T, name string) Detection {
	t.Helper()
	return detectRoot(t, fixturePath(t, name))
}

func detectRoot(t *testing.T, root string) Detection {
	t.Helper()
	guard, err := fsutil.NewGuard(root)
	require.NoError(t, err)
	service := NewService(guard)
	d, err := service.Detect(context.Background(), DetectOptions{})
	require.NoError(t, err)
	return d
}

func fixturePath(t *testing.T, name string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Join(filepath.Dir(file), "..", "..", "testdata", "fixtures", name)
}

func sortedLocalesFromFiles(files []LocaleFileCandidate) []string {
	seen := map[string]bool{}
	var out []string
	for _, file := range files {
		if !seen[file.Locale] {
			seen[file.Locale] = true
			out = append(out, file.Locale)
		}
	}
	sort.Strings(out)
	return out
}

func assertLocaleFile(t *testing.T, files []LocaleFileCandidate, locale string, namespace string, pattern string, path string, stringKeys int) {
	t.Helper()
	for _, file := range files {
		if file.Locale == locale && file.Namespace == namespace && file.Pattern == pattern {
			require.Equal(t, path, file.Path)
			require.Equal(t, stringKeys, file.StringKeys)
			require.Positive(t, file.Bytes)
			return
		}
	}
	require.Failf(t, "locale file not found", "locale=%q namespace=%q pattern=%q files=%+v", locale, namespace, pattern, files)
}

func requireWarningContains(t *testing.T, warnings []string, substring string) {
	t.Helper()
	for _, warning := range warnings {
		if strings.Contains(warning, substring) {
			return
		}
	}
	require.Failf(t, "warning not found", "substring=%q warnings=%v", substring, warnings)
}

func assertProposedConfig(t *testing.T, got config.File, want config.File) {
	t.Helper()
	require.Equal(t, want.SourceLocale, got.SourceLocale)
	require.Equal(t, want.TargetLocales, got.TargetLocales)
	require.Equal(t, want.LocaleFiles, got.LocaleFiles)
	require.Equal(t, want.DefaultNamespace, got.DefaultNamespace)
	require.Equal(t, want.TranslationFunctions, got.TranslationFunctions)
	require.Equal(t, want.NamespaceFunctions, got.NamespaceFunctions)
	require.Empty(t, got.IgnoredKeyPatterns)
	require.Empty(t, got.KeptKeyPatterns)
	require.Empty(t, got.DynamicKeyHints)
	require.Equal(t, 2, got.Format.Indent)
	require.False(t, got.Format.SortKeys)
	require.True(t, got.Format.TrailingNewline)
	require.Equal(t, "agent", got.Translation.Mode)
}

func writeFile(t *testing.T, root string, name string, data string) {
	t.Helper()
	path := filepath.Join(root, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(data), 0o644))
}

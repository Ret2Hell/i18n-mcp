package project

import (
	"maps"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Ret2Hell/i18n-mcp/internal/config"
	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
	"github.com/stretchr/testify/require"
)

func TestDetectI18nProjectWithDelegatedFrameworkHints(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "package.json", `{"dependencies":{"next":"15.0.0","next-intl":"4.0.0","react":"19.0.0"}}`)
	writeFile(t, root, "app/page.tsx", "export default function Page() { return null }\n")
	writeFile(t, root, "messages/en.json", `{"home":{"title":"Welcome","cta":"Start now"}}`)
	writeFile(t, root, "messages/fr.json", `{"home":{"title":"Bienvenue","cta":"Commencer"}}`)

	d := detectRoot(t, root)

	require.Equal(t, root, d.ProjectRoot)
	require.True(t, d.NextJS.LooksLikeNextJS)
	require.True(t, d.NextJS.AppRouter)
	require.Equal(t, "15.0.0", d.NextJS.NextDependency)
	require.Equal(t, "next-intl", d.DetectedLibrary)
	require.Equal(t, []LibraryHint{{Name: "next-intl", Version: "4.0.0", Source: "package.json", Confidence: "high"}}, d.Libraries)
	require.Equal(t, []string{"en", "fr"}, sortedLocalesFromFiles(d.LocaleFiles))
	require.Equal(t, []string{"en", "fr"}, d.SourceCandidates)
	require.Equal(t, []string{"fr"}, d.TargetLocales)
	assertProposedConfig(t, d.ProposedConfig, config.File{
		SourceLocale:         "en",
		TargetLocales:        []string{"fr"},
		LocaleFiles:          []string{"messages/{locale}.json"},
		DefaultNamespace:     "common",
		TranslationFunctions: []string{"t"},
		NamespaceFunctions:   []string{"useTranslations", "getTranslations"},
	})
	require.Empty(t, d.Warnings)
}

func TestDetectNamespacedI18nextProject(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "package.json", `{"dependencies":{"i18next":"23.0.0","react-i18next":"15.0.0"}}`)
	writeFile(t, root, "locales/en/common.json", `{"save":"Save","cancel":"Cancel"}`)
	writeFile(t, root, "locales/de/common.json", `{"save":"Speichern","cancel":"Abbrechen"}`)

	d := detectRoot(t, root)

	require.False(t, d.NextJS.LooksLikeNextJS)
	require.Equal(t, "react-i18next", d.DetectedLibrary)
	require.Equal(t, []string{"de", "en"}, sortedLocalesFromFiles(d.LocaleFiles))
	require.Len(t, d.Layouts, 1)
	require.Equal(t, "locales/{locale}/{namespace}.json", d.Layouts[0].Pattern)
	require.Equal(t, []string{"common"}, d.Layouts[0].Namespaces)
	require.Equal(t, []string{"en", "de"}, d.SourceCandidates)
	require.Equal(t, []string{"de"}, d.TargetLocales)
	require.Equal(t, []string{"locales/{locale}/{namespace}.json"}, d.ProposedConfig.LocaleFiles)
	require.Equal(t, []string{"useTranslation"}, d.ProposedConfig.NamespaceFunctions)
}

func TestDetectDoesNotRequireI18nMCPConfig(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "package.json", `{"private":true}`)
	writeFile(t, root, "messages/en.json", `{"hello":"Hello"}`)
	writeFile(t, root, "messages/es.json", `{"hello":"Hola"}`)
	_, err := os.Stat(filepath.Join(root, config.DefaultConfigFile))
	require.ErrorIs(t, err, os.ErrNotExist)

	d := detectRoot(t, root)

	require.Empty(t, d.DetectedLibrary)
	require.Contains(t, d.Warnings, "no supported i18n library dependency was detected")
	require.Equal(t, []string{"en", "es"}, sortedLocalesFromFiles(d.LocaleFiles))
	require.Equal(t, []string{"es"}, d.TargetLocales)
	require.Equal(t, []string{"messages/{locale}.json"}, d.ProposedConfig.LocaleFiles)
}

func TestLibraryRank(t *testing.T) {
	tests := []struct {
		name string
		want int
	}{
		{name: "next-intl", want: 0},
		{name: "next-i18next", want: 1},
		{name: "next-translate", want: 2},
		{name: "react-i18next", want: 3},
		{name: "i18next", want: 4},
		{name: "unknown", want: 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, libraryRank(tt.name))
		})
	}
}

func TestDetectLibrariesSortsByRank(t *testing.T) {
	hints := detectLibraries(packageJSON{Dependencies: map[string]string{
		"i18next":        "23.0.0",
		"next-translate": "2.0.0",
		"next-i18next":   "15.0.0",
		"next-intl":      "4.0.0",
		"react-i18next":  "15.0.0",
	}})

	require.Equal(t, []LibraryHint{
		{Name: "next-intl", Version: "4.0.0", Source: "package.json", Confidence: "high"},
		{Name: "next-i18next", Version: "15.0.0", Source: "package.json", Confidence: "high"},
		{Name: "next-translate", Version: "2.0.0", Source: "package.json", Confidence: "high"},
		{Name: "react-i18next", Version: "15.0.0", Source: "package.json", Confidence: "high"},
		{Name: "i18next", Version: "23.0.0", Source: "package.json", Confidence: "high"},
	}, hints)
}

func detectRoot(t *testing.T, root string) Detection {
	t.Helper()
	guard, err := fsutil.NewGuard(root)
	require.NoError(t, err)
	d, err := NewService(guard).Detect(t.Context(), DetectOptions{})
	require.NoError(t, err)
	return d
}

func sortedLocalesFromFiles(files []LocaleFileCandidate) []string {
	seen := map[string]struct{}{}
	for _, file := range files {
		seen[file.Locale] = struct{}{}
	}
	return slices.Sorted(maps.Keys(seen))
}

func assertProposedConfig(t *testing.T, got config.File, want config.File) {
	t.Helper()
	require.Equal(t, want.SourceLocale, got.SourceLocale)
	require.Equal(t, want.TargetLocales, got.TargetLocales)
	require.Equal(t, want.LocaleFiles, got.LocaleFiles)
	require.Equal(t, want.DefaultNamespace, got.DefaultNamespace)
	require.Equal(t, want.TranslationFunctions, got.TranslationFunctions)
	require.Equal(t, want.NamespaceFunctions, got.NamespaceFunctions)
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

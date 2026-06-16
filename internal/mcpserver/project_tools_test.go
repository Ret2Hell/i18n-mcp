package mcpserver_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Ret2Hell/i18n-mcp/internal/config"
	"github.com/Ret2Hell/i18n-mcp/internal/project"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

func TestProjectDetectTool(t *testing.T) {
	ctx := t.Context()
	root := fixturePath(t, "next-intl-basic")
	clientSession := newInMemoryMCPClientSession(t, ctx, root)

	res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.project.detect"})
	require.NoError(t, err)
	require.False(t, res.IsError)

	var out struct {
		ProjectRoot      string                        `json:"projectRoot"`
		NextJS           project.NextJSHints           `json:"nextjs"`
		DetectedLibrary  string                        `json:"detectedLibrary"`
		Libraries        []project.LibraryHint         `json:"libraries"`
		Layouts          []project.LocaleLayout        `json:"layouts"`
		LocaleFiles      []project.LocaleFileCandidate `json:"localeFiles"`
		SourceCandidates []string                      `json:"sourceLocaleCandidates"`
		TargetLocales    []string                      `json:"targetLocales"`
		ProposedConfig   config.File                   `json:"proposedConfig"`
		Warnings         []string                      `json:"warnings"`
	}
	unmarshalStructuredContent(t, res.StructuredContent, &out)

	require.Equal(t, root, out.ProjectRoot)
	require.True(t, out.NextJS.LooksLikeNextJS)
	require.True(t, out.NextJS.AppRouter)
	require.False(t, out.NextJS.PagesRouter)
	require.Equal(t, "15.0.0", out.NextJS.NextDependency)
	require.Equal(t, "next-intl", out.DetectedLibrary)
	require.Equal(t, []project.LibraryHint{{Name: "next-intl", Version: "4.0.0", Source: "package.json", Confidence: "high"}}, out.Libraries)
	require.Len(t, out.Layouts, 1)
	require.Equal(t, "messages/{locale}.json", out.Layouts[0].Pattern)
	require.Len(t, out.LocaleFiles, 2)
	require.Equal(t, []string{"en", "fr"}, out.SourceCandidates)
	require.Equal(t, []string{"fr"}, out.TargetLocales)
	require.Equal(t, "en", out.ProposedConfig.SourceLocale)
	require.Equal(t, []string{"fr"}, out.ProposedConfig.TargetLocales)
	require.Equal(t, []string{"messages/{locale}.json"}, out.ProposedConfig.LocaleFiles)
	require.Equal(t, []string{"t"}, out.ProposedConfig.TranslationFunctions)
	require.Equal(t, []string{"useTranslations", "getTranslations"}, out.ProposedConfig.NamespaceFunctions)
	require.Empty(t, out.Warnings)
}

func TestProjectDetectToolWithProjectRootArgument(t *testing.T) {
	ctx := t.Context()
	fixturesRoot := fixtureRoot(t)
	clientSession := newInMemoryMCPClientSession(t, ctx, fixturesRoot)

	res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "i18n.project.detect",
		Arguments: map[string]any{"projectRoot": "i18next-namespaces"},
	})
	require.NoError(t, err)
	require.False(t, res.IsError)

	var out struct {
		ProjectRoot     string              `json:"projectRoot"`
		NextJS          project.NextJSHints `json:"nextjs"`
		DetectedLibrary string              `json:"detectedLibrary"`
		ProposedConfig  config.File         `json:"proposedConfig"`
	}
	unmarshalStructuredContent(t, res.StructuredContent, &out)

	require.Equal(t, fixturePath(t, "i18next-namespaces"), out.ProjectRoot)
	require.True(t, out.NextJS.LooksLikeNextJS)
	require.True(t, out.NextJS.PagesRouter)
	require.Equal(t, "react-i18next", out.DetectedLibrary)
	require.Equal(t, "en", out.ProposedConfig.SourceLocale)
	require.Equal(t, []string{"de"}, out.ProposedConfig.TargetLocales)
	require.Equal(t, []string{"locales/{locale}/{namespace}.json"}, out.ProposedConfig.LocaleFiles)
	require.Contains(t, out.ProposedConfig.NamespaceFunctions, "useTranslation")
}

func TestProjectDetectToolFixtureMatrix(t *testing.T) {
	ctx := t.Context()
	clientSession := newInMemoryMCPClientSession(t, ctx, fixtureRoot(t))

	tests := []struct {
		name            string
		fixture         string
		library         string
		sourceLocale    string
		targetLocales   []string
		localeFiles     []string
		warningContains string
	}{
		{
			name:          "src app next-intl",
			fixture:       "next-intl-src-app",
			library:       "next-intl",
			sourceLocale:  "en-US",
			targetLocales: []string{"ja"},
			localeFiles:   []string{"src/messages/{locale}.json"},
		},
		{
			name:            "no config flat messages",
			fixture:         "no-config-flat",
			sourceLocale:    "en",
			targetLocales:   []string{"es"},
			localeFiles:     []string{"messages/{locale}.json"},
			warningContains: "no supported i18n library dependency was detected",
		},
		{
			name:            "next app no locales",
			fixture:         "no-locale-next-app",
			sourceLocale:    "en",
			localeFiles:     []string{"messages/{locale}.json", "locales/{locale}.json"},
			warningContains: "no common JSON locale layout was detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
				Name:      "i18n.project.detect",
				Arguments: map[string]any{"projectRoot": tt.fixture},
			})
			require.NoError(t, err)
			require.False(t, res.IsError)

			var out struct {
				ProjectRoot     string      `json:"projectRoot"`
				DetectedLibrary string      `json:"detectedLibrary"`
				ProposedConfig  config.File `json:"proposedConfig"`
				Warnings        []string    `json:"warnings"`
			}
			unmarshalStructuredContent(t, res.StructuredContent, &out)

			require.Equal(t, fixturePath(t, tt.fixture), out.ProjectRoot)
			require.Equal(t, tt.library, out.DetectedLibrary)
			require.Equal(t, tt.sourceLocale, out.ProposedConfig.SourceLocale)
			require.Equal(t, tt.targetLocales, out.ProposedConfig.TargetLocales)
			require.Equal(t, tt.localeFiles, out.ProposedConfig.LocaleFiles)
			if tt.warningContains == "" {
				require.Empty(t, out.Warnings)
			} else {
				require.Contains(t, out.Warnings, tt.warningContains)
			}
		})
	}
}

func fixturePath(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join(fixtureRoot(t), name)
}

func fixtureRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Join(filepath.Dir(file), "..", "..", "testdata", "fixtures")
}

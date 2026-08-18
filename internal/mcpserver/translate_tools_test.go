package mcpserver

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/Ret2Hell/i18n-mcp/internal/state"
	"github.com/Ret2Hell/i18n-mcp/internal/translate"
	"github.com/stretchr/testify/require"
)

type mcpGenerateProvider struct {
	name     string
	generate func(context.Context, translate.ProviderRequest) (*translate.ProviderResponse, error)
}

func (p mcpGenerateProvider) Name() string { return p.name }

func (p mcpGenerateProvider) Generate(ctx context.Context, req translate.ProviderRequest) (*translate.ProviderResponse, error) {
	return p.generate(ctx, req)
}

func TestTranslationGenerateConfigProviderMode(t *testing.T) {
	a := newMCPTranslationFixtureApp(t, `"mode": "provider", "provider": "mock"`)
	a.Translation.Providers = translate.NewProviderRegistry(mcpGenerateProvider{name: "mock", generate: func(context.Context, translate.ProviderRequest) (*translate.ProviderResponse, error) {
		return new(translate.ProviderResponse{Proposals: []translate.ProposedTranslation{{Locale: "fr", Namespace: "auth", Key: "login.title", SourceValue: "Log in", Value: "Connexion"}}, Usage: translate.ProviderUsage{TotalTokens: 7}}), nil
	}})

	out, err := translationGenerateHandler(t.Context(), nil, a, TranslationGenerateInput{})

	require.NoError(t, err)
	require.Equal(t, "provider", out.Mode)
	require.Equal(t, "mock", out.Provider)
	require.Equal(t, 7, out.Usage.TotalTokens)
	require.Len(t, out.Proposals, 1)
}

func TestTranslationGenerateExplicitProviderOverridesConfig(t *testing.T) {
	a := newMCPTranslationFixtureApp(t, `"mode": "agent", "provider": "mock"`)
	a.Translation.Providers = translate.NewProviderRegistry(mcpGenerateProvider{name: "mock", generate: func(context.Context, translate.ProviderRequest) (*translate.ProviderResponse, error) {
		return new(translate.ProviderResponse{Proposals: []translate.ProposedTranslation{{Locale: "fr", Namespace: "auth", Key: "login.title", SourceValue: "Log in", Value: "Connexion"}}}), nil
	}})

	out, err := translationGenerateHandler(t.Context(), nil, a, TranslationGenerateInput{Mode: "provider"})

	require.NoError(t, err)
	require.Equal(t, "provider", out.Mode)
	require.Len(t, out.Proposals, 1)
}

func TestTranslationGenerateRejectsDeprecatedSamplingMode(t *testing.T) {
	a := newMCPTranslationFixtureApp(t, `"mode": "sampling"`)

	out, err := translationGenerateHandler(t.Context(), nil, a, TranslationGenerateInput{})

	require.Nil(t, out)
	require.EqualError(t, err, "MCP sampling is deprecated; set translation.mode to agent or provider")
}

func TestTranslationGenerateRejectsAgentMode(t *testing.T) {
	a := newMCPTranslationFixtureApp(t, `"mode": "agent"`)

	out, err := translationGenerateHandler(t.Context(), nil, a, TranslationGenerateInput{})

	require.Nil(t, out)
	require.EqualError(t, err, "translation.generate is unavailable in agent mode; use translation.plan, translation.validate, and translation.apply")
}

func TestTranslationGenerateProviderMissingProviderError(t *testing.T) {
	a := newMCPTranslationFixtureApp(t, `"mode": "provider", "provider": "missing"`)
	a.Translation.Providers = translate.NewProviderRegistry()

	out, err := translationGenerateHandler(t.Context(), nil, a, TranslationGenerateInput{})

	require.Nil(t, out)
	require.EqualError(t, err, `translation provider "missing" is not configured`)
}

func TestTranslationGenerateProviderAcceptedRejectedAndNoWrites(t *testing.T) {
	a := newMCPTranslationFixtureApp(t, `"mode": "agent", "provider": "mock"`)
	beforeLocale := readMCPFixtureFile(t, a.ProjectRoot, "messages/fr/auth.json")
	beforeState := readMCPFixtureFile(t, a.ProjectRoot, state.DefaultStatePath)
	a.Translation.Providers = translate.NewProviderRegistry(mcpGenerateProvider{name: "mock", generate: func(context.Context, translate.ProviderRequest) (*translate.ProviderResponse, error) {
		return new(translate.ProviderResponse{Proposals: []translate.ProposedTranslation{
			{Locale: "fr", Namespace: "auth", Key: "login.title", SourceValue: "Log in", Value: "Connexion"},
			{Locale: "fr", Namespace: "auth", Key: "login.subtitle", SourceValue: "Welcome {name}", Value: "Bienvenue"},
		}}), nil
	}})

	out, err := translationGenerateHandler(t.Context(), nil, a, TranslationGenerateInput{Mode: "provider"})

	require.NoError(t, err)
	require.Len(t, out.Proposals, 1)
	require.Len(t, out.Rejected, 1)
	require.Equal(t, beforeLocale, readMCPFixtureFile(t, a.ProjectRoot, "messages/fr/auth.json"))
	require.Equal(t, beforeState, readMCPFixtureFile(t, a.ProjectRoot, state.DefaultStatePath))
}

func newMCPTranslationFixtureApp(t *testing.T, translationConfig string) *app.App {
	t.Helper()
	root := t.TempDir()
	writeMCPFixtureFile(t, root, ".i18n-mcp.json", `{
  "sourceLocale": "en",
  "targetLocales": ["fr"],
  "localeFiles": ["messages/{locale}/{namespace}.json"],
  "defaultNamespace": "common",
  "format": {"sortKeys": false, "indent": 2, "trailingNewline": true},
  "translation": {`+translationConfig+`}
}
`)
	writeMCPFixtureFile(t, root, "messages/en/auth.json", `{
  "login": {
    "title": "Log in",
    "subtitle": "Welcome {name}"
  }
}
`)
	writeMCPFixtureFile(t, root, "messages/fr/auth.json", `{
  "login": {
    "subtitle": "Bienvenue {name}"
  }
}
`)
	stateFile := state.NewFile("en", time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC))
	data, err := json.MarshalIndent(stateFile, "", "  ")
	require.NoError(t, err)
	writeMCPFixtureFile(t, root, state.DefaultStatePath, string(data)+"\n")
	a, err := app.New(t.Context(), app.Options{ProjectRoot: root, LogLevel: "error"})
	require.NoError(t, err)
	return a
}

func writeMCPFixtureFile(t *testing.T, root string, relPath string, contents string) {
	t.Helper()
	abs := filepath.Join(root, relPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(abs), 0o700))
	require.NoError(t, os.WriteFile(abs, []byte(contents), 0o600))
}

func readMCPFixtureFile(t *testing.T, root string, relPath string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, relPath))
	require.NoError(t, err)
	return string(data)
}

package mcpserver_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/Ret2Hell/i18n-mcp/internal/config"
	"github.com/Ret2Hell/i18n-mcp/internal/mcpserver"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

func TestConfigGetToolOverInMemoryMCP(t *testing.T) {
	ctx := context.Background()
	projectRoot := t.TempDir()
	configPath := filepath.Join(projectRoot, config.DefaultConfigFile)
	require.NoError(t, os.WriteFile(configPath, []byte(`{
  "sourceLocale": "en-US",
  "targetLocales": ["fr", "de"],
  "localeFiles": ["messages/{locale}/{namespace}.json"],
  "format": {"sortKeys": true},
  "translation": {"mode": "provider", "provider": "test-provider"}
}`), 0o600))

	clientSession := newInMemoryMCPClientSession(t, ctx, projectRoot)
	res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.config.get"})
	require.NoError(t, err)
	require.False(t, res.IsError)

	var out struct {
		Config config.Resolved `json:"config"`
	}
	unmarshalStructuredContent(t, res.StructuredContent, &out)

	require.True(t, out.Config.Exists)
	require.Equal(t, projectRoot, out.Config.ProjectRoot)
	require.Equal(t, configPath, out.Config.ConfigPath)
	require.Equal(t, "en-US", out.Config.SourceLocale)
	require.Equal(t, []string{"fr", "de"}, out.Config.TargetLocales)
	require.Equal(t, []string{"messages/{locale}/{namespace}.json"}, out.Config.LocaleFiles)
	require.True(t, out.Config.Format.SortKeys)
	require.Equal(t, 2, out.Config.Format.Indent)
	require.True(t, out.Config.Format.TrailingNewline)
	require.Equal(t, "provider", out.Config.Translation.Mode)
	require.Equal(t, "test-provider", out.Config.Translation.Provider)
}

func TestConfigValidateToolOverInMemoryMCP(t *testing.T) {
	ctx := context.Background()
	projectRoot := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, config.DefaultConfigFile), []byte(`{
  "sourceLocale": "en",
  "targetLocales": ["en"],
  "localeFiles": ["messages/*.json"],
  "format": {"indent": 20},
  "translation": {"mode": "bad"}
}`), 0o600))

	clientSession := newInMemoryMCPClientSession(t, ctx, projectRoot)
	res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.config.validate"})
	require.NoError(t, err)
	require.False(t, res.IsError)

	var out struct {
		Config     config.Resolved         `json:"config"`
		Validation config.ValidationResult `json:"validation"`
	}
	unmarshalStructuredContent(t, res.StructuredContent, &out)

	require.True(t, out.Config.Exists)
	require.Equal(t, projectRoot, out.Config.ProjectRoot)
	require.False(t, out.Validation.Valid)
	require.Empty(t, out.Validation.Warnings)

	codes := make(map[string]bool, len(out.Validation.Errors))
	for _, diagnostic := range out.Validation.Errors {
		codes[diagnostic.Code] = true
	}
	require.True(t, codes["target_contains_source"])
	require.True(t, codes["locale_pattern_missing_locale"])
	require.True(t, codes["invalid_indent"])
	require.True(t, codes["invalid_translation_mode"])
}

func newInMemoryMCPClientSession(t *testing.T, ctx context.Context, projectRoot string) *mcp.ClientSession {
	t.Helper()

	application, err := app.New(ctx, app.Options{ProjectRoot: projectRoot, LogLevel: "error"})
	require.NoError(t, err)

	server := mcpserver.New(application)
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.0.0"}, nil)
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	serverSession, err := server.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, serverSession.Close())
	})

	clientSession, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, clientSession.Close())
	})

	return clientSession
}

func unmarshalStructuredContent(t *testing.T, content any, out any) {
	t.Helper()

	data, err := json.Marshal(content)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, out))
}

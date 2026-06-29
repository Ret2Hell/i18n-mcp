package mcpserver_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Ret2Hell/i18n-mcp/internal/state"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

func TestKeysRenameToolDryRunAndConflict(t *testing.T) {
	ctx, clientSession := newTestClientSession(t, makeRenameMCPFixture(t))

	dryRun, err := clientSession.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.keys.rename", Arguments: map[string]any{
		"namespace": "auth",
		"fromKey":   "login.title",
		"toKey":     "login.heading",
	}})
	require.NoError(t, err)
	require.False(t, dryRun.IsError)

	conflict, err := clientSession.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.keys.rename", Arguments: map[string]any{
		"namespace": "auth",
		"fromKey":   "login.title",
		"toKey":     "login.subtitle",
	}})
	require.NoError(t, err)
	require.True(t, conflict.IsError)
}

func makeRenameMCPFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeRenameMCPFile(t, root, ".i18n-mcp.json", `{
  "sourceLocale": "en",
  "targetLocales": ["fr"],
  "localeFiles": ["messages/{locale}/{namespace}.json"],
  "defaultNamespace": "common",
  "format": {"sortKeys": false, "indent": 2, "trailingNewline": true},
  "translation": {"mode": "agent"}
}
`)
	writeRenameMCPFile(t, root, "messages/en/auth.json", `{
  "login": {
    "title": "Log in",
    "subtitle": "Welcome back"
  }
}
`)
	writeRenameMCPFile(t, root, "messages/fr/auth.json", `{
  "login": {
    "title": "Connexion",
    "subtitle": "Bienvenue"
  }
}
`)
	stateFile := state.NewFile("en", time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC))
	stateFile.Entries[state.EntryKey("fr", "auth", "login.title")] = state.Entry{
		Locale:             "fr",
		Namespace:          "auth",
		Key:                "login.title",
		SourceHash:         state.SourceHash("Log in"),
		TranslatedFromHash: state.SourceHash("Log in"),
		TargetHash:         state.TargetHash("Connexion"),
		Status:             state.StatusCurrent,
		Reviewed:           true,
		UpdatedAt:          time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		UpdatedBy:          "fixture",
	}
	data, err := json.MarshalIndent(stateFile, "", "  ")
	require.NoError(t, err)
	writeRenameMCPFile(t, root, state.DefaultStatePath, string(data)+"\n")
	return root
}

func writeRenameMCPFile(t *testing.T, root string, relPath string, contents string) {
	t.Helper()
	abs := filepath.Join(root, relPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(abs), 0o700))
	require.NoError(t, os.WriteFile(abs, []byte(contents), 0o600))
}

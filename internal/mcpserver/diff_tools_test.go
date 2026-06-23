package mcpserver_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Ret2Hell/i18n-mcp/internal/diff"
	"github.com/Ret2Hell/i18n-mcp/internal/state"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

func TestKeysDiffTool(t *testing.T) {
	ctx, clientSession := newTestClientSession(t, makeMCPDiffFixture(t))

	res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.keys.diff"})
	require.NoError(t, err)
	require.False(t, res.IsError)

	data, err := json.Marshal(res.StructuredContent)
	require.NoError(t, err)
	var out struct {
		Report diff.Report `json:"report"`
	}
	require.NoError(t, json.Unmarshal(data, &out))
	require.Equal(t, 11, out.Report.Summary.Total)
	require.Equal(t, 1, out.Report.Summary.Stale)
}

func TestDiffResource(t *testing.T) {
	ctx, clientSession := newTestClientSession(t, makeMCPDiffFixture(t))

	res, err := clientSession.ReadResource(ctx, &mcp.ReadResourceParams{URI: "i18n://analysis/diff"})
	require.NoError(t, err)
	require.Len(t, res.Contents, 1)
	require.Equal(t, "application/json", res.Contents[0].MIMEType)

	var report diff.Report
	require.NoError(t, json.Unmarshal([]byte(res.Contents[0].Text), &report))
	require.Equal(t, 11, report.Summary.Total)
	require.Equal(t, 1, report.Summary.Invalid)
}

func makeMCPDiffFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeMCPDiffFile(t, root, ".i18n-mcp.json", `{
  "sourceLocale": "en",
  "targetLocales": ["fr", "de"],
  "localeFiles": ["messages/{locale}.json"],
  "defaultNamespace": "common"
}`)
	writeMCPDiffFile(t, root, "messages/en.json", `{
  "current": "Current source",
  "stale": "New stale source",
  "unknown": "Unknown source",
  "invalid": "Hello {name}",
  "missing": "Missing source"
}`)
	writeMCPDiffFile(t, root, "messages/fr.json", `{
  "current": "Source actuelle",
  "stale": "Ancienne traduction",
  "unknown": "Traduction inconnue",
  "invalid": "Bonjour",
  "extra": "Texte en trop"
}`)

	stateFile := state.NewFile("en", time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC))
	stateFile.Entries[state.EntryKey("fr", "common", "current")] = state.Entry{
		Locale:             "fr",
		Namespace:          "common",
		Key:                "current",
		SourceHash:         state.SourceHash("Current source"),
		TranslatedFromHash: state.SourceHash("Current source"),
		TargetHash:         state.TargetHash("Source actuelle"),
		Status:             state.StatusCurrent,
		UpdatedAt:          time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
	}
	stateFile.Entries[state.EntryKey("fr", "common", "stale")] = state.Entry{
		Locale:             "fr",
		Namespace:          "common",
		Key:                "stale",
		SourceHash:         state.SourceHash("Old stale source"),
		TranslatedFromHash: state.SourceHash("Old stale source"),
		TargetHash:         state.TargetHash("Ancienne traduction"),
		Status:             state.StatusCurrent,
		UpdatedAt:          time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
	}
	data, err := json.MarshalIndent(stateFile, "", "  ")
	require.NoError(t, err)
	writeMCPDiffFile(t, root, state.DefaultStatePath, string(data)+"\n")
	return root
}

func writeMCPDiffFile(t *testing.T, root string, relPath string, contents string) {
	t.Helper()
	abs := filepath.Join(root, relPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(abs), 0o700))
	require.NoError(t, os.WriteFile(abs, []byte(contents), 0o600))
}

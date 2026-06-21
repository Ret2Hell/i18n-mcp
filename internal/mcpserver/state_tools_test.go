package mcpserver_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Ret2Hell/i18n-mcp/internal/state"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

func TestStateRebuildToolDryRunAndApply(t *testing.T) {
	root := makeStateFixture(t)
	ctx, clientSession := newTestClientSession(t, root)

	dryRun, err := clientSession.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.state.rebuild"})
	require.NoError(t, err)
	require.False(t, dryRun.IsError)
	assertStateRebuildResult(t, dryRun.StructuredContent, true, false, 1)
	_, err = os.Stat(filepath.Join(root, state.DefaultStatePath))
	require.ErrorIs(t, err, os.ErrNotExist)

	apply, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "i18n.state.rebuild",
		Arguments: map[string]any{"apply": true},
	})
	require.NoError(t, err)
	require.False(t, apply.IsError)
	assertStateRebuildResult(t, apply.StructuredContent, false, true, 1)

	data, err := os.ReadFile(filepath.Join(root, state.DefaultStatePath))
	require.NoError(t, err)
	var file state.File
	require.NoError(t, json.Unmarshal(data, &file))
	require.Equal(t, "en", file.SourceLocale)
	require.Len(t, file.Entries, 1)
}

func assertStateRebuildResult(t *testing.T, structured any, dryRun bool, applied bool, entries int) {
	t.Helper()
	data, err := json.Marshal(structured)
	require.NoError(t, err)
	var out struct {
		Result struct {
			DryRun  bool `json:"dryRun"`
			Applied bool `json:"applied"`
			Entries int  `json:"entries"`
		} `json:"result"`
	}
	require.NoError(t, json.Unmarshal(data, &out))
	require.Equal(t, dryRun, out.Result.DryRun)
	require.Equal(t, applied, out.Result.Applied)
	require.Equal(t, entries, out.Result.Entries)
}

func makeStateFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, root, ".i18n-mcp.json", `{
  "sourceLocale": "en",
  "targetLocales": ["fr"],
  "localeFiles": ["messages/{locale}.json"],
  "defaultNamespace": "common"
}`)
	writeFile(t, root, "messages/en.json", `{
  "hello": "Hello"
}`)
	writeFile(t, root, "messages/fr.json", `{
  "hello": "Bonjour"
}`)
	return root
}

func writeFile(t *testing.T, root string, relPath string, contents string) {
	t.Helper()
	abs := filepath.Join(root, relPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(abs), 0o700))
	require.NoError(t, os.WriteFile(abs, []byte(contents), 0o600))
}

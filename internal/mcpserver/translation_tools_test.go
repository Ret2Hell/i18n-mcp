package mcpserver_test

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/Ret2Hell/i18n-mcp/internal/state"
	"github.com/Ret2Hell/i18n-mcp/internal/translate"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

func TestTranslationApplyToolDryRunByDefaultAndApplyWrites(t *testing.T) {
	root := makeTranslationApplyFixture(t)
	ctx, clientSession := newTestClientSession(t, root)

	args := map[string]any{
		"translations": []map[string]any{{
			"locale":      "fr",
			"namespace":   "common",
			"key":         "hello",
			"sourceValue": "Hello",
			"value":       "Bonjour",
		}},
	}
	dryRun, err := clientSession.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.translation.apply", Arguments: args})
	require.NoError(t, err)
	require.False(t, dryRun.IsError)
	out := decodeApplyOutput(t, dryRun.StructuredContent)
	require.True(t, out.DryRun)
	require.Zero(t, out.Applied)
	require.Len(t, out.ChangedFiles, 1)
	require.False(t, out.ChangedFiles[0].Written)
	require.NoFileExists(t, filepath.Join(root, "messages", "fr.json"))
	require.NoFileExists(t, filepath.Join(root, state.DefaultStatePath))

	args["apply"] = true
	applied, err := clientSession.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.translation.apply", Arguments: args})
	require.NoError(t, err)
	require.False(t, applied.IsError)
	out = decodeApplyOutput(t, applied.StructuredContent)
	require.False(t, out.DryRun)
	require.Equal(t, 1, out.Applied)
	require.Equal(t, 1, out.StateUpdates)
	require.FileExists(t, filepath.Join(root, "messages", "fr.json"))
	require.FileExists(t, filepath.Join(root, state.DefaultStatePath))
}

func TestTranslationApplyToolRejectsWithStructuredError(t *testing.T) {
	root := makeTranslationApplyFixture(t)
	ctx, clientSession := newTestClientSession(t, root)

	res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "i18n.translation.apply",
		Arguments: map[string]any{
			"apply": true,
			"translations": []map[string]any{{
				"locale":      "fr",
				"namespace":   "common",
				"key":         "hello",
				"sourceValue": "Old source",
				"value":       "Bonjour",
			}},
		},
	})
	require.NoError(t, err)
	require.True(t, res.IsError)
	out := decodeApplyOutput(t, res.StructuredContent)
	require.Len(t, out.Rejected, 1)
	require.NoFileExists(t, filepath.Join(root, "messages", "fr.json"))
	require.NoFileExists(t, filepath.Join(root, state.DefaultStatePath))
}

func decodeApplyOutput(t *testing.T, structured any) translate.ApplyOutput {
	t.Helper()
	data, err := json.Marshal(structured)
	require.NoError(t, err)
	var out translate.ApplyOutput
	require.NoError(t, json.Unmarshal(data, &out))
	return out
}

func makeTranslationApplyFixture(t *testing.T) string {
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
	return root
}

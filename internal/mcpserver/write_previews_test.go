package mcpserver_test

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Ret2Hell/i18n-mcp/internal/config"
	"github.com/Ret2Hell/i18n-mcp/internal/keyops"
	"github.com/Ret2Hell/i18n-mcp/internal/pagination"
	"github.com/Ret2Hell/i18n-mcp/internal/preview"
	"github.com/Ret2Hell/i18n-mcp/internal/translate"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

func assertBoundedResult(t *testing.T, res *mcp.CallToolResult) {
	t.Helper()
	raw, err := json.Marshal(res)
	require.NoError(t, err)
	require.LessOrEqual(t, len(raw), pagination.ResponseBudget)
}

func readCompletePatch(t *testing.T, session *mcp.ClientSession, file preview.File) {
	t.Helper()
	uri := file.PatchURI
	var full strings.Builder
	for uri != "" {
		res, err := session.ReadResource(t.Context(), &mcp.ReadResourceParams{URI: uri})
		require.NoError(t, err)
		raw, err := json.Marshal(res)
		require.NoError(t, err)
		require.LessOrEqual(t, len(raw), pagination.ResponseBudget)
		var chunk struct {
			Offset       int    `json:"offset"`
			Data         string `json:"data"`
			SHA256       string `json:"sha256"`
			NextResource string `json:"nextResource"`
		}
		require.NoError(t, json.Unmarshal([]byte(res.Contents[0].Text), &chunk))
		require.Equal(t, full.Len(), chunk.Offset)
		decoded, err := base64.StdEncoding.DecodeString(chunk.Data)
		require.NoError(t, err)
		full.Write(decoded)
		uri = chunk.NextResource
	}
	require.Equal(t, file.DiffBytes, full.Len())
	sum := sha256.Sum256([]byte(full.String()))
	require.Equal(t, file.DiffSHA256, hex.EncodeToString(sum[:]))
}

func TestLargeRenameRequiresReviewAndRejectsStalePatch(t *testing.T) {
	root := t.TempDir()
	raw, err := json.Marshal(map[string]string{"old": strings.Repeat("é", 20000)})
	require.NoError(t, err)
	writeFile(t, root, "locales/en.json", string(raw))
	writeFile(t, root, "locales/fr.json", string(raw))
	ctx, session := newTestClientSession(t, root)
	args := map[string]any{"namespace": "common", "fromKey": "old", "toKey": "new"}
	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.keys.rename", Arguments: args})
	require.NoError(t, err)
	require.False(t, res.IsError)
	assertBoundedResult(t, res)
	var out keyops.RenameOutput
	unmarshalStructuredContent(t, res.StructuredContent, &out)
	require.True(t, out.Preview.Truncated)
	for _, file := range out.ChangedFiles {
		readCompletePatch(t, session, file)
	}
	token := out.Preview.Token
	args["apply"] = true
	res, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.keys.rename", Arguments: args})
	require.NoError(t, err)
	require.True(t, res.IsError)
	assertBoundedResult(t, res)
	fr, err := os.ReadFile(filepath.Join(root, "locales/fr.json"))
	require.NoError(t, err)
	require.Equal(t, string(raw), string(fr))
	writeFile(t, root, "locales/en.json", `{"old":"Changed outside preview"}`)
	args["previewToken"] = token
	res, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.keys.rename", Arguments: args})
	require.NoError(t, err)
	require.True(t, res.IsError)
	fr, err = os.ReadFile(filepath.Join(root, "locales/fr.json"))
	require.NoError(t, err)
	require.Equal(t, string(raw), string(fr))
	delete(args, "previewToken")
	args["apply"] = false
	res, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.keys.rename", Arguments: args})
	require.NoError(t, err)
	require.False(t, res.IsError)
	unmarshalStructuredContent(t, res.StructuredContent, &out)
	for _, file := range out.ChangedFiles {
		readCompletePatch(t, session, file)
	}
	args["previewToken"] = out.Preview.Token
	args["apply"] = true
	res, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.keys.rename", Arguments: args})
	require.NoError(t, err)
	require.False(t, res.IsError)
	assertBoundedResult(t, res)
	unmarshalStructuredContent(t, res.StructuredContent, &out)
	require.Equal(t, 2, out.Renamed)
	for _, file := range out.ChangedFiles {
		require.True(t, file.Written)
	}
}

func TestLargeConfigPreviewIsBoundedBeforeApply(t *testing.T) {
	root := t.TempDir()
	old := map[string]any{"sourceLocale": "en", "localeFiles": []string{"locales/{locale}.json"}, "ignoredKeyPatterns": strings.Split(strings.Repeat("retained,", 5000), ",")}
	raw, err := json.Marshal(old)
	require.NoError(t, err)
	writeFile(t, root, ".i18n-mcp.json", string(raw))
	ctx, session := newTestClientSession(t, root)
	args := map[string]any{"config": map[string]any{"sourceLocale": "en", "localeFiles": []string{"locales/{locale}.json"}}, "apply": true}
	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.config.write", Arguments: args})
	require.NoError(t, err)
	require.True(t, res.IsError)
	assertBoundedResult(t, res)
	var out struct {
		Result config.WriteOutput `json:"result"`
	}
	unmarshalStructuredContent(t, res.StructuredContent, &out)
	require.False(t, out.Result.Applied)
	require.True(t, out.Result.Preview.Truncated)
	readCompletePatch(t, session, out.Result.ChangedFile)
	contents, err := os.ReadFile(filepath.Join(root, ".i18n-mcp.json"))
	require.NoError(t, err)
	require.Equal(t, string(raw), string(contents))
	args["previewToken"] = out.Result.Preview.Token
	res, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.config.write", Arguments: args})
	require.NoError(t, err)
	require.False(t, res.IsError)
	assertBoundedResult(t, res)
}

func TestLargeTranslationPreviewSummarizesStateAndRequiresReview(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "locales/en.json", `{"hello":"Hello"}`)
	raw, err := json.Marshal(map[string]string{"hello": strings.Repeat("a", 40000)})
	require.NoError(t, err)
	writeFile(t, root, "locales/fr.json", string(raw))
	ctx, session := newTestClientSession(t, root)
	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.translation.plan", Arguments: map[string]any{"statuses": []string{"unknown"}}})
	require.NoError(t, err)
	require.False(t, res.IsError)
	var batch struct {
		Batch translate.Batch `json:"batch"`
	}
	unmarshalStructuredContent(t, res.StructuredContent, &batch)
	args := map[string]any{"batchId": batch.Batch.BatchID, "translations": []map[string]any{{"locale": "fr", "namespace": "common", "key": "hello", "sourceValue": "Hello", "value": "Bonjour"}}, "apply": true}
	res, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.translation.apply", Arguments: args})
	require.NoError(t, err)
	require.True(t, res.IsError)
	assertBoundedResult(t, res)
	var out translate.ApplyOutput
	unmarshalStructuredContent(t, res.StructuredContent, &out)
	require.Equal(t, 1, out.PlannedStateUpdates)
	require.Zero(t, out.Applied)
	_, err = os.Stat(filepath.Join(root, ".i18n-mcp/state.json"))
	require.ErrorIs(t, err, os.ErrNotExist)
	for _, file := range out.ChangedFiles {
		readCompletePatch(t, session, file)
	}
	args["previewToken"] = out.Preview.Token
	res, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.translation.apply", Arguments: args})
	require.NoError(t, err)
	require.False(t, res.IsError)
	assertBoundedResult(t, res)
	unmarshalStructuredContent(t, res.StructuredContent, &out)
	require.Equal(t, 1, out.Applied)
	require.Equal(t, 1, out.StateUpdates)
}

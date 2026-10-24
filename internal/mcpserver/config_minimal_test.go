package mcpserver_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Ret2Hell/i18n-mcp/internal/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

func TestReadmeMinimalConfigPreviewApplyAndExplicitFalse(t *testing.T) {
	root := t.TempDir()
	ctx, session := newTestClientSession(t, root)
	minimal := map[string]any{"sourceLocale": "en", "targetLocales": []string{"fr"}, "localeFiles": []string{"messages/{locale}.json"}, "defaultNamespace": "common"}
	args := map[string]any{"config": minimal}
	preview, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.config.write", Arguments: args})
	require.NoError(t, err)
	require.False(t, preview.IsError)
	_, err = os.Stat(filepath.Join(root, config.DefaultConfigFile))
	require.ErrorIs(t, err, os.ErrNotExist)
	args["apply"] = true
	applied, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.config.write", Arguments: args})
	require.NoError(t, err)
	require.False(t, applied.IsError)
	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.config.get"})
	require.NoError(t, err)
	var out struct {
		Config config.Resolved `json:"config"`
	}
	unmarshalStructuredContent(t, res.StructuredContent, &out)
	require.Equal(t, 2, out.Config.Format.Indent)
	require.True(t, out.Config.CI.FailOnMissing)
	require.True(t, out.Config.Format.TrailingNewline)
	minimal["format"] = map[string]any{"indent": 0, "trailingNewline": false}
	minimal["ci"] = map[string]any{"failOnMissing": false, "failOnInvalid": false}
	applied, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.config.write", Arguments: args})
	require.NoError(t, err)
	require.False(t, applied.IsError)
	res, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.config.get"})
	require.NoError(t, err)
	unmarshalStructuredContent(t, res.StructuredContent, &out)
	require.Zero(t, out.Config.Format.Indent)
	require.False(t, out.Config.Format.TrailingNewline)
	require.False(t, out.Config.CI.FailOnMissing)
	require.False(t, out.Config.CI.FailOnInvalid)
	res, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.config.validate"})
	require.NoError(t, err)
	require.False(t, res.IsError)
}

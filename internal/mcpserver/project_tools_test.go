package mcpserver_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Ret2Hell/i18n-mcp/internal/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

func TestProjectDetectTool(t *testing.T) {
	ctx := t.Context()
	root := t.TempDir()
	writeProjectDetectFixtureFile(t, root, "package.json", `{"dependencies":{"next-intl":"4.0.0"}}`)
	writeProjectDetectFixtureFile(t, root, "messages/en.json", `{"home":{"title":"Welcome"}}`)
	writeProjectDetectFixtureFile(t, root, "messages/fr.json", `{"home":{"title":"Bienvenue"}}`)
	clientSession := newInMemoryMCPClientSession(t, ctx, root)

	res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.project.detect"})
	require.NoError(t, err)
	require.False(t, res.IsError)

	var out struct {
		ProjectRoot     string      `json:"projectRoot"`
		DetectedLibrary string      `json:"detectedLibrary"`
		ProposedConfig  config.File `json:"proposedConfig"`
	}
	unmarshalStructuredContent(t, res.StructuredContent, &out)

	require.Equal(t, root, out.ProjectRoot)
	require.Equal(t, "next-intl", out.DetectedLibrary)
	require.Equal(t, "en", out.ProposedConfig.SourceLocale)
	require.Equal(t, []string{"fr"}, out.ProposedConfig.TargetLocales)
	require.Equal(t, []string{"messages/{locale}.json"}, out.ProposedConfig.LocaleFiles)
	require.Equal(t, []string{"t"}, out.ProposedConfig.TranslationFunctions)
}

func TestProjectDetectProgressNotifications(t *testing.T) {
	ctx := t.Context()
	root := t.TempDir()
	writeProjectDetectFixtureFile(t, root, "package.json", `{"dependencies":{"next-intl":"4.0.0"}}`)
	writeProjectDetectFixtureFile(t, root, "messages/en.json", `{"home":{"title":"Welcome"}}`)
	writeProjectDetectFixtureFile(t, root, "messages/fr.json", `{"home":{"title":"Bienvenue"}}`)

	var progress []*mcp.ProgressNotificationParams
	clientSession := newInMemoryMCPClientSessionWithOptions(t, ctx, root, &mcp.ClientOptions{
		ProgressNotificationHandler: func(_ context.Context, req *mcp.ProgressNotificationClientRequest) {
			progress = append(progress, req.Params)
		},
	})

	withoutToken, err := clientSession.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.project.detect"})
	require.NoError(t, err)
	require.False(t, withoutToken.IsError)
	require.Empty(t, progress)

	params := &mcp.CallToolParams{Name: "i18n.project.detect"}
	params.SetProgressToken("detect-progress")
	withToken, err := clientSession.CallTool(ctx, params)
	require.NoError(t, err)
	require.False(t, withToken.IsError)
	require.NotEmpty(t, progress)
	require.Equal(t, "detect-progress", progress[0].ProgressToken)

	require.Equal(t, withoutToken.StructuredContent, withToken.StructuredContent)
}

func writeProjectDetectFixtureFile(t *testing.T, root string, name string, data string) {
	t.Helper()
	path := filepath.Join(root, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(data), 0o644))
}

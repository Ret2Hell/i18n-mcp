package mcpserver_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Ret2Hell/i18n-mcp/internal/config"
	"github.com/Ret2Hell/i18n-mcp/internal/state"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

func TestBootstrapWorkflow(t *testing.T) {
	root := makeBootstrapFixture(t)
	ctx, clientSession := newTestClientSession(t, root)

	detect, err := clientSession.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.project.detect", Arguments: map[string]any{}})
	require.NoError(t, err)
	require.False(t, detect.IsError)
	proposed := proposedConfigFromDetect(t, detect.StructuredContent)
	require.NotEmpty(t, proposed.LocaleFiles)

	dryRun, err := clientSession.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.config.write", Arguments: map[string]any{"config": proposed}})
	require.NoError(t, err)
	require.False(t, dryRun.IsError)
	require.NoFileExists(t, filepath.Join(root, config.DefaultConfigFile))

	apply, err := clientSession.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.config.write", Arguments: map[string]any{"config": proposed, "apply": true}})
	require.NoError(t, err)
	require.False(t, apply.IsError)
	require.FileExists(t, filepath.Join(root, config.DefaultConfigFile))

	validated, err := clientSession.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.config.validate", Arguments: map[string]any{}})
	require.NoError(t, err)
	require.False(t, validated.IsError)

	locales, err := clientSession.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.locales.list", Arguments: map[string]any{}})
	require.NoError(t, err)
	require.False(t, locales.IsError)

	stateDryRun, err := clientSession.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.state.rebuild", Arguments: map[string]any{}})
	require.NoError(t, err)
	require.False(t, stateDryRun.IsError)
	require.NoFileExists(t, filepath.Join(root, state.DefaultStatePath))

	stateApply, err := clientSession.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.state.rebuild", Arguments: map[string]any{"apply": true}})
	require.NoError(t, err)
	require.False(t, stateApply.IsError)
	require.FileExists(t, filepath.Join(root, state.DefaultStatePath))
}

func makeBootstrapFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeBootstrapFile(t, root, "package.json", `{"dependencies":{"next":"latest","next-intl":"latest"}}
`)
	writeBootstrapFile(t, root, "messages/en.json", `{
  "hello": "Hello",
  "welcome": "Welcome {name}"
}
`)
	writeBootstrapFile(t, root, "messages/fr.json", `{
  "hello": "Bonjour",
  "welcome": "Bienvenue {name}"
}
`)
	writeBootstrapFile(t, root, "app/page.tsx", `import {useTranslations} from 'next-intl'

export default function Page() {
  const t = useTranslations('common')
  return <h1>{t('hello')}</h1>
}
`)
	return root
}

func writeBootstrapFile(t *testing.T, root string, relPath string, contents string) {
	t.Helper()
	abs := filepath.Join(root, relPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(abs), 0o700))
	require.NoError(t, os.WriteFile(abs, []byte(contents), 0o600))
}

func proposedConfigFromDetect(t *testing.T, structured any) config.File {
	t.Helper()
	data, err := json.Marshal(structured)
	require.NoError(t, err)
	var out struct {
		ProposedConfig config.File `json:"proposedConfig"`
	}
	require.NoError(t, json.Unmarshal(data, &out))
	return out.ProposedConfig
}

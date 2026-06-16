package cli

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSchemaCommandPrintsConfigSchema(t *testing.T) {
	t.Setenv("I18N_MCP_PROJECT", "")
	t.Setenv("I18N_MCP_CONFIG", "")
	t.Setenv("I18N_MCP_LOG_LEVEL", "")
	t.Setenv("I18N_MCP_OUTPUT", "")

	opts := &RootOptions{}
	cmd := newRootCommand(opts)
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(new(bytes.Buffer))
	cmd.SetArgs([]string{"schema", "--project", t.TempDir()})

	require.NoError(t, cmd.Execute())

	var got map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	require.Equal(t, "https://example.com/i18n-mcp.schema.json", got["$id"])
	require.Equal(t, "object", got["type"])

	properties, ok := got["properties"].(map[string]any)
	require.True(t, ok)
	require.Contains(t, properties, "sourceLocale")
	require.Contains(t, properties, "targetLocales")
	require.Contains(t, properties, "localeFiles")
	require.Contains(t, properties, "format")
	require.Contains(t, properties, "translation")
}

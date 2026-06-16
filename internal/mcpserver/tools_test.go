package mcpserver_test

import (
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

func TestListToolsOverInMemoryMCP(t *testing.T) {
	ctx := t.Context()
	clientSession := newInMemoryMCPClientSession(t, ctx, t.TempDir())

	res, err := clientSession.ListTools(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, res.NextCursor)

	toolsByName := make(map[string]*mcp.Tool, len(res.Tools))
	for _, tool := range res.Tools {
		toolsByName[tool.Name] = tool
	}

	expected := map[string]string{
		"i18n.health":          "i18n MCP Health",
		"i18n.config.get":      "Get i18n MCP Config",
		"i18n.config.validate": "Validate i18n MCP Config",
	}
	for name, title := range expected {
		t.Run(name, func(t *testing.T) {
			tool, ok := toolsByName[name]
			require.True(t, ok, "tool %q is not registered", name)
			require.Equal(t, title, tool.Title)
			require.NotEmpty(t, tool.Description)
			require.NotNil(t, tool.InputSchema)
			require.NotNil(t, tool.OutputSchema)
			require.NotNil(t, tool.Annotations)
			require.Equal(t, title, tool.Annotations.Title)
			require.True(t, tool.Annotations.ReadOnlyHint)
			require.NotNil(t, tool.Annotations.OpenWorldHint)
			require.False(t, *tool.Annotations.OpenWorldHint)
		})
	}
}

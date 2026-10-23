package mcpserver_test

import (
	"encoding/json"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

func TestProductionServerDisablesDeadKeyFeatures(t *testing.T) {
	root := makeDeadKeyMCPFixture(t)
	before := readMCPDeadKeyFile(t, root, "messages/en/common.json")
	ctx, session := newTestClientSession(t, root)

	tools, err := session.ListTools(ctx, nil)
	require.NoError(t, err)
	toolNames := make([]string, 0, len(tools.Tools))
	for _, tool := range tools.Tools {
		toolNames = append(toolNames, tool.Name)
		if tool.Name == "i18n.report.generate" {
			schema, err := json.Marshal(tool.InputSchema)
			require.NoError(t, err)
			require.NotContains(t, string(schema), "refreshUsage")
		}
	}
	require.Len(t, toolNames, 13)
	require.Contains(t, toolNames, "i18n.keys.rename")
	for _, name := range []string{"i18n.keys.usage_scan", "i18n.keys.dead_report", "i18n.keys.prune"} {
		require.NotContains(t, toolNames, name)
		_, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name: name,
			Arguments: map[string]any{
				"apply": true,
				"keys":  []map[string]any{{"namespace": "common", "key": "unused"}},
			},
		})
		require.ErrorContains(t, err, "unknown tool")
	}
	require.Equal(t, before, readMCPDeadKeyFile(t, root, "messages/en/common.json"))

	resources, err := session.ListResources(ctx, nil)
	require.NoError(t, err)
	uris := make([]string, 0, len(resources.Resources))
	for _, resource := range resources.Resources {
		uris = append(uris, resource.URI)
	}
	for _, uri := range []string{"i18n://analysis/usage", "i18n://analysis/dead-keys"} {
		require.NotContains(t, uris, uri)
		_, err := session.ReadResource(ctx, &mcp.ReadResourceParams{URI: uri})
		require.Error(t, err)
	}

	prompts, err := session.ListPrompts(ctx, nil)
	require.NoError(t, err)
	require.Len(t, prompts.Prompts, 5)
	for _, prompt := range prompts.Prompts {
		require.NotEqual(t, "i18n_audit_dead_keys", prompt.Name)
	}
	_, err = session.GetPrompt(ctx, &mcp.GetPromptParams{Name: "i18n_audit_dead_keys"})
	require.Error(t, err)
}

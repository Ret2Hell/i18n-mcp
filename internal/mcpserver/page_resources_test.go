package mcpserver_test

import (
	"encoding/json"
	"testing"

	"github.com/Ret2Hell/i18n-mcp/internal/diff"
	"github.com/Ret2Hell/i18n-mcp/internal/locale"
	"github.com/Ret2Hell/i18n-mcp/internal/pagination"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

func TestResourceContinuationsAreBoundedAndComplete(t *testing.T) {
	ctx, session := newTestClientSession(t, makeMCPDiffFixture(t))
	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.keys.diff", Arguments: map[string]any{"limit": 2}})
	require.NoError(t, err)
	require.False(t, res.IsError)
	var out struct {
		Report diff.Report `json:"report"`
	}
	unmarshalStructuredContent(t, res.StructuredContent, &out)
	require.NotEmpty(t, out.Report.Page.NextResource)
	require.Less(t, len(res.Content[0].(*mcp.TextContent).Text), 200)
	resource, err := session.ReadResource(ctx, &mcp.ReadResourceParams{URI: out.Report.Page.NextResource})
	require.NoError(t, err)
	var page diff.Report
	require.NoError(t, json.Unmarshal([]byte(resource.Contents[0].Text), &page))
	require.Len(t, page.Items, 2)
	require.NotEqual(t, out.Report.Items[0].Key, page.Items[0].Key)
	raw, err := json.Marshal(resource)
	require.NoError(t, err)
	require.LessOrEqual(t, len(raw), pagination.ResponseBudget)

	res, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.locales.list", Arguments: map[string]any{"includeUnits": true, "limit": 2, "locales": []string{"en"}}})
	require.NoError(t, err)
	require.False(t, res.IsError)
	var locales struct {
		Inventory locale.Inventory `json:"inventory"`
	}
	unmarshalStructuredContent(t, res.StructuredContent, &locales)
	require.Equal(t, 5, locales.Inventory.Page.TotalMatched)
	total := len(locales.Inventory.Units)
	next := locales.Inventory.Page.NextResource
	for next != "" {
		resource, err = session.ReadResource(ctx, &mcp.ReadResourceParams{URI: next})
		require.NoError(t, err)
		var inv locale.Inventory
		require.NoError(t, json.Unmarshal([]byte(resource.Contents[0].Text), &inv))
		total += len(inv.Units)
		next = inv.Page.NextResource
	}
	require.Equal(t, 5, total)
}

package mcpserver_test

import (
	"github.com/Ret2Hell/i18n-mcp/internal/diff"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestDiffPagesAndFilters(t *testing.T) {
	root := makeMCPDiffFixture(t)
	ctx, session := newTestClientSession(t, root)
	args := map[string]any{"limit": 2, "locales": []string{"fr"}}
	var items []diff.KeyDiff
	var firstCursor string
	for {
		res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.keys.diff", Arguments: args})
		require.NoError(t, err)
		require.False(t, res.IsError)
		var out struct {
			Report diff.Report `json:"report"`
		}
		unmarshalStructuredContent(t, res.StructuredContent, &out)
		require.Equal(t, 6, out.Report.Summary.Total)
		require.NotNil(t, out.Report.Page)
		items = append(items, out.Report.Items...)
		if !out.Report.Page.Truncated {
			break
		}
		if firstCursor == "" {
			firstCursor = out.Report.Page.NextCursor
		}
		args["cursor"] = out.Report.Page.NextCursor
	}
	require.Len(t, items, 6)
	seen := map[string]bool{}
	for _, item := range items {
		require.False(t, seen[item.Key])
		seen[item.Key] = true
	}
	writeMCPDiffFile(t, root, "messages/fr.json", `{"current":"Changed"}`)
	args["cursor"] = firstCursor
	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.keys.diff", Arguments: args})
	require.NoError(t, err)
	require.True(t, res.IsError)
	res, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "i18n.keys.diff", Arguments: map[string]any{"statuses": []string{"missing"}, "keyPrefix": "missing"}})
	require.NoError(t, err)
	require.False(t, res.IsError)
	var out struct {
		Report diff.Report `json:"report"`
	}
	unmarshalStructuredContent(t, res.StructuredContent, &out)
	require.Equal(t, 2, out.Report.Summary.Missing)
	require.Len(t, out.Report.Items, 2)
	for _, item := range out.Report.Items {
		require.NotEmpty(t, item.SourceHash)
	}
}

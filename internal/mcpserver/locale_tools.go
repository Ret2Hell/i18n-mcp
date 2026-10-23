package mcpserver

import (
	"context"

	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/Ret2Hell/i18n-mcp/internal/locale"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// LocalesListInput is the input for the locales.list tool.
type LocalesListInput = locale.PageInput

// LocalesListOutput is the output for the locales.list tool.
type LocalesListOutput struct {
	Inventory locale.Inventory `json:"inventory" jsonschema:"locale inventory summary and optional flattened units"`
}

func localesListTool(a *app.App) func(context.Context, *mcp.CallToolRequest, LocalesListInput) (*mcp.CallToolResult, LocalesListOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in LocalesListInput) (*mcp.CallToolResult, LocalesListOutput, error) {
		inv, err := a.Locales.Page(ctx, a.RequestStateSigner, a.ProjectRoot, in)
		if err != nil {
			return nil, LocalesListOutput{}, err
		}
		if inv.Page != nil && inv.Page.Truncated {
			next := in
			next.Cursor = inv.Page.NextCursor
			inv.Page.NextResource = pageURI("locales", next)
		}
		return nil, LocalesListOutput{Inventory: inv}, nil
	}
}

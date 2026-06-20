package mcpserver

import (
	"context"

	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/Ret2Hell/i18n-mcp/internal/locale"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type LocalesListInput struct {
	IncludeUnits bool `json:"includeUnits,omitzero" jsonschema:"include flattened translation units in the response; false returns summary only"`
}

type LocalesListOutput struct {
	Inventory locale.Inventory `json:"inventory" jsonschema:"locale inventory summary and optional flattened units"`
}

func localesListTool(a *app.App) func(context.Context, *mcp.CallToolRequest, LocalesListInput) (*mcp.CallToolResult, LocalesListOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, in LocalesListInput) (*mcp.CallToolResult, LocalesListOutput, error) {
		_ = req
		inv, err := a.Locales.Inventory(ctx)
		if err != nil {
			return nil, LocalesListOutput{}, err
		}
		if !in.IncludeUnits {
			inv.Units = nil
		}
		return nil, LocalesListOutput{Inventory: inv}, nil
	}
}

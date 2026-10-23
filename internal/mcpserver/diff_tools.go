package mcpserver

import (
	"context"

	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/Ret2Hell/i18n-mcp/internal/diff"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// KeysDiffInput is the input for the keys.diff tool.
type KeysDiffInput = diff.PageInput

// KeysDiffOutput is the output for the keys.diff tool.
type KeysDiffOutput struct {
	Report diff.Report `json:"report" jsonschema:"locale key diff report with status records and summary counts"`
}

func keysDiffTool(a *app.App) func(context.Context, *mcp.CallToolRequest, KeysDiffInput) (*mcp.CallToolResult, KeysDiffOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in KeysDiffInput) (*mcp.CallToolResult, KeysDiffOutput, error) {
		report, err := a.Diff.Page(ctx, a.RequestStateSigner, a.ProjectRoot, in)
		if err != nil {
			return nil, KeysDiffOutput{}, err
		}
		return nil, KeysDiffOutput{Report: report}, nil
	}
}

package mcpserver

import (
	"context"

	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/Ret2Hell/i18n-mcp/internal/diff"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type KeysDiffInput struct{}

type KeysDiffOutput struct {
	Report diff.Report `json:"report" jsonschema:"locale key diff report with status records and summary counts"`
}

func keysDiffTool(a *app.App) func(context.Context, *mcp.CallToolRequest, KeysDiffInput) (*mcp.CallToolResult, KeysDiffOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, _ KeysDiffInput) (*mcp.CallToolResult, KeysDiffOutput, error) {
		report, err := a.Diff.Analyze(ctx)
		if err != nil {
			return nil, KeysDiffOutput{}, err
		}
		return nil, KeysDiffOutput{Report: report}, nil
	}
}

package mcpserver

import (
	"context"

	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/Ret2Hell/i18n-mcp/internal/report"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func reportGenerateTool(a *app.App) func(context.Context, *mcp.CallToolRequest, report.GenerateInput) (*mcp.CallToolResult, report.GenerateOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, in report.GenerateInput) (*mcp.CallToolResult, report.GenerateOutput, error) {
		_ = req
		out, err := a.Reports.Generate(ctx, in)
		if err != nil {
			return nil, report.GenerateOutput{}, err
		}
		return nil, out, nil
	}
}

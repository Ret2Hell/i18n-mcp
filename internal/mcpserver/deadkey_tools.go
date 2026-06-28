package mcpserver

import (
	"context"

	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/Ret2Hell/i18n-mcp/internal/deadkey"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type DeadReportOutput struct {
	Report deadkey.Report `json:"report" jsonschema:"dead-key classification report with evidence and confidence"`
}

func deadReportTool(a *app.App) func(context.Context, *mcp.CallToolRequest, deadkey.ReportInput) (*mcp.CallToolResult, DeadReportOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, in deadkey.ReportInput) (*mcp.CallToolResult, DeadReportOutput, error) {
		_ = req
		report, err := a.DeadKeys.Report(ctx, in)
		if err != nil {
			return nil, DeadReportOutput{}, err
		}
		return nil, DeadReportOutput{Report: report}, nil
	}
}

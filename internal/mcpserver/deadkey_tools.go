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

func keysPruneTool(a *app.App) func(context.Context, *mcp.CallToolRequest, deadkey.PruneInput) (*mcp.CallToolResult, deadkey.PruneOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, in deadkey.PruneInput) (*mcp.CallToolResult, deadkey.PruneOutput, error) {
		_ = req
		out, err := a.DeadKeys.Prune(ctx, in)
		if err != nil {
			return nil, deadkey.PruneOutput{}, err
		}
		if len(out.Rejected) > 0 {
			return &mcp.CallToolResult{IsError: true}, out, nil
		}
		return nil, out, nil
	}
}

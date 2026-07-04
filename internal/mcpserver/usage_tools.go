package mcpserver

import (
	"context"

	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/Ret2Hell/i18n-mcp/internal/mcpadapter"
	"github.com/Ret2Hell/i18n-mcp/internal/scanner"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type UsageScanOutput struct {
	Report scanner.Report `json:"report" jsonschema:"static translation key usage scan report"`
}

func usageScanTool(a *app.App) func(context.Context, *mcp.CallToolRequest, scanner.ScanInput) (*mcp.CallToolResult, UsageScanOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, in scanner.ScanInput) (*mcp.CallToolResult, UsageScanOutput, error) {
		in.Progress = mcpadapter.NewMCPProgressReporter(req, a.Logger)
		report, err := a.Scanner.Scan(ctx, in)
		if err != nil {
			return nil, UsageScanOutput{}, err
		}
		return nil, UsageScanOutput{Report: report}, nil
	}
}

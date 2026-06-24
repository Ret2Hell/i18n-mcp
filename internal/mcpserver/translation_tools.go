package mcpserver

import (
	"context"

	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/Ret2Hell/i18n-mcp/internal/translate"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func translationApplyTool(a *app.App) func(context.Context, *mcp.CallToolRequest, translate.ApplyInput) (*mcp.CallToolResult, translate.ApplyOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, in translate.ApplyInput) (*mcp.CallToolResult, translate.ApplyOutput, error) {
		_ = req
		out, err := a.Translation.Apply(ctx, in)
		if err != nil {
			return nil, translate.ApplyOutput{}, err
		}
		if len(out.Rejected) > 0 {
			return &mcp.CallToolResult{IsError: true}, out, nil
		}
		return nil, out, nil
	}
}

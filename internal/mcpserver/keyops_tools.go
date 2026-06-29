package mcpserver

import (
	"context"

	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/Ret2Hell/i18n-mcp/internal/keyops"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func keysRenameTool(a *app.App) func(context.Context, *mcp.CallToolRequest, keyops.RenameInput) (*mcp.CallToolResult, keyops.RenameOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, in keyops.RenameInput) (*mcp.CallToolResult, keyops.RenameOutput, error) {
		_ = req
		out, err := a.KeyOps.Rename(ctx, in)
		if err != nil {
			return nil, keyops.RenameOutput{}, err
		}
		if len(out.Conflicts) > 0 {
			return &mcp.CallToolResult{IsError: true}, out, nil
		}
		return nil, out, nil
	}
}

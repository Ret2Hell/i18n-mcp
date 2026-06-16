package mcpserver

import (
	"context"

	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/Ret2Hell/i18n-mcp/internal/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ConfigGetInput struct{}

type ConfigGetOutput struct {
	Config config.Resolved `json:"config" jsonschema:"resolved i18n MCP configuration"`
}

func configGetTool(a *app.App) func(context.Context, *mcp.CallToolRequest, ConfigGetInput) (*mcp.CallToolResult, ConfigGetOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, in ConfigGetInput) (*mcp.CallToolResult, ConfigGetOutput, error) {
		_, _ = req, in
		cfg, err := a.Config.Resolve(ctx)
		if err != nil {
			return nil, ConfigGetOutput{}, err
		}
		return nil, ConfigGetOutput{Config: cfg}, nil
	}
}

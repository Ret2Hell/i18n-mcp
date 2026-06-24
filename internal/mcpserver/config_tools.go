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
	return func(ctx context.Context, _ *mcp.CallToolRequest, _ ConfigGetInput) (*mcp.CallToolResult, ConfigGetOutput, error) {
		cfg, err := a.Config.Resolve(ctx)
		if err != nil {
			return nil, ConfigGetOutput{}, err
		}
		return nil, ConfigGetOutput{Config: cfg}, nil
	}
}

type ConfigValidateInput struct{}

type ConfigValidateOutput struct {
	Config     config.Resolved         `json:"config" jsonschema:"resolved configuration that was validated"`
	Validation config.ValidationResult `json:"validation" jsonschema:"validation result with errors and warnings"`
}

func configValidateTool(a *app.App) func(context.Context, *mcp.CallToolRequest, ConfigValidateInput) (*mcp.CallToolResult, ConfigValidateOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, _ ConfigValidateInput) (*mcp.CallToolResult, ConfigValidateOutput, error) {
		cfg, err := a.Config.Resolve(ctx)
		if err != nil {
			return nil, ConfigValidateOutput{}, err
		}
		validation := a.Config.Validate(ctx, cfg)
		return nil, ConfigValidateOutput{Config: cfg, Validation: validation}, nil
	}
}

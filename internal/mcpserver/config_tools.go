package mcpserver

import (
	"context"

	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/Ret2Hell/i18n-mcp/internal/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ConfigGetInput is the input for the config.get tool.
type ConfigGetInput struct{}

// ConfigGetOutput is the output for the config.get tool.
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

// ConfigValidateInput is the input for the config.validate tool.
type ConfigValidateInput struct{}

// ConfigValidateOutput is the output for the config.validate tool.
type ConfigValidateOutput struct {
	Config     config.Resolved         `json:"config" jsonschema:"resolved configuration that was validated"`
	Validation config.ValidationResult `json:"validation" jsonschema:"validation result with errors and warnings"`
}

// ConfigWriteOutput is the output for the config.write tool.
type ConfigWriteOutput struct {
	Result config.WriteOutput `json:"result" jsonschema:"config write preview or apply result"`
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

func configWriteTool(a *app.App) func(context.Context, *mcp.CallToolRequest, config.WriteInput) (*mcp.CallToolResult, ConfigWriteOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in config.WriteInput) (*mcp.CallToolResult, ConfigWriteOutput, error) {
		result, err := a.Config.Write(ctx, in)
		if err != nil {
			return nil, ConfigWriteOutput{}, err
		}
		out := ConfigWriteOutput{Result: result}
		if !result.Validation.Valid {
			return new(mcp.CallToolResult{IsError: true}), out, nil
		}
		return nil, out, nil
	}
}

package mcpserver

import (
	"context"
	"fmt"

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
		canonical := in
		canonical.Apply = false
		canonical.DryRun = nil
		canonical.PreviewToken = ""
		canonical.BeforeWrite = nil
		result, err := a.Config.Write(ctx, canonical)
		if err != nil {
			return nil, ConfigWriteOutput{}, err
		}
		token := ""
		if !in.DryRunValue() {
			token = in.PreviewToken
		}
		view, err := a.Previews.Prepare(ctx, "i18n.config.write", canonical, []config.ChangedFile{result.ChangedFile}, nil, token)
		if err != nil {
			return nil, ConfigWriteOutput{}, err
		}
		compact := result
		compact.ChangedFile = view.Files[0]
		compact.Preview = view.Metadata
		out := ConfigWriteOutput{Result: compact}
		if err := checkWritePreview(out); err != nil {
			return nil, ConfigWriteOutput{}, err
		}
		if !result.Validation.Valid {
			return &mcp.CallToolResult{IsError: true}, out, nil
		}
		if in.DryRunValue() {
			return nil, out, nil
		}
		if err := requireReview(view, in.PreviewToken); err != nil {
			return writePreviewError(err), out, nil
		}
		in.BeforeWrite = func(ctx context.Context, actual config.WriteOutput) error {
			return a.Previews.Verify(ctx, "i18n.config.write", canonical, []config.ChangedFile{actual.ChangedFile}, nil, view.Metadata.Token)
		}
		previous := result
		result, err = a.Config.Write(ctx, in)
		result.ChangedFile = reviewedFiles([]config.ChangedFile{result.ChangedFile}, view)[0]
		result.Preview = view.Metadata
		result.Warnings = boundWriteWarnings(result.Warnings, view)
		if !result.Validation.Valid && err == nil {
			err = fmt.Errorf("configuration no longer validates; request a new preview")
		}
		if err != nil && !result.Applied && checkWritePreview(ConfigWriteOutput{Result: result}) != nil {
			result = previous
			view.Metadata.DiagnosticsTruncated = true
		}
		out.Result = result
		if err != nil {
			return writePreviewError(err), out, nil
		}
		return nil, out, nil
	}
}

package mcpserver

import (
	"context"

	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/Ret2Hell/i18n-mcp/internal/state"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// StateRebuildInput is the input for the state.rebuild tool.
type StateRebuildInput struct {
	Apply bool `json:"apply,omitzero" jsonschema:"write .i18n-mcp/state.json when true; false previews only"`
}

// StateRebuildOutput is the output for the state.rebuild tool.
type StateRebuildOutput struct {
	Result state.RebuildResult `json:"result" jsonschema:"state rebuild preview result"`
}

func stateRebuildTool(a *app.App) func(context.Context, *mcp.CallToolRequest, StateRebuildInput) (*mcp.CallToolResult, StateRebuildOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in StateRebuildInput) (*mcp.CallToolResult, StateRebuildOutput, error) {
		result, err := a.State.Rebuild(ctx, state.RebuildOptions{Apply: in.Apply})
		if err != nil {
			return nil, StateRebuildOutput{}, err
		}
		return nil, StateRebuildOutput{Result: result}, nil
	}
}

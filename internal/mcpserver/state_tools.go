package mcpserver

import (
	"context"

	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/Ret2Hell/i18n-mcp/internal/state"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type StateRebuildInput struct{}

type StateRebuildOutput struct {
	Result state.RebuildResult `json:"result" jsonschema:"state rebuild preview result"`
}

func stateRebuildTool(a *app.App) func(context.Context, *mcp.CallToolRequest, StateRebuildInput) (*mcp.CallToolResult, StateRebuildOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, in StateRebuildInput) (*mcp.CallToolResult, StateRebuildOutput, error) {
		_, _ = req, in
		result, err := a.State.Rebuild(ctx, state.RebuildOptions{Apply: false})
		if err != nil {
			return nil, StateRebuildOutput{}, err
		}
		return nil, StateRebuildOutput{Result: result}, nil
	}
}

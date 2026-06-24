package mcpserver

import (
	"context"

	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/Ret2Hell/i18n-mcp/internal/version"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type HealthInput struct{}

type HealthOutput struct {
	Name        string       `json:"name" jsonschema:"server name"`
	Version     version.Info `json:"version" jsonschema:"build version information"`
	ProjectRoot string       `json:"projectRoot" jsonschema:"configured project root"`
	Status      string       `json:"status" jsonschema:"health status"`
}

func healthTool(a *app.App) func(context.Context, *mcp.CallToolRequest, HealthInput) (*mcp.CallToolResult, HealthOutput, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, _ HealthInput) (*mcp.CallToolResult, HealthOutput, error) {
		return nil, HealthOutput{
			Name:        "i18n-mcp",
			Version:     version.Get(),
			ProjectRoot: a.ProjectRoot,
			Status:      "ok",
		}, nil
	}
}

package mcpserver

import "github.com/modelcontextprotocol/go-sdk/mcp"

func readOnly(title string) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{
		Title:         title,
		ReadOnlyHint:  true,
		OpenWorldHint: new(false),
	}
}

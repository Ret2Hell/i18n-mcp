package mcpserver

import "github.com/modelcontextprotocol/go-sdk/mcp"

func readOnly(title string) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{
		Title:         title,
		ReadOnlyHint:  true,
		OpenWorldHint: new(false),
	}
}

func writeOp(title string, destructive bool, idempotent bool) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{
		Title:           title,
		ReadOnlyHint:    false,
		DestructiveHint: new(destructive),
		IdempotentHint:  idempotent,
		OpenWorldHint:   new(false),
	}
}

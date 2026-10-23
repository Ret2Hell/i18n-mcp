package mcpserver

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// workLimit admits one expensive operation per server and rejects excess work
// rather than accumulating an unbounded queue. Health and catalogs stay usable.
func workLimit() mcp.Middleware {
	slots := make(chan struct{}, 1)
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			expensive := method == "resources/read"
			if params, ok := req.GetParams().(*mcp.CallToolParamsRaw); ok {
				expensive = params.Name != "i18n.health"
			}
			if !expensive {
				return next(ctx, method, req)
			}
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			select {
			case slots <- struct{}{}:
				defer func() { <-slots }()
			default:
				if method == "tools/call" {
					return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: "Server is busy with another project operation; retry this request after it completes."}}}, nil
				}
				return nil, fmt.Errorf("server is busy with another project operation; retry after it completes")
			}
			return next(ctx, method, req)
		}
	}
}

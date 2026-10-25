package mcpserver

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/Ret2Hell/i18n-mcp/internal/pagination"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// boundedResults keeps structured data canonical rather than duplicating it as
// JSON text. The hard read-response limit also covers metadata and JSON escaping.
func boundedResults(next mcp.MethodHandler) mcp.MethodHandler {
	return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
		result, err := next(ctx, method, req)
		if err != nil || result == nil {
			return result, err
		}
		value, ok := result.(*mcp.CallToolResult)
		if !ok {
			return result, nil
		}
		if value.StructuredContent != nil && len(value.Content) == 1 {
			if text, ok := value.Content[0].(*mcp.TextContent); ok && strings.HasPrefix(strings.TrimSpace(text.Text), "{") {
				value.Content = []mcp.Content{&mcp.TextContent{Text: "Result is in structuredContent; follow page.nextCursor or page.nextResource when truncated."}}
			}
		}
		params, ok := req.GetParams().(*mcp.CallToolParamsRaw)
		if !ok {
			return result, nil
		}
		switch params.Name {
		case "i18n.keys.diff", "i18n.locales.list", "i18n.translation.plan", "i18n.report.generate":
			raw, err := json.Marshal(value)
			if err != nil {
				return nil, err
			}
			if len(raw) > pagination.ResponseBudget {
				return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: "Result exceeds the 128 KiB response budget. Reduce limit, narrow filters, or use a local CLI export. No results were silently skipped."}}}, nil
			}
		}
		return result, nil
	}
}

package mcpserver

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const catalogCacheTTLMs = 5 * 60 * 1000

func cacheHints(next mcp.MethodHandler) mcp.MethodHandler {
	return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
		result, err := next(ctx, method, req)
		if err != nil || result == nil {
			return result, err
		}
		switch value := result.(type) {
		case *mcp.DiscoverResult:
			setPublicCatalogCache(&value.Cacheable)
		case *mcp.ListToolsResult:
			setPublicCatalogCache(&value.Cacheable)
		case *mcp.ListPromptsResult:
			setPublicCatalogCache(&value.Cacheable)
		case *mcp.ListResourcesResult:
			setPublicCatalogCache(&value.Cacheable)
		case *mcp.ListResourceTemplatesResult:
			setPublicCatalogCache(&value.Cacheable)
		case *mcp.ReadResourceResult:
			value.TTLMs = 0
			value.CacheScope = "private"
		}
		return result, nil
	}
}

func setPublicCatalogCache(cacheable *mcp.Cacheable) {
	cacheable.TTLMs = catalogCacheTTLMs
	cacheable.CacheScope = "public"
}

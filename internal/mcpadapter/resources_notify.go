package mcpadapter

import (
	"context"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ResourceNotifier sends MCP resource update notifications.
type ResourceNotifier struct {
	Server *mcp.Server
	Logger *slog.Logger
}

// Updated notifies subscribed clients that the given resource URIs changed.
func (n ResourceNotifier) Updated(ctx context.Context, uris ...string) {
	if n.Server == nil {
		return
	}
	for _, uri := range uniqueResourceURIs(uris) {
		if err := n.Server.ResourceUpdated(ctx, &mcp.ResourceUpdatedNotificationParams{URI: uri}); err != nil && n.Logger != nil {
			n.Logger.Debug("sending resource update", "uri", uri, "error", err)
		}
	}
}

func uniqueResourceURIs(uris []string) []string {
	seen := make(map[string]struct{}, len(uris))
	out := make([]string, 0, len(uris))
	for _, uri := range uris {
		if uri == "" {
			continue
		}
		if _, ok := seen[uri]; ok {
			continue
		}
		seen[uri] = struct{}{}
		out = append(out, uri)
	}
	return out
}

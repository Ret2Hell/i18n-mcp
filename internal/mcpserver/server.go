package mcpserver

import (
	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/Ret2Hell/i18n-mcp/internal/mcpadapter"
	"github.com/Ret2Hell/i18n-mcp/internal/version"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// New creates and configures the MCP server.
func New(a *app.App) *mcp.Server {
	opts := &mcp.ServerOptions{
		Instructions: "Use this server to inspect, validate, translate, and safely update JSON i18n locale files in the configured Next.js project. Prefer dry-run tools before write tools.",
		Logger:       a.Logger,
		PageSize:     100,
		Capabilities: &mcp.ServerCapabilities{
			Completions: &mcp.CompletionCapabilities{},
		},
		CompletionHandler:  complete(a),
		SubscribeHandler:   mcpadapter.ValidateSubscribe,
		UnsubscribeHandler: mcpadapter.ValidateUnsubscribe,
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "i18n-mcp",
		Title:   "i18n MCP Server",
		Version: version.Version,
	}, opts)
	server.AddReceivingMiddleware(cacheHints)
	notifier := mcpadapter.ResourceNotifier{Server: server, Logger: a.Logger}
	a.Translation.Notifier = notifier
	a.DeadKeys.Notifier = notifier
	a.KeyOps.Notifier = notifier
	a.Reports.Notifier = notifier

	registerTools(server, a)
	registerResources(server, a)
	registerPrompts(server, a)
	return server
}

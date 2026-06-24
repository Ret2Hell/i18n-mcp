package mcpserver

import (
	"context"
	"log/slog"

	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/Ret2Hell/i18n-mcp/internal/version"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func New(a *app.App) *mcp.Server {
	opts := &mcp.ServerOptions{
		Instructions: "Use this server to inspect, validate, translate, and safely update JSON i18n locale files in the configured project. Prefer dry-run tools before write tools.",
		Logger:       a.Logger,
		PageSize:     100,
		Capabilities: &mcp.ServerCapabilities{
			Logging: &mcp.LoggingCapabilities{},
		},
		InitializedHandler: func(_ context.Context, req *mcp.InitializedRequest) {
			a.Logger.Info("mcp session initialized", slog.String("session", req.Session.ID()))
		},
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "i18n-mcp",
		Title:   "i18n MCP Server",
		Version: version.Version,
	}, opts)

	registerTools(server, a)
	registerResources(server, a)
	return server
}

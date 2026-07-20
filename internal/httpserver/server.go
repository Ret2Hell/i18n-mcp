// Package httpserver serves MCP over Streamable HTTP.
package httpserver

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// AuthConfig reserves HTTP authentication configuration for authenticated
// non-loopback deployments.
type AuthConfig struct {
	Required   bool
	Middleware func(http.Handler) http.Handler
}

// Config configures the Streamable HTTP server.
type Config struct {
	Addr           string
	MCPPath        string
	ProjectRoot    string
	SessionTimeout time.Duration
	JSONResponse   bool
	Auth           AuthConfig
}

// AppFactory creates an MCP server for an HTTP request.
type AppFactory interface {
	ServerForRequest(req *http.Request) *mcp.Server
}

// Run serves MCP and health endpoints until ctx is canceled or serving fails.
func Run(ctx context.Context, cfg Config, app AppFactory, logger *slog.Logger) error {
	cfg.Addr = cmp.Or(cfg.Addr, "127.0.0.1:7339")
	cfg.MCPPath = cmp.Or(cfg.MCPPath, "/mcp")
	cfg.SessionTimeout = cmp.Or(cfg.SessionTimeout, 30*time.Minute)
	if err := validateBinding(cfg); err != nil {
		return err
	}

	var mcpHandler http.Handler = mcp.NewStreamableHTTPHandler(app.ServerForRequest, &mcp.StreamableHTTPOptions{
		SessionTimeout: cfg.SessionTimeout,
		JSONResponse:   cfg.JSONResponse,
		Logger:         logger,
	})

	mux := http.NewServeMux()
	if cfg.Auth.Required {
		mcpHandler = cfg.Auth.Middleware(mcpHandler)
	}
	mux.Handle(cfg.MCPPath, mcpHandler)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil && logger != nil {
			logger.Warn("HTTP server shutdown failed", "error", err)
		}
	}()

	err := server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func validateBinding(cfg Config) error {
	host, _, err := net.SplitHostPort(cfg.Addr)
	if err != nil {
		return fmt.Errorf("invalid HTTP listen address %q: %w", cfg.Addr, err)
	}
	ip := net.ParseIP(host)
	isLoopback := host == "localhost" || (ip != nil && ip.IsLoopback())
	if isLoopback {
		return nil
	}
	if cfg.Auth.Required && cfg.Auth.Middleware != nil {
		return nil
	}
	return fmt.Errorf("refusing non-loopback HTTP listen address %q without required authentication", cfg.Addr)
}

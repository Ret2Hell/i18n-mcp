// Package httpserver serves MCP over Streamable HTTP.
package httpserver

import (
	"cmp"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

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
	cfg.Auth.MetadataPath = cmp.Or(cfg.Auth.MetadataPath, DefaultAuthConfig().MetadataPath)
	cfg.Auth.MetadataURL = inferMetadataURL(cfg)
	if err := cfg.Auth.Validate(cfg.Addr); err != nil {
		return err
	}

	mcpHandler := mcp.NewStreamableHTTPHandler(app.ServerForRequest, &mcp.StreamableHTTPOptions{
		SessionTimeout: cfg.SessionTimeout,
		JSONResponse:   cfg.JSONResponse,
		Logger:         logger,
	})

	mux := http.NewServeMux()
	if cfg.Auth.ResourceURL != "" {
		mux.Handle(cfg.Auth.MetadataPath, ProtectedResourceHandler(cfg.Auth))
	}

	handler := http.Handler(mcpHandler)
	if cfg.Auth.Required {
		verifierFactory := TokenVerifierFactory(DevStaticTokenVerifier)
		verifier, err := verifierFactory(cfg.Auth)
		if err != nil {
			return err
		}
		handler = ProtectMCPHandler(handler, cfg.Auth, verifier)
	}

	mux.Handle(cfg.MCPPath, handler)
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

func inferMetadataURL(cfg Config) string {
	if cfg.Auth.MetadataURL != "" {
		return cfg.Auth.MetadataURL
	}
	if cfg.Auth.ResourceURL == "" || cfg.Auth.MetadataPath == "" {
		return ""
	}
	return strings.TrimRight(cfg.Auth.ResourceURL, "/") + cfg.Auth.MetadataPath
}

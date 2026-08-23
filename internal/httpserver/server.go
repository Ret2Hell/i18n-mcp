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
	Addr         string
	MCPPath      string
	JSONResponse bool
	Auth         AuthConfig
}

// ServerProvider returns the MCP server used to handle HTTP requests.
type ServerProvider interface {
	ServerForRequest(req *http.Request) *mcp.Server
}

// Run serves MCP and health endpoints until ctx is canceled or serving fails.
func Run(ctx context.Context, cfg Config, provider ServerProvider, logger *slog.Logger) error {
	cfg = configWithDefaults(cfg)
	handler, err := newHandler(cfg, provider, logger)
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
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

	err = server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func configWithDefaults(cfg Config) Config {
	cfg.Addr = cmp.Or(cfg.Addr, "127.0.0.1:7339")
	cfg.MCPPath = cmp.Or(cfg.MCPPath, "/mcp")
	cfg.Auth.MetadataPath = cmp.Or(cfg.Auth.MetadataPath, DefaultAuthConfig().MetadataPath)
	cfg.Auth.MetadataURL = inferMetadataURL(cfg)
	return cfg
}

func newHandler(cfg Config, provider ServerProvider, logger *slog.Logger) (http.Handler, error) {
	cfg = configWithDefaults(cfg)
	if err := cfg.Auth.Validate(cfg.Addr); err != nil {
		return nil, err
	}

	mcpHandler := mcp.NewStreamableHTTPHandler(provider.ServerForRequest, &mcp.StreamableHTTPOptions{
		Stateless:                    true,
		JSONResponse:                 cfg.JSONResponse,
		Logger:                       logger,
		PropagateRequestCancellation: true,
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
			return nil, err
		}
		handler = ProtectMCPHandler(handler, cfg.Auth, verifier)
	}

	mux.Handle(cfg.MCPPath, handler)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	return mux, nil
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

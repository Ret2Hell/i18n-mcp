package cli

import (
	"context"
	"errors"
	"net/http"
	"os"

	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/Ret2Hell/i18n-mcp/internal/httpserver"
	"github.com/Ret2Hell/i18n-mcp/internal/mcpadapter"
	"github.com/Ret2Hell/i18n-mcp/internal/mcpserver"
	"github.com/Ret2Hell/i18n-mcp/internal/watch"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

func newServeCommand(opts *RootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run the MCP server",
	}
	cmd.AddCommand(newServeStdioCommand(opts))
	cmd.AddCommand(newServeHTTPCommand(opts))
	return cmd
}

func newServeHTTPCommand(opts *RootOptions) *cobra.Command {
	defaults := httpserver.DefaultAuthConfig()
	var addr string
	var path string
	var authRequired bool
	var resourceURL string
	var authScopes []string
	var authIssuers []string
	var devTokenEnv string
	cmd := &cobra.Command{
		Use:   "http",
		Short: "Run the MCP server over Streamable HTTP",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			application, err := app.New(cmd.Context(), app.Options{
				ProjectRoot: opts.Project,
				ConfigPath:  opts.Config,
				LogLevel:    opts.LogLevel,
			})
			if err != nil {
				return err
			}
			cfg := httpserver.Config{
				Addr:        addr,
				MCPPath:     path,
				ProjectRoot: opts.Project,
				Auth: httpserver.AuthConfig{
					Required:             authRequired,
					ResourceURL:          resourceURL,
					ResourceName:         defaults.ResourceName,
					MetadataPath:         defaults.MetadataPath,
					RequiredScopes:       authScopes,
					AuthorizationServers: authIssuers,
					DevStaticTokenEnv:    devTokenEnv,
				},
			}
			return httpserver.Run(cmd.Context(), cfg, serverFactory{application: application}, application.Logger)
		},
	}
	cmd.Flags().StringVar(&addr, "addr", "127.0.0.1:7339", "HTTP listen address")
	cmd.Flags().StringVar(&path, "path", "/mcp", "HTTP MCP endpoint path")
	cmd.Flags().BoolVar(&authRequired, "auth-required", false, "require bearer token auth for HTTP MCP requests")
	cmd.Flags().StringVar(&resourceURL, "auth-resource", "", "protected resource URL for bearer auth metadata")
	cmd.Flags().StringSliceVar(&authScopes, "auth-scope", defaults.RequiredScopes, "required bearer token scopes")
	cmd.Flags().StringSliceVar(&authIssuers, "auth-issuer", nil, "authorization server issuer URLs")
	cmd.Flags().StringVar(&devTokenEnv, "dev-static-token-env", "", "environment variable containing a development bearer token")
	return cmd
}

type serverFactory struct {
	application *app.App
}

func (f serverFactory) ServerForRequest(_ *http.Request) *mcp.Server {
	return mcpserver.New(f.application)
}

func newServeStdioCommand(opts *RootOptions) *cobra.Command {
	var watchLocales bool
	cmd := &cobra.Command{
		Use:   "stdio",
		Short: "Run the MCP server over stdin/stdout",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStdio(cmd.Context(), opts, watchLocales)
		},
	}
	cmd.Flags().BoolVar(&watchLocales, "watch", false, "watch locale files and emit MCP resource update notifications")
	return cmd
}

func runStdio(ctx context.Context, opts *RootOptions, watchLocales bool) error {
	application, err := app.New(ctx, app.Options{
		ProjectRoot: opts.Project,
		ConfigPath:  opts.Config,
		LogLevel:    opts.LogLevel,
	})
	if err != nil {
		return err
	}

	server := mcpserver.New(application)
	if watchLocales {
		if err := startLocaleWatcher(ctx, application, mcpadapter.ResourceNotifier{Server: server, Logger: application.Logger}); err != nil {
			return err
		}
	}
	transport := &mcp.LoggingTransport{
		Transport: &mcp.StdioTransport{},
		Writer:    os.Stderr,
	}
	return server.Run(ctx, transport)
}

func startLocaleWatcher(ctx context.Context, application *app.App, notifier mcpadapter.ResourceNotifier) error {
	cfg, err := application.Config.Resolve(ctx)
	if err != nil {
		return err
	}
	dirs, err := watch.GuardedLocaleDirs(ctx, application.Guard, application.Locales)
	if err != nil {
		return err
	}
	localeWatcher, err := watch.NewLocaleWatcher(notifier)
	if err != nil {
		return err
	}
	for _, dir := range dirs {
		if err := localeWatcher.AddDir(dir); err != nil {
			return err
		}
	}
	mapper := watch.NewLocaleMapper(application.Guard, cfg)
	go func() {
		if err := localeWatcher.Run(ctx, mapper); err != nil && !errors.Is(err, context.Canceled) {
			application.Logger.Warn("locale watcher stopped", "error", err)
		}
	}()
	return nil
}

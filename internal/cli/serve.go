package cli

import (
	"context"
	"errors"
	"os"

	"github.com/Ret2Hell/i18n-mcp/internal/app"
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
	return cmd
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

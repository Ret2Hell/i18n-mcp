package cli

import (
	"context"
	"os"

	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/Ret2Hell/i18n-mcp/internal/mcpserver"
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
	return &cobra.Command{
		Use:   "stdio",
		Short: "Run the MCP server over stdin/stdout",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStdio(cmd.Context(), opts)
		},
	}
}

func runStdio(ctx context.Context, opts *RootOptions) error {
	application, err := app.New(ctx, app.Options{
		ProjectRoot: opts.Project,
		ConfigPath:  opts.Config,
		LogLevel:    opts.LogLevel,
	})
	if err != nil {
		return err
	}

	server := mcpserver.New(application)
	transport := &mcp.LoggingTransport{
		Transport: &mcp.StdioTransport{},
		Writer:    os.Stderr,
	}
	return server.Run(ctx, transport)
}

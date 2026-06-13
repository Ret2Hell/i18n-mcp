package cli

import (
	"context"

	"github.com/spf13/cobra"
)

type RootOptions struct {
	Project  string
	Config   string
	LogLevel string
	Output   string
}

func Execute(ctx context.Context) error {
	opts := &RootOptions{}
	cmd := newRootCommand(ctx, opts)
	return cmd.ExecuteContext(ctx)
}

func newRootCommand(ctx context.Context, opts *RootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "i18n-mcp",
		Short:         "MCP server for JSON i18n locale management",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.PersistentFlags().StringVar(&opts.Project, "project", ".", "Next.js project root")
	cmd.PersistentFlags().StringVar(&opts.Config, "config", "", "optional .i18n-mcp.json path")
	cmd.PersistentFlags().StringVar(&opts.LogLevel, "log-level", "warn", "debug, info, warn, error")
	cmd.PersistentFlags().StringVar(&opts.Output, "output", "table", "table, json, markdown")

	return cmd
}

func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	return 1
}

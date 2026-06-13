package cli

import (
	"context"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           "i18n-mcp",
	Short:         "MCP server for JSON i18n locale management",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute(ctx context.Context) error {
	return rootCmd.ExecuteContext(ctx)
}

func init() {
}

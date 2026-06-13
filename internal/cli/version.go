package cli

import (
	"encoding/json"
	"fmt"

	"github.com/Ret2Hell/i18n-mcp/internal/version"
	"github.com/spf13/cobra"
)

func newVersionCommand(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			info := version.Get()
			switch opts.Output {
			case "json":
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(info)
			case "markdown":
				_, err := fmt.Fprintf(cmd.OutOrStdout(), "| Field | Value |\n| --- | --- |\n| Version | `%s` |\n| Commit | `%s` |\n| Built | `%s` |\n", info.Version, info.Commit, info.Date)
				return err
			default:
				_, err := fmt.Fprintf(cmd.OutOrStdout(), "i18n-mcp\n  version  %s\n  commit   %s\n  built    %s\n", info.Version, info.Commit, info.Date)
				return err
			}
		},
	}
}

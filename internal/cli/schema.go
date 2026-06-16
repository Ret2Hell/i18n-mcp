package cli

import (
	"encoding/json"

	"github.com/Ret2Hell/i18n-mcp/internal/config"
	"github.com/spf13/cobra"
)

func newSchemaCommand(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "schema",
		Short: "Print the .i18n-mcp.json JSON Schema",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = opts
			schema, err := config.Schema()
			if err != nil {
				return err
			}
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(schema)
		},
	}
}

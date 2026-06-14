package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/Ret2Hell/i18n-mcp/internal/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type RootOptions struct {
	Project  string
	Config   string
	LogLevel string
	Output   string
}

func Execute(ctx context.Context) error {
	opts := &RootOptions{}
	cmd := newRootCommand(opts)
	return cmd.ExecuteContext(ctx)
}

func newRootCommand(opts *RootOptions) *cobra.Command {
	v := viper.New()

	cmd := &cobra.Command{
		Use:           version.AppName,
		Short:         "MCP server for JSON i18n locale management",
		Version:       version.Get().Version,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return initConfig(v, opts)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.PersistentFlags().StringVar(&opts.Project, "project", ".", "Next.js project root")
	cmd.PersistentFlags().StringVar(&opts.Config, "config", "", "optional .i18n-mcp.json path")
	cmd.PersistentFlags().StringVar(&opts.LogLevel, "log-level", "warn", "debug, info, warn, error")
	cmd.SetVersionTemplate("{{.Name}} {{.Version}}\n")

	cmd.PersistentFlags().StringVar(&opts.Output, "output", "table", "table, json, markdown")

	mustBind(v.BindPFlag("project", cmd.PersistentFlags().Lookup("project")))
	mustBind(v.BindPFlag("config", cmd.PersistentFlags().Lookup("config")))
	mustBind(v.BindPFlag("log-level", cmd.PersistentFlags().Lookup("log-level")))
	mustBind(v.BindPFlag("output", cmd.PersistentFlags().Lookup("output")))

	cmd.AddCommand(newServeCommand(opts))
	cmd.AddCommand(newVersionCommand(opts))

	return cmd
}

func initConfig(v *viper.Viper, opts *RootOptions) error {
	v.SetEnvPrefix("I18N_MCP")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()

	opts.Project = v.GetString("project")
	opts.Config = v.GetString("config")
	opts.LogLevel = v.GetString("log-level")
	opts.Output = v.GetString("output")

	if opts.Project == "" {
		return fmt.Errorf("project root is required")
	}
	if _, err := os.Stat(opts.Project); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("project root does not exist: %s", opts.Project)
		}
		return err
	}
	return nil
}

func mustBind(err error) {
	if err != nil {
		panic(err)
	}
}

func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	return 1
}

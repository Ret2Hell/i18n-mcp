package cli

import (
	"errors"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestNewRootCommandSilencesUsageAndErrors(t *testing.T) {
	cmd := newRootCommand(&RootOptions{})
	require.True(t, cmd.SilenceUsage)
	require.True(t, cmd.SilenceErrors)
}

func TestInitConfigRequiresExistingProjectRoot(t *testing.T) {
	t.Setenv("I18N_MCP_PROJECT", "")
	t.Setenv("I18N_MCP_CONFIG", "")
	t.Setenv("I18N_MCP_LOG_LEVEL", "")
	t.Setenv("I18N_MCP_OUTPUT", "")

	missingRoot := t.TempDir() + "/missing"
	opts := &RootOptions{}
	v := viper.New()
	v.SetDefault("project", missingRoot)

	err := initConfig(v, opts)
	require.Error(t, err)
	require.Contains(t, err.Error(), "project root does not exist")
}

func TestInitConfigAcceptsExistingProjectRoot(t *testing.T) {
	t.Setenv("I18N_MCP_PROJECT", "")
	t.Setenv("I18N_MCP_CONFIG", "")
	t.Setenv("I18N_MCP_LOG_LEVEL", "")
	t.Setenv("I18N_MCP_OUTPUT", "")

	root := t.TempDir()
	opts := &RootOptions{}
	v := viper.New()
	v.SetDefault("project", root)
	v.SetDefault("config", ".i18n-mcp.test.json")
	v.SetDefault("log-level", "debug")
	v.SetDefault("output", "json")

	require.NoError(t, initConfig(v, opts))
	require.Equal(t, root, opts.Project)
	require.Equal(t, ".i18n-mcp.test.json", opts.Config)
	require.Equal(t, "debug", opts.LogLevel)
	require.Equal(t, "json", opts.Output)
}

func TestExitCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "nil", err: nil, want: 0},
		{name: "error", err: errors.New("boom"), want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, ExitCode(tt.err))
		})
	}
}

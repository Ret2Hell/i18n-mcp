package cli

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVersionCommandJSON(t *testing.T) {
	opts := &RootOptions{}
	cmd := newRootCommand(opts)
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(new(bytes.Buffer))
	cmd.SetArgs([]string{"version", "--output", "json"})

	require.NoError(t, cmd.Execute())

	var got map[string]string
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	require.NotEmpty(t, got["version"])
	require.NotContains(t, got, "dirty")
}

func TestVersionCommandText(t *testing.T) {
	opts := &RootOptions{}
	cmd := newRootCommand(opts)
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(new(bytes.Buffer))
	cmd.SetArgs([]string{"version"})

	require.NoError(t, cmd.Execute())
	require.Contains(t, buf.String(), "i18n-mcp\n")
	require.Contains(t, buf.String(), "  version  ")
	require.Contains(t, buf.String(), "  commit   ")
	require.Contains(t, buf.String(), "  built    ")
	require.NotContains(t, buf.String(), "dirty")
}

func TestVersionCommandMarkdown(t *testing.T) {
	opts := &RootOptions{}
	cmd := newRootCommand(opts)
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(new(bytes.Buffer))
	cmd.SetArgs([]string{"version", "--output", "markdown"})

	require.NoError(t, cmd.Execute())
	require.Contains(t, buf.String(), "| Field | Value |")
	require.Contains(t, buf.String(), "| Version | `")
	require.Contains(t, buf.String(), "| Commit | `")
	require.Contains(t, buf.String(), "| Built | `")
	require.NotContains(t, buf.String(), "dirty")
}

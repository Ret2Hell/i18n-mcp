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
}

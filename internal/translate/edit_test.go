package translate

import (
	"testing"

	"github.com/Ret2Hell/i18n-mcp/internal/config"
	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
	"github.com/stretchr/testify/require"
)

func TestJSONEditCreatesMissingTargetFileDocument(t *testing.T) {
	guard, err := fsutil.NewGuard(t.TempDir())
	require.NoError(t, err)
	svc := &Service{guard: guard}
	cfg := config.Resolved{File: config.File{Format: config.FormatConfig{Indent: 4, TrailingNewline: false}}}

	before, doc, err := svc.readEditableJSON("missing.json", cfg)
	require.NoError(t, err)
	require.Equal(t, []byte("{}\n"), before)

	require.NoError(t, doc.SetString([]string{"hello"}, "world"))
	after, err := doc.Render()
	require.NoError(t, err)
	require.Equal(t, "{\n    \"hello\": \"world\"\n}", string(after))
}

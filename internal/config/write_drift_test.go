package config

import (
	"context"
	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

func TestConfigWriteChecksActualFileAfterApprovalHook(t *testing.T) {
	root := t.TempDir()
	guard, err := fsutil.NewGuard(root)
	require.NoError(t, err)
	service := NewService(guard, "")
	path := filepath.Join(root, DefaultConfigFile)
	require.NoError(t, os.WriteFile(path, []byte("{}"), 0600))
	external := []byte("{\"external\":true}")
	out, err := service.Write(t.Context(), WriteInput{Config: inputForWrite(t, validWriteConfig()), Apply: true, BeforeWrite: func(context.Context, WriteOutput) error { return os.WriteFile(path, external, 0600) }})
	require.ErrorContains(t, err, "changed since preview")
	require.False(t, out.Applied)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, external, data)
}

package state

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
	"github.com/stretchr/testify/require"
)

func TestCanceledSaveDoesNotCreateState(t *testing.T) {
	root := t.TempDir()
	guard, err := fsutil.NewGuard(root)
	require.NoError(t, err)
	store := NewStore(guard)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	require.ErrorIs(t, store.Save(ctx, EmptyFile()), context.Canceled)
	_, err = os.Stat(filepath.Join(root, DefaultStatePath))
	require.ErrorIs(t, err, os.ErrNotExist)
	_, err = store.Load(ctx)
	require.ErrorIs(t, err, context.Canceled)
}

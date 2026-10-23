package locale

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Ret2Hell/i18n-mcp/internal/config"
	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
	"github.com/stretchr/testify/require"
)

func TestFilteredInventorySkipsUnrelatedFiles(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "locales"), 0700))
	for name, contents := range map[string]string{"en": `{"hello":"Hello"}`, "fr": `{"hello":"Bonjour"}`, "de": `invalid JSON`} {
		require.NoError(t, os.WriteFile(filepath.Join(root, "locales", name+".json"), []byte(contents), 0600))
	}
	guard, err := fsutil.NewGuard(root)
	require.NoError(t, err)
	service := NewService(guard, config.NewService(guard, ""))
	inv, err := service.InventoryFiltered(t.Context(), []string{"fr"}, nil)
	require.NoError(t, err)
	require.Equal(t, []string{"en", "fr"}, inv.Locales)
	_, err = service.Inventory(t.Context())
	require.Error(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = service.Inventory(ctx)
	require.ErrorIs(t, err, context.Canceled)
	_, err = FlattenContext(ctx, FileRef{}, map[string]any{"hello": "Hello"})
	require.ErrorIs(t, err, context.Canceled)
}

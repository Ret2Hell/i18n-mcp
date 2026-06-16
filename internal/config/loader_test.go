package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
	"github.com/stretchr/testify/require"
)

func TestResolveMissingConfigUsesDefaults(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	guard, err := fsutil.NewGuard(root)
	require.NoError(t, err)

	service := NewService(guard, "")
	got, err := service.Resolve(ctx)
	require.NoError(t, err)
	require.False(t, got.Exists)
	require.Equal(t, "en", got.SourceLocale)
	require.Equal(t, []string{"messages/{locale}.json", "locales/{locale}.json"}, got.LocaleFiles)
	require.Equal(t, 2, got.Format.Indent)
}

func TestResolveValidConfigMergesDefaults(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	data := []byte(`{
  "sourceLocale": "en-US",
  "targetLocales": ["fr", "de"],
  "localeFiles": ["messages/{locale}/{namespace}.json"],
  "format": {"sortKeys": true}
}`)
	require.NoError(t, os.WriteFile(filepath.Join(root, DefaultConfigFile), data, 0o600))

	guard, err := fsutil.NewGuard(root)
	require.NoError(t, err)
	service := NewService(guard, "")

	got, err := service.Resolve(ctx)
	require.NoError(t, err)
	require.True(t, got.Exists)
	require.Equal(t, "en-US", got.SourceLocale)
	require.Equal(t, []string{"fr", "de"}, got.TargetLocales)
	require.Equal(t, []string{"messages/{locale}/{namespace}.json"}, got.LocaleFiles)
	require.True(t, got.Format.SortKeys)
	require.Equal(t, 2, got.Format.Indent)
	require.True(t, got.Format.TrailingNewline)
}

func TestResolveRejectsInvalidJSON(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, DefaultConfigFile), []byte(`{"sourceLocale":`), 0o600))

	guard, err := fsutil.NewGuard(root)
	require.NoError(t, err)
	service := NewService(guard, "")

	_, err = service.Resolve(ctx)
	require.Error(t, err)
}

func TestResolveRejectsConfigPathOutsideRoot(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), ".i18n-mcp.json")

	guard, err := fsutil.NewGuard(root)
	require.NoError(t, err)
	service := NewService(guard, outside)

	_, err = service.Resolve(ctx)
	require.Error(t, err)
}

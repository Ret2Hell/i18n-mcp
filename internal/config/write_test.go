package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
	"github.com/stretchr/testify/require"
)

func TestWriteDryRunPreviewsWithoutWriting(t *testing.T) {
	ctx := t.Context()
	root := t.TempDir()
	guard, err := fsutil.NewGuard(root)
	require.NoError(t, err)
	service := NewService(guard, "")
	cfg := validWriteConfig()

	out, err := service.Write(ctx, WriteInput{Config: cfg})

	require.NoError(t, err)
	require.True(t, out.DryRun)
	require.False(t, out.Applied)
	require.True(t, out.Validation.Valid)
	require.Equal(t, DefaultConfigFile, out.ConfigPath)
	require.True(t, out.ChangedFile.Changed)
	require.False(t, out.ChangedFile.Written)
	require.Contains(t, out.ChangedFile.Diff, "+  \"sourceLocale\": \"en\"")
	_, err = os.Stat(filepath.Join(root, DefaultConfigFile))
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestWriteApplyWritesChangedConfig(t *testing.T) {
	ctx := t.Context()
	root := t.TempDir()
	guard, err := fsutil.NewGuard(root)
	require.NoError(t, err)
	service := NewService(guard, "")
	cfg := validWriteConfig()

	out, err := service.Write(ctx, WriteInput{Config: cfg, Apply: true})

	require.NoError(t, err)
	require.False(t, out.DryRun)
	require.True(t, out.Applied)
	require.True(t, out.ChangedFile.Changed)
	require.True(t, out.ChangedFile.Written)
	written, err := os.ReadFile(filepath.Join(root, DefaultConfigFile))
	require.NoError(t, err)
	require.Contains(t, string(written), "\"targetLocales\": [")
	require.Contains(t, string(written), "\"fr\"")
}

func TestWriteApplyUnchangedConfigDoesNotRewrite(t *testing.T) {
	ctx := t.Context()
	root := t.TempDir()
	guard, err := fsutil.NewGuard(root)
	require.NoError(t, err)
	service := NewService(guard, "")
	cfg := validWriteConfig()
	initial, err := renderConfig(cfg)
	require.NoError(t, err)
	configPath := filepath.Join(root, DefaultConfigFile)
	require.NoError(t, os.WriteFile(configPath, initial, 0o640))
	beforeInfo, err := os.Stat(configPath)
	require.NoError(t, err)

	out, err := service.Write(ctx, WriteInput{Config: cfg, Apply: true})

	require.NoError(t, err)
	require.False(t, out.DryRun)
	require.False(t, out.Applied)
	require.False(t, out.ChangedFile.Changed)
	require.False(t, out.ChangedFile.Written)
	afterInfo, err := os.Stat(configPath)
	require.NoError(t, err)
	require.Equal(t, beforeInfo.Mode().Perm(), afterInfo.Mode().Perm())
}

func TestWriteInvalidConfigReturnsValidationWithoutWriting(t *testing.T) {
	ctx := t.Context()
	root := t.TempDir()
	guard, err := fsutil.NewGuard(root)
	require.NoError(t, err)
	service := NewService(guard, "")
	cfg := validWriteConfig()
	cfg.SourceLocale = ""

	out, err := service.Write(ctx, WriteInput{Config: cfg, Apply: true})

	require.NoError(t, err)
	require.False(t, out.Validation.Valid)
	require.False(t, out.Applied)
	require.False(t, out.ChangedFile.Written)
	_, err = os.Stat(filepath.Join(root, DefaultConfigFile))
	require.ErrorIs(t, err, os.ErrNotExist)
}

func validWriteConfig() File {
	return File{
		SourceLocale:     "en",
		TargetLocales:    []string{"fr"},
		LocaleFiles:      []string{"messages/{locale}/{namespace}.json"},
		DefaultNamespace: "common",
		Format: FormatConfig{
			Indent:          2,
			TrailingNewline: true,
		},
		Translation: TranslationConfig{Mode: "agent"},
	}
}

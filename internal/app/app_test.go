package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewNormalizesProjectRoot(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	projectRoot := filepath.Join(root, ".")

	application, err := New(ctx, Options{ProjectRoot: projectRoot, ConfigPath: ".i18n-mcp.test.json", LogLevel: "error"})
	require.NoError(t, err)
	require.Equal(t, root, application.ProjectRoot)
	require.Equal(t, root, application.Guard.Root())
	require.Equal(t, root, application.Options.ProjectRoot)
	require.Equal(t, ".i18n-mcp.test.json", application.Options.ConfigPath)
	require.NotNil(t, application.Logger)
	require.NotNil(t, application.Config)
}

func TestNewRejectsMissingProjectRoot(t *testing.T) {
	ctx := context.Background()
	missingRoot := filepath.Join(t.TempDir(), "missing")

	_, err := New(ctx, Options{ProjectRoot: missingRoot})
	require.Error(t, err)
	require.Contains(t, err.Error(), "resolve project root")
}

func TestNewRejectsFileProjectRoot(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	file := filepath.Join(root, "file.txt")
	require.NoError(t, os.WriteFile(file, []byte{}, 0o600))

	_, err := New(ctx, Options{ProjectRoot: file})
	require.Error(t, err)
	require.Contains(t, err.Error(), "project root is not a directory")
}

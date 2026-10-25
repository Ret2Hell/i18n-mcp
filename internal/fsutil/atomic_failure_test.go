package fsutil

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAtomicWriteFileCleansTempOnRenameFailure(t *testing.T) {
	root := t.TempDir()
	guard, err := NewGuard(root)
	require.NoError(t, err)
	// Replacing a populated directory with a file must fail after temp creation.
	target := filepath.Join(root, "messages", "en.json")
	require.NoError(t, os.MkdirAll(target, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(target, "keep"), []byte("original"), 0600))
	err = AtomicWriteFile(guard, "messages/en.json", []byte("replacement"), 0600)
	require.ErrorContains(t, err, "rename temp file")
	data, err := os.ReadFile(filepath.Join(target, "keep"))
	require.NoError(t, err)
	require.Equal(t, "original", string(data))
	entries, err := os.ReadDir(filepath.Dir(target))
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, "en.json", entries[0].Name())
	require.True(t, entries[0].IsDir())
}

func TestAtomicWriteFileCannotCreateTempInReadOnlyDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix directory permissions")
	}
	root := t.TempDir()
	guard, err := NewGuard(root)
	require.NoError(t, err)
	dir := filepath.Join(root, "messages")
	require.NoError(t, os.Mkdir(dir, 0700))
	target := filepath.Join(dir, "en.json")
	require.NoError(t, os.WriteFile(target, []byte("original"), 0600))
	require.NoError(t, os.Chmod(dir, 0500))
	t.Cleanup(func() { require.NoError(t, os.Chmod(dir, 0700)) })
	// Root/capability-enabled environments can bypass mode bits.
	probe, err := os.CreateTemp(dir, "probe-*")
	if err == nil {
		require.NoError(t, probe.Close())
		require.NoError(t, os.Remove(probe.Name()))
		t.Skip("filesystem permissions do not restrict this user")
	}
	err = AtomicWriteFile(guard, "messages/en.json", []byte("replacement"), 0600)
	require.ErrorContains(t, err, "create temp file")
	require.ErrorIs(t, err, os.ErrPermission)
	data, err := os.ReadFile(target)
	require.NoError(t, err)
	require.Equal(t, "original", string(data))
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
}

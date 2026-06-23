package fsutil

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAtomicWriteFileCreatesParentAndCleansTempFile(t *testing.T) {
	root := t.TempDir()
	guard, err := NewGuard(root)
	require.NoError(t, err)

	require.NoError(t, AtomicWriteFile(guard, "messages/en.json", []byte(`{"hello":"world"}`), 0o600))

	path := filepath.Join(root, "messages", "en.json")
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, `{"hello":"world"}`, string(data))

	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())

	entries, err := os.ReadDir(filepath.Join(root, "messages"))
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, "en.json", entries[0].Name())
}

func TestAtomicWriteFileReplacesRegularFile(t *testing.T) {
	root := t.TempDir()
	guard, err := NewGuard(root)
	require.NoError(t, err)

	path := filepath.Join(root, "messages", "en.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte("old"), 0o600))

	require.NoError(t, AtomicWriteFile(guard, "messages/en.json", []byte("new"), 0o644))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "new", string(data))
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o644), info.Mode().Perm())
}

func TestAtomicWriteFileRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	guard, err := NewGuard(root)
	require.NoError(t, err)

	err = AtomicWriteFile(guard, "../outside.json", []byte("{}"), 0o600)
	require.Error(t, err)
}

func TestRejectSymlinkAncestorsRejectsRootParent(t *testing.T) {
	root := t.TempDir()

	err := rejectSymlinkAncestors(root, filepath.Dir(root))
	require.Error(t, err)
	require.Contains(t, err.Error(), "path escapes project root")
}

func TestAtomicWriteFileRejectsTargetSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires privileges on some Windows systems")
	}
	root := t.TempDir()
	realPath := filepath.Join(root, "real.json")
	linkPath := filepath.Join(root, "messages", "en.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(linkPath), 0o700))
	require.NoError(t, os.WriteFile(realPath, []byte("old"), 0o600))
	require.NoError(t, os.Symlink(realPath, linkPath))

	guard, err := NewGuard(root)
	require.NoError(t, err)

	err = AtomicWriteFile(guard, "messages/en.json", []byte("new"), 0o600)
	require.Error(t, err)
	require.Contains(t, err.Error(), "refuse to write through symlink")

	data, err := os.ReadFile(realPath)
	require.NoError(t, err)
	require.Equal(t, "old", string(data))
}

func TestAtomicWriteFileRejectsSymlinkAncestor(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires privileges on some Windows systems")
	}
	root := t.TempDir()
	outside := t.TempDir()
	require.NoError(t, os.Symlink(outside, filepath.Join(root, "messages")))

	guard, err := NewGuard(root)
	require.NoError(t, err)

	err = AtomicWriteFile(guard, "messages/en.json", []byte("new"), 0o600)
	require.Error(t, err)
	require.Contains(t, err.Error(), "refuse to write through symlink path component")
}

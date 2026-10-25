package fsutil

import (
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

func TestVerifyFileVersionDetectsContentAndExistenceChanges(t *testing.T) {
	root := t.TempDir()
	guard, err := NewGuard(root)
	require.NoError(t, err)
	require.NoError(t, VerifyFileVersion(guard, "file.json", nil, true))
	require.NoError(t, os.WriteFile(filepath.Join(root, "file.json"), []byte("before"), 0600))
	require.Error(t, VerifyFileVersion(guard, "file.json", nil, true))
	require.NoError(t, VerifyFileVersion(guard, "file.json", []byte("before"), false))
	require.Error(t, VerifyFileVersion(guard, "file.json", []byte("different"), false))
	require.NoError(t, os.Remove(filepath.Join(root, "file.json")))
	require.Error(t, VerifyFileVersion(guard, "file.json", []byte("before"), false))
}

func TestDiffExposesMissingFinalNewline(t *testing.T) {
	patch := UnifiedDiff("file.json", []byte("before"), []byte("after\n"))
	require.Contains(t, patch, "-before\n\\ No newline at end of file\n")
}

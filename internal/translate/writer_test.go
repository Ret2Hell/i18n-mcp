package translate

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
	"github.com/stretchr/testify/require"
)

func TestWriteEditsWritesChangedFilesAtomically(t *testing.T) {
	root := t.TempDir()
	guard, err := fsutil.NewGuard(root)
	require.NoError(t, err)
	svc := &Service{guard: guard}

	report, err := svc.WriteEdits(t.Context(), []FileEdit{
		{Path: "locales/en.json", Before: []byte(`{"hello":"old"}`), After: []byte(`{"hello":"world"}`)},
		{Path: "locales/fr.json", Before: []byte(`{"hello":"old"}`), After: []byte(`{"hello":"bonjour"}`)},
	})
	require.NoError(t, err)
	require.Equal(t, WriteReport{Written: []string{"locales/en.json", "locales/fr.json"}}, report)

	assertFileContent(t, filepath.Join(root, "locales", "en.json"), `{"hello":"world"}`)
	assertFileContent(t, filepath.Join(root, "locales", "fr.json"), `{"hello":"bonjour"}`)
	assertNoTempFiles(t, filepath.Join(root, "locales"))
}

func TestWriteEditsPreservesExistingPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits are not portable on Windows")
	}
	root := t.TempDir()
	path := filepath.Join(root, "locales", "en.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte("old"), 0o640))

	guard, err := fsutil.NewGuard(root)
	require.NoError(t, err)
	svc := &Service{guard: guard}

	report, err := svc.WriteEdits(t.Context(), []FileEdit{
		{Path: "locales/en.json", Before: []byte("old"), After: []byte("new")},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"locales/en.json"}, report.Written)

	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o640), info.Mode().Perm())
}

func TestWriteEditsRejectsSymlinkTarget(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires privileges on some Windows systems")
	}
	root := t.TempDir()
	realPath := filepath.Join(root, "real.json")
	linkPath := filepath.Join(root, "locales", "en.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(linkPath), 0o700))
	require.NoError(t, os.WriteFile(realPath, []byte("old"), 0o600))
	require.NoError(t, os.Symlink(realPath, linkPath))

	guard, err := fsutil.NewGuard(root)
	require.NoError(t, err)
	svc := &Service{guard: guard}

	report, err := svc.WriteEdits(t.Context(), []FileEdit{
		{Path: "locales/en.json", Before: []byte("old"), After: []byte("new")},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "refuse to write through symlink")
	require.Equal(t, WriteReport{Skipped: []string{"locales/en.json"}}, report)
	assertFileContent(t, realPath, "old")
}

func TestWriteEditsReportsSkippedFilesAfterFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires privileges on some Windows systems")
	}
	root := t.TempDir()
	realPath := filepath.Join(root, "real.json")
	linkPath := filepath.Join(root, "locales", "fail.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(linkPath), 0o700))
	require.NoError(t, os.WriteFile(realPath, []byte("old"), 0o600))
	require.NoError(t, os.Symlink(realPath, linkPath))

	guard, err := fsutil.NewGuard(root)
	require.NoError(t, err)
	svc := &Service{guard: guard}

	report, err := svc.WriteEdits(t.Context(), []FileEdit{
		{Path: "locales/written.json", Before: []byte("old"), After: []byte("new")},
		{Path: "locales/unchanged.json", Before: []byte("same"), After: []byte("same")},
		{Path: "locales/fail.json", Before: []byte("old"), After: []byte("new")},
		{Path: "locales/not-attempted.json", Before: []byte("old"), After: []byte("new")},
	})
	require.Error(t, err)
	require.Equal(t, WriteReport{
		Written:   []string{"locales/written.json"},
		Unchanged: []string{"locales/unchanged.json"},
		Skipped:   []string{"locales/fail.json", "locales/not-attempted.json"},
	}, report)
	assertFileContent(t, filepath.Join(root, "locales", "written.json"), "new")
	require.NoFileExists(t, filepath.Join(root, "locales", "not-attempted.json"))
}

func TestAppendRemainingPreservesOrderAndFirstMatch(t *testing.T) {
	edits := []FileEdit{{Path: "first"}, {Path: "failed"}, {Path: "next"}, {Path: "failed"}}
	for _, tc := range []struct {
		name, failed string
		edits        []FileEdit
		want         []string
	}{
		{name: "first match", failed: "failed", edits: edits, want: []string{"existing", "failed", "next", "failed"}},
		{name: "first edit", failed: "first", edits: edits, want: []string{"existing", "first", "failed", "next", "failed"}},
		{name: "missing", failed: "missing", edits: edits, want: []string{"existing"}},
		{name: "empty", failed: "failed", want: []string{"existing"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, appendRemaining([]string{"existing"}, tc.edits, tc.failed))
		})
	}
	require.Nil(t, appendRemaining(nil, edits, "missing"))
}

func TestWriteEditsTreatsNilAndEmptyBytesAsUnchanged(t *testing.T) {
	guard, err := fsutil.NewGuard(t.TempDir())
	require.NoError(t, err)
	svc := &Service{guard: guard}

	report, err := svc.WriteEdits(t.Context(), []FileEdit{
		{Path: "nil-before.json", After: []byte{}},
		{Path: "nil-after.json", Before: []byte{}},
	})
	require.NoError(t, err)
	require.Equal(t, WriteReport{Unchanged: []string{"nil-before.json", "nil-after.json"}}, report)
	require.NoFileExists(t, filepath.Join(guard.Root(), "nil-before.json"))
	require.NoFileExists(t, filepath.Join(guard.Root(), "nil-after.json"))
}

func assertFileContent(t *testing.T, path string, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, want, string(data))
}

func assertNoTempFiles(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	for _, entry := range entries {
		require.NotContains(t, entry.Name(), ".tmp-")
	}
}

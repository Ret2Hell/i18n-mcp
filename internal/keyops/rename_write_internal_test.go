package keyops

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
	"github.com/stretchr/testify/require"
)

func TestAppendRemainingIncludesFailedAndFollowingEdits(t *testing.T) {
	edits := []fileEdit{{Path: "first.json"}, {Path: "failed.json"}, {Path: "last.json"}}

	got := appendRemaining([]string{"already.json"}, edits, "failed.json")

	require.Equal(t, []string{"already.json", "failed.json", "last.json"}, got)
}

func TestAppendRemainingLeavesOutputWhenFailedPathMissing(t *testing.T) {
	edits := []fileEdit{{Path: "first.json"}, {Path: "last.json"}}

	got := appendRemaining([]string{"already.json"}, edits, "missing.json")

	require.Equal(t, []string{"already.json"}, got)
}

func TestMarkWrittenMarksOnlyWrittenFiles(t *testing.T) {
	files := []ChangedFile{{Path: "a.json"}, {Path: "b.json"}}

	got := markWritten(files, []string{"b.json"})

	require.False(t, got[0].Written)
	require.True(t, got[1].Written)
}

func TestWriteEditsReportsResolveFailureAsSkipped(t *testing.T) {
	root := t.TempDir()
	guard, err := fsutil.NewGuard(root)
	require.NoError(t, err)
	svc := &Service{guard: guard}
	edits := []fileEdit{
		{Path: "ok.json", Before: []byte("old"), After: []byte("new")},
		{Path: "../escape.json", Before: []byte("old"), After: []byte("new")},
		{Path: "after.json", Before: []byte("old"), After: []byte("new")},
	}

	report, err := svc.writeEdits(t.Context(), edits)

	require.Error(t, err)
	require.Equal(t, []string{"ok.json"}, report.Written)
	require.Equal(t, []string{"../escape.json", "after.json"}, report.Skipped)
	data, readErr := os.ReadFile(filepath.Join(root, "ok.json"))
	require.NoError(t, readErr)
	require.Equal(t, "new", string(data))
}

func TestWriteEditsReportsUnchangedBeforeWrites(t *testing.T) {
	root := t.TempDir()
	guard, err := fsutil.NewGuard(root)
	require.NoError(t, err)
	svc := &Service{guard: guard}
	edits := []fileEdit{
		{Path: "same.json", Before: []byte("same"), After: []byte("same")},
		{Path: "changed.json", Before: []byte("old"), After: []byte("new")},
	}

	report, err := svc.writeEdits(t.Context(), edits)

	require.NoError(t, err)
	require.Equal(t, []string{"same.json"}, report.Unchanged)
	require.Equal(t, []string{"changed.json"}, report.Written)
	require.Empty(t, report.Skipped)
}

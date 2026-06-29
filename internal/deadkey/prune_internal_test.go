package deadkey

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
	"github.com/stretchr/testify/require"
)

func TestAppendRemainingPruneEditsIncludesFailedAndFollowingEdits(t *testing.T) {
	edits := []pruneEdit{{Path: "first.json"}, {Path: "failed.json"}, {Path: "last.json"}}

	got := appendRemainingPruneEdits([]string{"already.json"}, edits, "failed.json")

	require.Equal(t, []string{"already.json", "failed.json", "last.json"}, got)
}

func TestAppendRemainingPruneEditsLeavesOutputWhenFailedPathMissing(t *testing.T) {
	edits := []pruneEdit{{Path: "first.json"}, {Path: "last.json"}}

	got := appendRemainingPruneEdits([]string{"already.json"}, edits, "missing.json")

	require.Equal(t, []string{"already.json"}, got)
}

func TestMarkPruneWrittenMarksOnlyWrittenFiles(t *testing.T) {
	files := []ChangedFile{{Path: "a.json"}, {Path: "b.json"}}

	got := markPruneWritten(files, []string{"b.json"})

	require.False(t, got[0].Written)
	require.True(t, got[1].Written)
}

func TestPreviewPruneEditsReportsChangedAndUnchangedFiles(t *testing.T) {
	edits := []pruneEdit{
		{Path: "changed.json", Before: []byte("old\n"), After: []byte("new\n")},
		{Path: "same.json", Before: []byte("same\n"), After: []byte("same\n")},
	}

	got := previewPruneEdits(edits)

	require.Len(t, got, 2)
	require.Equal(t, "changed.json", got[0].Path)
	require.True(t, got[0].Changed)
	require.NotEmpty(t, got[0].Diff)
	require.Equal(t, "same.json", got[1].Path)
	require.False(t, got[1].Changed)
	require.Empty(t, got[1].Diff)
}

func TestWritePruneEditsReportsResolveFailureAsSkipped(t *testing.T) {
	root := t.TempDir()
	guard, err := fsutil.NewGuard(root)
	require.NoError(t, err)
	svc := &Service{guard: guard}
	edits := []pruneEdit{
		{Path: "ok.json", Before: []byte("old"), After: []byte("new")},
		{Path: "../escape.json", Before: []byte("old"), After: []byte("new")},
		{Path: "after.json", Before: []byte("old"), After: []byte("new")},
	}

	report, err := svc.writePruneEdits(t.Context(), edits)

	require.Error(t, err)
	require.Equal(t, []string{"ok.json"}, report.Written)
	require.Equal(t, []string{"../escape.json", "after.json"}, report.Skipped)
	data, readErr := os.ReadFile(filepath.Join(root, "ok.json"))
	require.NoError(t, readErr)
	require.Equal(t, "new", string(data))
}

func TestWritePruneEditsReportsUnchangedBeforeWrites(t *testing.T) {
	root := t.TempDir()
	guard, err := fsutil.NewGuard(root)
	require.NoError(t, err)
	svc := &Service{guard: guard}
	edits := []pruneEdit{
		{Path: "same.json", Before: []byte("same"), After: []byte("same")},
		{Path: "changed.json", Before: []byte("old"), After: []byte("new")},
	}

	report, err := svc.writePruneEdits(t.Context(), edits)

	require.NoError(t, err)
	require.Equal(t, []string{"same.json"}, report.Unchanged)
	require.Equal(t, []string{"changed.json"}, report.Written)
	require.Empty(t, report.Skipped)
}

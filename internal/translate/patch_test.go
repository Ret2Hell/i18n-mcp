package translate

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPreviewEditsMarksChangedFiles(t *testing.T) {
	edits := []FileEdit{
		{Path: "locales/en.json", Before: []byte("same\n"), After: []byte("same\n")},
		{Path: "locales/fr.json", Before: []byte("hello\n"), After: []byte("bonjour\n")},
	}

	changedFiles := PreviewEdits(edits)

	require.Len(t, changedFiles, 2)
	require.Equal(t, "locales/en.json", changedFiles[0].Path)
	require.False(t, changedFiles[0].Changed)
	require.NotEmpty(t, changedFiles[0].BeforeSHA256)
	require.Equal(t, changedFiles[0].BeforeSHA256, changedFiles[0].AfterSHA256)
	require.Equal(t, "locales/fr.json", changedFiles[1].Path)
	require.True(t, changedFiles[1].Changed)
	require.Contains(t, changedFiles[1].Diff, "--- a/locales/fr.json\n")
	require.Contains(t, changedFiles[1].Diff, "-hello\n")
	require.Contains(t, changedFiles[1].Diff, "+bonjour\n")
}

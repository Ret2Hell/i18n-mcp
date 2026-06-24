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
	require.Equal(t, ChangedFile{Path: "locales/en.json", Changed: false}, changedFiles[0])
	require.Equal(t, "locales/fr.json", changedFiles[1].Path)
	require.True(t, changedFiles[1].Changed)
	require.Contains(t, changedFiles[1].Diff, "--- a/locales/fr.json\n")
	require.Contains(t, changedFiles[1].Diff, "-hello\n")
	require.Contains(t, changedFiles[1].Diff, "+bonjour\n")
}

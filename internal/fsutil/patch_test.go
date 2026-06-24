package fsutil

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnifiedDiffEmptyForIdenticalContent(t *testing.T) {
	diff := UnifiedDiff("locales/en.json", []byte("hello\n"), []byte("hello\n"))
	require.Empty(t, diff)
}

func TestUnifiedDiffContainsHeadersAndChangedLines(t *testing.T) {
	diff := UnifiedDiff("locales/en.json", []byte("hello\nold"), []byte("hello\nnew\n"))

	require.True(t, strings.HasPrefix(diff, "--- a/locales/en.json\n+++ b/locales/en.json\n@@ -1,2 +1,2 @@\n"))
	require.Contains(t, diff, "-hello\n")
	require.Contains(t, diff, "-old\n")
	require.Contains(t, diff, "+hello\n")
	require.Contains(t, diff, "+new\n")
}

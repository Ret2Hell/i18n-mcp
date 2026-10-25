package fsutil

import (
	"bytes"
	"fmt"
	"strings"
)

// UnifiedDiff returns a simple unified diff for before and after content.
func UnifiedDiff(path string, before []byte, after []byte) string {
	if bytes.Equal(before, after) {
		return ""
	}
	beforeLines := splitPatchLines(string(before))
	afterLines := splitPatchLines(string(after))

	var b strings.Builder
	fmt.Fprintf(&b, "--- a/%s\n", path)
	fmt.Fprintf(&b, "+++ b/%s\n", path)
	fmt.Fprintf(&b, "@@ -1,%d +1,%d @@\n", len(beforeLines), len(afterLines))
	for _, line := range beforeLines {
		writePatchLine(&b, '-', line)
	}
	for _, line := range afterLines {
		writePatchLine(&b, '+', line)
	}
	return b.String()
}

func writePatchLine(b *strings.Builder, prefix byte, line string) {
	b.WriteByte(prefix)
	b.WriteString(line)
	if !strings.HasSuffix(line, "\n") {
		b.WriteString("\n\\ No newline at end of file\n")
	}
}

func splitPatchLines(s string) []string {
	if s == "" {
		return nil
	}
	lines := strings.SplitAfter(s, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

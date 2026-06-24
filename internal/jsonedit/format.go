package jsonedit

import (
	"bytes"
	"strings"
)

// DetectFormat infers indentation and trailing newline style from JSON bytes.
func DetectFormat(data []byte, fallbackIndent int, sortKeys bool) Format {
	indent := fallbackIndent
	if indent <= 0 {
		indent = 2
	}
	format := Format{
		Indent:          strings.Repeat(" ", indent),
		TrailingNewline: bytes.HasSuffix(data, []byte("\n")),
		SortKeys:        sortKeys,
	}
	for _, line := range bytes.Split(data, []byte("\n")) {
		trimmed := bytes.TrimLeft(line, " \t")
		if len(trimmed) == len(line) || len(trimmed) == 0 {
			continue
		}
		format.Indent = string(line[:len(line)-len(trimmed)])
		break
	}
	return format
}

package validate

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestICUArgumentSyntaxErrors(t *testing.T) {
	for _, tc := range []struct{ input, message string }{
		{`{}`, "expected argument name"},
		{`{name text}`, "expected comma or closing brace"},
		{`{n, number short}`, "expected format style"},
		{`{n, number, }`, "empty format style"},
		{`{n, number, {bad}}`, "braces are not supported inside format styles"},
		{`{n, number, short`, "unclosed formatted argument"},
		{`{n, plural}`, "expected selector list"},
		{`{n, plural, offset:}`, "invalid plural offset"},
		{`{n, plural, offset:18446744073709551616 other {x}}`, "invalid plural offset"},
		{`{n, plural, offset:1other {x}}`, "expected whitespace after offset"},
		{`{n, plural,`, "expected selector"},
		{`{n, plural, =x {x} other {y}}`, "invalid exact numeric selector"},
		{`{n, plural, bogus {x} other {y}}`, "invalid plural category"},
		{`{n, plural, =01 {x} =1 {y} other {z}}`, "duplicate selector =1"},
		{`{n, select, other {x} other {y}}`, "duplicate selector other"},
		{`{n, select, choice x}`, "expected selector body"},
		{`{n, select, choice {x}}`, "missing required other selector"},
		{`{n, select, other {x`, "unclosed branch body"},
	} {
		t.Run(tc.input, func(t *testing.T) {
			_, err := parseICU(t.Context(), tc.input)
			require.ErrorContains(t, err, tc.message)
			require.ErrorContains(t, err, "at byte")
		})
	}
}

func TestICUArgumentShapes(t *testing.T) {
	for _, kind := range []string{"number", "date", "time"} {
		t.Run(kind, func(t *testing.T) {
			nodes, err := parseICU(t.Context(), "{value, "+kind+", short}")
			require.NoError(t, err)
			require.Equal(t, []icuNode{{name: "value", kind: kind, style: "short"}}, nodes)
			nodes, err = parseICU(t.Context(), "{value, "+kind+"}")
			require.NoError(t, err)
			require.Equal(t, "", nodes[0].style)
		})
	}
	nodes, err := parseICU(t.Context(), "{n, plural, offset:2\n =01 {{name}} other {#}}")
	require.NoError(t, err)
	require.Equal(t, uint64(2), nodes[0].offset)
	require.Equal(t, []icuNode{{name: "name", kind: "argument"}}, nodes[0].branches["=1"])
	require.Equal(t, []icuNode{{text: "#"}}, nodes[0].branches["other"])
	nodes, err = parseICU(t.Context(), `{n, select, arbitrary-label {yes} other {no}}`)
	require.NoError(t, err)
	require.Contains(t, nodes[0].branches, "arbitrary-label")
}

func TestICUArgumentNodeLimits(t *testing.T) {
	_, err := parseICU(t.Context(), strings.Repeat("{n}", maxICUNodes+1))
	require.ErrorContains(t, err, "node limit")
	// A select branch contributes a node independently of nested arguments.
	_, err = parseICU(t.Context(), "{n, select, other {"+strings.Repeat("{n}", maxICUNodes-1)+"}}")
	require.ErrorContains(t, err, "node limit")
}

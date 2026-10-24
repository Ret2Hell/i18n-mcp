package validate

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestICUBranchTextIsNotAnArgument(t *testing.T) {
	cases := []struct{ source, target string }{
		{`{count, plural, =0 {Notes} one {# note} other {# notes}}`, `{count, plural, =0 {Бележки} one {# бележка} other {# бележки}}`},
		{`You've earned {count} achievement{count, plural, one {} other {s}}!`, `Спечелихте {count} постижени{count, plural, one {е} other {я}}!`},
		{`{n, plural, one {{name}: # item} other {{name}: # items}}`, `{n, plural, one {{name}: # rzecz} few {{name}: # rzeczy} many {{name}: # rzeczy} other {{name}: # rzeczy}}`},
		{`{gender, select, female {{count, plural, one {{name}: #} other {{name}: #}}} other {{name}}}`, `{gender, select, female {{count, plural, one {{name}: #} few {{name}: #} other {{name}: #}}} other {{name}}}`},
		{`{n, plural, other {'{literal}' and ''quote'' #}}`, `{n, plural, other {'{littéral}' et ''citation'' #}}`},
		{`{n, number, ::currency/USD} on {date, date, short}`, `{n, number, ::currency/USD} le {date, date, short}`},
		{`{rank, selectordinal, one {#st} two {#nd} few {#rd} other {#th}}`, `{rank, selectordinal, one {#er} other {#e}}`},
		{`{count, plural, one {# question} other {# questions}}`, `Ek olarak {count} soru eklenir.`},
		{`{count}: {count, plural, one {# item} other {# items}}`, `{count}: toplam {count} öğe`},
	}
	for _, c := range cases {
		t.Run(c.source, func(t *testing.T) {
			r := NewService().ValidateStrings(c.source, c.target)
			require.True(t, r.OK, "%+v", r.Issues)
		})
	}
	require.Equal(t, []string{"{count}"}, ExtractPlaceholders(cases[0].source))
	require.Equal(t, []string{"n", "name"}, ExtractICUArguments(cases[2].source))
}

func TestICURejectsInvalidStructureAndChangedContracts(t *testing.T) {
	source := `{n, plural, offset:1 =0 {none} one {one} other {{name}: #}}`
	for _, target := range []string{
		`{n, plural, offset:1 =0 {none} one {one}}`,
		`{n, plural, offset:1 =0 {none} other {one} other {two}}`,
		`{n, plural, offset:-1 =0 {none} other {{name}: #}}`,
		`{n, plural, offset:1 =0 none other {{name}: #}}`,
		`{n, plural, =0 {none} other {{name}: #}}`,
		`{n, plural, offset:1 other {{name}: #}}`,
		`{n, number} {name}`,
		`{n, plural, offset:1 =0 {none} other {#}}`,
		`{n, plural, offset:1 =0 {none} other {{extra}: #}}`,
	} {
		t.Run(target, func(t *testing.T) { r := NewService().ValidateStrings(source, target); require.False(t, r.OK) })
	}
	r := NewService().ValidateStrings(`{n, select, yes {yes} other {no}}`, `{n, select, other {non}}`)
	require.False(t, r.OK)
	requireIssueCode(t, r.Issues, "icu_selector_changed")
	r = NewService().ValidateStrings(`{n, plural, other {broken}`, `{n, plural, other {valid}}`)
	requireIssueCode(t, r.Issues, "icu_source_invalid")
}

func TestICUParserBoundsAndQuotes(t *testing.T) {
	_, err := parseICU(t.Context(), strings.Repeat("x", maxICUBytes+1))
	require.Error(t, err)
	deep := strings.Repeat("{n, plural, other {", maxICUDepth+1) + strings.Repeat("}}", maxICUDepth+1)
	_, err = parseICU(t.Context(), deep)
	require.ErrorContains(t, err, "nesting limit")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = parseICU(ctx, `{n, plural, other {#}}`)
	require.ErrorIs(t, err, context.Canceled)
	nodes, err := parseICU(t.Context(), `L'heure: '{' {name} ''`)
	require.NoError(t, err)
	require.Len(t, icuArguments(nodes), 1)
	_, err = parseICU(t.Context(), `{n, unknown}`)
	require.ErrorContains(t, err, "unsupported")
}

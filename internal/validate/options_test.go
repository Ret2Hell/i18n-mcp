package validate

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPrintfIsExplicitlyOptIn(t *testing.T) {
	for _, pair := range []Pair{
		{Source: `{score}% score`, Target: `Skóre: {score} %`},
		{Source: `{percent}% of users`, Target: `%{percent} des utilisateurs`, Options: Options{MessageFormat: "icu"}},
		{Source: `50% done`, Target: `50 % terminé`},
	} {
		r := NewService().ValidatePair(pair)
		require.True(t, r.OK, "%+v", r.Issues)
	}
	tokens, err := ExtractPlaceholdersFor(t.Context(), `{score}% score`, Options{MessageFormat: "icu"})
	require.NoError(t, err)
	require.Equal(t, []string{"{score}"}, tokens)
}

func TestPrintfPositionsWidthPrecisionAndEscapes(t *testing.T) {
	for _, c := range []struct {
		source, target string
		valid          bool
	}{
		{`%s: %d`, `%1$s : %2$d`, true},
		{`%1$s: %2$d`, `%2$d : %1$s`, true},
		{`%s: %d`, `%d : %s`, false},
		{`%s: %d`, `%s : %f`, false},
		{`%*.*f`, `%3$*1$.*2$f`, true},
		{`%*s`, `%s`, false},
		{`%%s %s`, `%%s %s`, true},
		{`%%%s`, `%%%s`, true},
		{`%02d`, `%d`, true},
		{`%0$s`, `%s`, false},
		{`%s`, `%q`, false},
	} {
		r := NewService().ValidatePair(Pair{Source: c.source, Target: c.target, Options: Options{Printf: true}})
		require.Equal(t, c.valid, r.OK, "%s -> %s: %+v", c.source, c.target, r.Issues)
	}
	tokens, err := ExtractPlaceholdersFor(t.Context(), `%%s %02d {name}`, Options{Printf: true})
	require.NoError(t, err)
	require.Equal(t, []string{"%02d", "{name}"}, tokens)
}

func TestExplicitMessageSyntax(t *testing.T) {
	r := NewService().ValidatePair(Pair{Source: `{{name}} has {literal}`, Target: `{{name}} a {texte}`, Options: Options{MessageFormat: "i18next"}})
	require.True(t, r.OK)
	r = NewService().ValidatePair(Pair{Source: `Hello {name}`, Target: `Bonjour {user}`, Options: Options{MessageFormat: "icu"}})
	require.False(t, r.OK)
	requireIssueCode(t, r.Issues, "icu_argument_missing")
	r = NewService().ValidatePair(Pair{Source: `Hello {name}`, Target: `Bonjour {name`, Options: Options{MessageFormat: "icu"}})
	require.False(t, r.OK)
	requireIssueCode(t, r.Issues, "icu_target_invalid")
}

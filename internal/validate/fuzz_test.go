package validate

import "testing"

func FuzzExtractPlaceholders(f *testing.F) {
	f.Add("Hello {name}")
	f.Add("{{ user }} %{count} %02d")
	f.Add("%%% {{{")
	f.Fuzz(func(t *testing.T, s string) {
		_ = ExtractPlaceholders(s)
	})
}

func FuzzExtractTagTokens(f *testing.F) {
	f.Add("<strong>Hello</strong>")
	f.Add("<link href=\"/docs\">docs</link>")
	f.Add("<<<< >>>>")
	f.Fuzz(func(t *testing.T, s string) {
		_ = ExtractTagTokens(s)
		_ = ExtractTags(s)
	})
}

func FuzzICUShape(f *testing.F) {
	f.Add("{count, plural, one {# item} other {# items}}")
	f.Add("{count, plural,")
	f.Add("plain text")
	f.Fuzz(func(t *testing.T, s string) {
		_ = LooksLikeICU(s)
		_ = HasBalancedBraces(s)
		_ = ExtractICUArguments(s)
	})
}

func FuzzValidateStrings(f *testing.F) {
	f.Add("Hello {name}", "Bonjour {name}")
	f.Add("<strong>Hello</strong>", "<strong>Bonjour</strong>")
	f.Add("{count, plural, one {# item} other {# items}}", "{count, plural, one {# element} other {# elements}}")
	f.Fuzz(func(t *testing.T, source string, target string) {
		_ = NewService().ValidateStrings(source, target)
	})
}

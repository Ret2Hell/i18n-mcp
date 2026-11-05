package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigJSONUsesStrictV2Decoding(t *testing.T) {
	for name, input := range map[string]string{
		"duplicate":        `{"sourceLocale":"en","sourceLocale":"fr"}`,
		"nested_duplicate": `{"format":{"indent":2,"indent":4}}`,
		"wrong_case":       `{"SourceLocale":"en"}`,
		"nested_case":      `{"format":{"Indent":2}}`,
		"unknown":          `{"typo":true}`,
		"invalid_utf8":     "{\"sourceLocale\":\"\xff\"}",
		"trailing_value":   `{} {}`,
		"trailing_junk":    `{} junk`,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := decodeInput([]byte(input))
			require.Error(t, err)
		})
	}
}

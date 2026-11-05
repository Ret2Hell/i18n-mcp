package locale

import (
	"encoding/json/jsontext"
	"os"
	"path/filepath"
	"testing"

	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
	"github.com/stretchr/testify/require"
)

func TestParseJSONRejectsAmbiguousOrInvalidInput(t *testing.T) {
	root := t.TempDir()
	guard, err := fsutil.NewGuard(root)
	require.NoError(t, err)
	for name, input := range map[string]string{
		"duplicate":         `{"hello":"first","hello":"second"}`,
		"escaped_duplicate": `{"hello":"first","\u0068ello":"second"}`,
		"nested_duplicate":  `{"nested":{"hello":"first","hello":"second"}}`,
		"invalid_utf8":      "{\"hello\":\"\xff\"}",
		"trailing_value":    `{} {}`,
		"trailing_junk":     `{} junk`,
		"empty":             ``,
	} {
		t.Run(name, func(t *testing.T) {
			require.NoError(t, os.WriteFile(filepath.Join(root, "en.json"), []byte(input), 0o600))
			_, err := ParseJSONFile(t.Context(), guard, "en.json")
			var parseError *ParseError
			require.ErrorAs(t, err, &parseError)
			require.Equal(t, "en.json", parseError.Path)
		})
	}
}

func TestParseJSONPreservesNumericLiteralsAtEveryDepth(t *testing.T) {
	root := t.TempDir()
	guard, err := fsutil.NewGuard(root)
	require.NoError(t, err)
	input := `{"large":9007199254740993,"nested":[1.2300,-2e100,{"max":18446744073709551615}],"message":"Hello \/ \ud83d\ude00"}`
	require.NoError(t, os.WriteFile(filepath.Join(root, "en.json"), []byte(input), 0o600))
	doc, err := ParseJSONFile(t.Context(), guard, "en.json")
	require.NoError(t, err)
	value := doc.Value.(map[string]any)
	require.Equal(t, jsontext.Value("9007199254740993"), value["large"])
	nested := value["nested"].([]any)
	require.Equal(t, jsontext.Value("1.2300"), nested[0])
	require.Equal(t, jsontext.Value("-2e100"), nested[1])
	require.Equal(t, jsontext.Value("18446744073709551615"), nested[2].(map[string]any)["max"])
	require.Equal(t, "Hello / 😀", value["message"])
	require.Equal(t, input, string(doc.Raw))
}

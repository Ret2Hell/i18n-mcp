package jsonedit

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOrderedJSONUsesStrictV2Syntax(t *testing.T) {
	for name, input := range map[string]string{
		"duplicate":        `{"a":"first","a":"second"}`,
		"escaped_name":     `{"a":"first","\u0061":"second"}`,
		"nested_duplicate": `{"nested":{"a":1,"a":2}}`,
		"invalid_utf8":     "{\"a\":\"\xff\"}",
		"go_only_escape":   `{"a":"\x41"}`,
		"trailing_junk":    `{} junk`,
		"trailing_value":   `{} {}`,
		"invalid_space":    "{}\u00a0",
	} {
		t.Run(name, func(t *testing.T) {
			_, err := Parse([]byte(input), 2, false)
			require.Error(t, err)
		})
	}
	doc, err := Parse([]byte(`{"message":"Hello \/ \ud83d\ude00","number":1.2300}`), 2, false)
	require.NoError(t, err)
	message, exists, err := doc.String([]string{"message"})
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, "Hello / 😀", message)
	output, err := doc.Render()
	require.NoError(t, err)
	require.Contains(t, string(output), "1.2300")
}

func TestOrderedJSONRenderDoesNotSilentlyReplaceInvalidData(t *testing.T) {
	for name, root := range map[string]*Value{
		"string": {Kind: KindString, String: string([]byte{0xff})},
		"key": {Kind: KindObject, Object: []Member{
			{Key: string([]byte{0xff}), Value: &Value{Kind: KindString, String: "value"}},
		}},
		"duplicate": {Kind: KindObject, Object: []Member{
			{Key: "a", Value: &Value{Kind: KindString, String: "first"}},
			{Key: "a", Value: &Value{Kind: KindString, String: "second"}},
		}},
		"invalid_raw": {Kind: KindRaw, RawJSON: "not JSON"},
	} {
		t.Run(name, func(t *testing.T) {
			data, err := (&Document{Root: root}).Render()
			require.Error(t, err)
			require.Nil(t, data)
		})
	}
}

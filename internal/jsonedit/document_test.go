package jsonedit

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJSONEditPreservesExistingKeyOrder(t *testing.T) {
	doc, err := Parse([]byte("{\n  \"z\": \"old\",\n  \"a\": \"first\"\n}\n"), 2, false)
	require.NoError(t, err)

	require.NoError(t, doc.SetString([]string{"z"}, "new"))
	after, err := doc.Render()
	require.NoError(t, err)

	require.Equal(t, "{\n  \"z\": \"new\",\n  \"a\": \"first\"\n}\n", string(after))
}

func TestJSONEditAppendsNewKeyWhenSortDisabled(t *testing.T) {
	doc, err := Parse([]byte("{\n  \"b\": \"bee\",\n  \"a\": \"aye\"\n}\n"), 2, false)
	require.NoError(t, err)

	require.NoError(t, doc.SetString([]string{"c"}, "see"))
	after, err := doc.Render()
	require.NoError(t, err)

	require.Equal(t, "{\n  \"b\": \"bee\",\n  \"a\": \"aye\",\n  \"c\": \"see\"\n}\n", string(after))
}

func TestJSONEditSortsKeysWhenConfigured(t *testing.T) {
	doc, err := Parse([]byte("{\n  \"b\": {\n    \"d\": \"dee\",\n    \"c\": \"see\"\n  },\n  \"a\": \"aye\"\n}\n"), 2, true)
	require.NoError(t, err)

	require.NoError(t, doc.SetString([]string{"b", "aa"}, "double aye"))
	after, err := doc.Render()
	require.NoError(t, err)

	require.Equal(t, "{\n  \"a\": \"aye\",\n  \"b\": {\n    \"aa\": \"double aye\",\n    \"c\": \"see\",\n    \"d\": \"dee\"\n  }\n}\n", string(after))
}

func TestJSONEditPreservesTrailingNewline(t *testing.T) {
	doc, err := Parse([]byte("{\n\t\"a\": \"aye\"\n}"), 2, false)
	require.NoError(t, err)

	require.NoError(t, doc.SetString([]string{"b"}, "bee"))
	after, err := doc.Render()
	require.NoError(t, err)

	require.Equal(t, "{\n\t\"a\": \"aye\",\n\t\"b\": \"bee\"\n}", string(after))
}

func TestJSONEditPreservesRawScalars(t *testing.T) {
	doc, err := Parse([]byte("{\n  \"n\": 1.2300,\n  \"ok\": true,\n  \"items\": [null, 2]\n}\n"), 2, false)
	require.NoError(t, err)

	after, err := doc.Render()
	require.NoError(t, err)

	require.Equal(t, "{\n  \"n\": 1.2300,\n  \"ok\": true,\n  \"items\": [\n    null,\n    2\n  ]\n}\n", string(after))
}

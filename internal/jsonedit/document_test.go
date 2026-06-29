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

func TestJSONEditDeletePrunesEmptyParents(t *testing.T) {
	doc, err := Parse([]byte("{\n  \"keep\": \"yes\",\n  \"nested\": {\n    \"remove\": {\n      \"me\": \"bye\"\n    }\n  }\n}\n"), 2, false)
	require.NoError(t, err)

	deleted, err := doc.Delete([]string{"nested", "remove", "me"})
	require.NoError(t, err)
	require.True(t, deleted)
	after, err := doc.Render()
	require.NoError(t, err)

	require.Equal(t, "{\n  \"keep\": \"yes\"\n}\n", string(after))
}

func TestJSONEditDeleteErrorsOnNonObjectParent(t *testing.T) {
	doc, err := Parse([]byte("{\n  \"items\": []\n}\n"), 2, false)
	require.NoError(t, err)

	deleted, err := doc.Delete([]string{"items", "name"})

	require.Error(t, err)
	require.False(t, deleted)
}

func TestJSONEditStringReadsOnlyStringValues(t *testing.T) {
	doc, err := Parse([]byte(`{"title":"Hello","nested":{"label":"Name"},"count":1,"items":[]}`), 2, false)
	require.NoError(t, err)

	tests := []struct {
		name string
		path []string
		want string
		ok   bool
	}{
		{name: "top level string", path: []string{"title"}, want: "Hello", ok: true},
		{name: "nested string", path: []string{"nested", "label"}, want: "Name", ok: true},
		{name: "missing path", path: []string{"missing"}},
		{name: "raw scalar is not string", path: []string{"count"}},
		{name: "array is not string", path: []string{"items"}},
		{name: "empty path points at root object", path: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok, err := doc.String(tt.path)
			require.NoError(t, err)
			require.Equal(t, tt.ok, ok)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestJSONEditStringErrorsWhenPathTraversesScalar(t *testing.T) {
	doc, err := Parse([]byte(`{"title":"Hello"}`), 2, false)
	require.NoError(t, err)

	got, ok, err := doc.String([]string{"title", "text"})

	require.NoError(t, err)
	require.False(t, ok)
	require.Empty(t, got)
}

func TestJSONEditRenameStringMovesValueAndPrunesParents(t *testing.T) {
	doc, err := Parse([]byte("{\n  \"login\": {\n    \"title\": \"Log in\"\n  },\n  \"other\": \"Keep\"\n}\n"), 2, false)
	require.NoError(t, err)

	renamed, err := doc.RenameString([]string{"login", "title"}, []string{"auth", "heading"}, false)
	require.NoError(t, err)
	require.True(t, renamed)
	after, err := doc.Render()
	require.NoError(t, err)

	require.Equal(t, "{\n  \"other\": \"Keep\",\n  \"auth\": {\n    \"heading\": \"Log in\"\n  }\n}\n", string(after))
}

func TestJSONEditRenameStringConflictPolicies(t *testing.T) {
	tests := []struct {
		name      string
		overwrite bool
		wantErr   error
		wantTo    string
		wantFrom  bool
	}{
		{name: "reject existing destination", overwrite: false, wantErr: ErrPathExists, wantTo: "Existing", wantFrom: true},
		{name: "overwrite existing destination", overwrite: true, wantTo: "Log in", wantFrom: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := Parse([]byte(`{"login":{"title":"Log in"},"auth":{"heading":"Existing"}}`), 2, false)
			require.NoError(t, err)

			renamed, err := doc.RenameString([]string{"login", "title"}, []string{"auth", "heading"}, tt.overwrite)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.False(t, renamed)
			} else {
				require.NoError(t, err)
				require.True(t, renamed)
			}
			toValue, ok, err := doc.String([]string{"auth", "heading"})
			require.NoError(t, err)
			require.True(t, ok)
			require.Equal(t, tt.wantTo, toValue)
			_, fromOK, err := doc.String([]string{"login", "title"})
			require.NoError(t, err)
			require.Equal(t, tt.wantFrom, fromOK)
		})
	}
}

func TestJSONEditRenameStringNoopsAndInvalidMoves(t *testing.T) {
	tests := []struct {
		name    string
		from    []string
		to      []string
		wantErr error
	}{
		{name: "same path", from: []string{"login", "title"}, to: []string{"login", "title"}},
		{name: "destination descends from source", from: []string{"login"}, to: []string{"login", "title"}, wantErr: ErrAncestorDescendantPath},
		{name: "source descends from destination", from: []string{"login", "title"}, to: []string{"login"}, wantErr: ErrAncestorDescendantPath},
		{name: "missing source", from: []string{"missing"}, to: []string{"auth", "heading"}},
		{name: "non string source", from: []string{"count"}, to: []string{"auth", "heading"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := Parse([]byte(`{"login":{"title":"Log in"},"count":1}`), 2, false)
			require.NoError(t, err)

			renamed, err := doc.RenameString(tt.from, tt.to, false)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			require.False(t, renamed)
		})
	}
}

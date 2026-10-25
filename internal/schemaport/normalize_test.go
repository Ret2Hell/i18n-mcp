package schemaport_test

import (
	"encoding/json"
	"github.com/Ret2Hell/i18n-mcp/internal/config"
	"github.com/Ret2Hell/i18n-mcp/internal/schemaport"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/stretchr/testify/require"
	"testing"
)

func clone(t *testing.T, s *jsonschema.Schema) *jsonschema.Schema {
	t.Helper()
	raw, err := json.Marshal(s)
	require.NoError(t, err)
	var out jsonschema.Schema
	require.NoError(t, json.Unmarshal(raw, &out))
	return &out
}

func TestNormalizationPreservesNullableConstraintsAndExistingAnyOf(t *testing.T) {
	two := any(2)
	schemas := []*jsonschema.Schema{
		{Types: []string{"null", "array"}, Items: &jsonschema.Schema{Type: "string"}},
		{Types: []string{"integer", "null"}, AnyOf: []*jsonschema.Schema{{Const: &two}, {Type: "null"}}},
	}
	for _, source := range schemas {
		normalized := clone(t, source)
		schemaport.Normalize(normalized)
		before, err := source.Resolve(nil)
		require.NoError(t, err)
		after, err := normalized.Resolve(nil)
		require.NoError(t, err)
		for _, value := range []any{nil, false, 0, 1, 2, "a", []any{}, []any{"a"}, []any{2}, map[string]any{}} {
			require.Equal(t, before.Validate(value) == nil, after.Validate(value) == nil, "%#v", value)
		}
		require.Empty(t, normalized.Types)
	}
}

func TestConfigurationOmissionNullAndFalseSemanticsAreUnchanged(t *testing.T) {
	original, err := jsonschema.For[config.FileInput](nil)
	require.NoError(t, err)
	normalized := clone(t, original)
	schemaport.Normalize(normalized)
	before, err := original.Resolve(nil)
	require.NoError(t, err)
	after, err := normalized.Resolve(nil)
	require.NoError(t, err)
	for _, extra := range []string{``, `,"format":null`, `,"format":{}`, `,"format":{"trailingNewline":false,"indent":0}`, `,"ci":{"failOnMissing":false}`, `,"targetLocales":[]`, `,"targetLocales":null`, `,"format":{"indent":"bad"}`, `,"unknown":true`} {
		var value any
		require.NoError(t, json.Unmarshal([]byte(`{"sourceLocale":"en","localeFiles":["messages/{locale}.json"]`+extra+`}`), &value))
		require.Equal(t, before.Validate(value) == nil, after.Validate(value) == nil, extra)
	}
}

func TestDefaultsAndExamplesAreDataNotSchemas(t *testing.T) {
	raw := json.RawMessage(`{"type":["array","null"]}`)
	s := &jsonschema.Schema{Types: []string{"null", "object"}, Default: raw, Examples: []any{map[string]any{"type": []any{"string", "null"}}}}
	schemaport.Normalize(s)
	require.JSONEq(t, string(raw), string(s.Default))
	require.Equal(t, []any{"string", "null"}, s.Examples[0].(map[string]any)["type"])
}

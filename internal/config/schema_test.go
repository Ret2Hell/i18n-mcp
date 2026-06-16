package config

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSchemaReturnsConfigSchema(t *testing.T) {
	schema, err := Schema()
	require.NoError(t, err)
	require.NotNil(t, schema)
	require.Equal(t, "https://example.com/i18n-mcp.schema.json", schema.ID)
	require.Equal(t, "i18n MCP Config", schema.Title)
	require.NotEmpty(t, schema.Description)

	data, err := json.Marshal(schema)
	require.NoError(t, err)
	require.Contains(t, string(data), "sourceLocale")
	require.Contains(t, string(data), "targetLocales")
	require.Contains(t, string(data), "translation")
}

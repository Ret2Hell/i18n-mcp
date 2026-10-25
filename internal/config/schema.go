package config

import (
	"github.com/Ret2Hell/i18n-mcp/internal/schemaport"
	"github.com/google/jsonschema-go/jsonschema"
)

// Schema generates the JSON schema for File configuration.
func Schema() (*jsonschema.Schema, error) {
	schema, err := jsonschema.For[FileInput](nil)
	if err != nil {
		return nil, err
	}
	schemaport.Normalize(schema)
	schema.ID = "https://example.com/i18n-mcp.schema.json"
	schema.Title = "i18n MCP Config"
	schema.Description = "Configuration for the i18n MCP server."
	return schema, nil
}

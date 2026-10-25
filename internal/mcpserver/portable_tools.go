package mcpserver

import (
	"encoding/json"
	"reflect"

	"github.com/Ret2Hell/i18n-mcp/internal/schemaport"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func portableSchema[T any](provided any) *jsonschema.Schema {
	var schema *jsonschema.Schema
	if provided != nil {
		raw, err := json.Marshal(provided)
		if err != nil {
			panic(err)
		}
		if err := json.Unmarshal(raw, &schema); err != nil {
			panic(err)
		}
	} else {
		inferred, err := jsonschema.For[T](nil)
		if err != nil {
			panic(err)
		}
		schema = inferred
	}
	schemaport.Normalize(schema)
	return schema
}

// addTool normalizes the same schema that the SDK uses for validation, not just
// the catalog's displayed schema. Inference errors remain registration failures.
func addTool[In, Out any](server *mcp.Server, tool *mcp.Tool, handler mcp.ToolHandlerFor[In, Out]) {
	copyTool := *tool
	copyTool.InputSchema = portableSchema[In](tool.InputSchema)
	if tool.OutputSchema != nil || reflect.TypeFor[Out]() != reflect.TypeFor[any]() {
		copyTool.OutputSchema = portableSchema[Out](tool.OutputSchema)
	}
	mcp.AddTool(server, &copyTool, handler)
}

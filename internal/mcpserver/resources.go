package mcpserver

import (
	"context"
	"encoding/json"

	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerResources(s *mcp.Server, a *app.App) {
	s.AddResource(&mcp.Resource{
		URI:         "i18n://locales",
		Name:        "locales",
		Title:       "i18n Locale Inventory",
		Description: "Locale files, locales, namespaces, key counts, warnings, and duplicate namespace issues.",
		MIMEType:    "application/json",
		Annotations: &mcp.Annotations{Audience: []mcp.Role{"assistant"}, Priority: 0.9},
	}, readLocalesResource(a))
}

func readLocalesResource(a *app.App) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		if req.Params.URI != "i18n://locales" {
			return nil, mcp.ResourceNotFoundError(req.Params.URI)
		}
		inv, err := a.Locales.Inventory(ctx)
		if err != nil {
			return nil, err
		}
		inv.Units = nil
		return jsonResource(req.Params.URI, inv)
	}
}

func jsonResource(uri string, value any) (*mcp.ReadResourceResult, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      uri,
			MIMEType: "application/json",
			Text:     string(data),
		}},
	}, nil
}

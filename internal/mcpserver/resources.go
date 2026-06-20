package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"

	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/Ret2Hell/i18n-mcp/internal/locale"
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

	s.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "i18n://locales/{locale}/{namespace}",
		Name:        "locale_namespace",
		Title:       "Locale Namespace",
		Description: "Read one locale namespace as raw JSON files and flattened translation units.",
		MIMEType:    "application/json",
		Annotations: &mcp.Annotations{Audience: []mcp.Role{"assistant"}, Priority: 0.9},
	}, readLocaleNamespaceResource(a))
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

func readLocaleNamespaceResource(a *app.App) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		localeCode, namespace, ok := parseLocaleNamespaceURI(req.Params.URI)
		if !ok {
			return nil, mcp.ResourceNotFoundError(req.Params.URI)
		}

		content, err := a.Locales.Namespace(ctx, localeCode, namespace)
		if err != nil {
			if errors.Is(err, locale.ErrNamespaceNotFound) {
				return nil, mcp.ResourceNotFoundError(req.Params.URI)
			}
			return nil, err
		}
		return jsonResource(req.Params.URI, content)
	}
}

func parseLocaleNamespaceURI(rawURI string) (string, string, bool) {
	parsed, err := url.Parse(rawURI)
	if err != nil || parsed.Scheme != "i18n" || parsed.Host != "locales" {
		return "", "", false
	}
	path, _ := strings.CutPrefix(parsed.Path, "/")
	localePart, namespacePart, ok := strings.Cut(path, "/")
	if !ok || strings.Contains(namespacePart, "/") {
		return "", "", false
	}
	localeCode, err := url.PathUnescape(localePart)
	if err != nil {
		return "", "", false
	}
	namespace, err := url.PathUnescape(namespacePart)
	if err != nil {
		return "", "", false
	}
	if localeCode == "" || namespace == "" {
		return "", "", false
	}
	return localeCode, namespace, true
}

package mcpserver

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/Ret2Hell/i18n-mcp/internal/diff"
	"github.com/Ret2Hell/i18n-mcp/internal/locale"
	"github.com/Ret2Hell/i18n-mcp/internal/pagination"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func pageURI(kind string, input any) string {
	raw, _ := json.Marshal(input)
	return "i18n://pages/" + kind + "/" + base64.RawURLEncoding.EncodeToString(raw)
}

func setDiffContinuation(report *diff.Report, in diff.PageInput) {
	if report.Page != nil && report.Page.Truncated {
		next := in
		next.Cursor = report.Page.NextCursor
		report.Page.NextResource = pageURI("diff", next)
	}
}

func registerPageResources(s *mcp.Server, a *app.App) {
	for _, kind := range []string{"diff", "locales"} {
		s.AddResourceTemplate(&mcp.ResourceTemplate{
			URITemplate: "i18n://pages/" + kind + "/{query}", Name: kind + "_page", Title: "Bounded " + kind + " detail page", MIMEType: "application/json",
		}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			prefix := "i18n://pages/" + kind + "/"
			encoded, ok := strings.CutPrefix(req.Params.URI, prefix)
			if !ok || len(encoded) > pagination.ResponseBudget {
				return nil, mcp.ResourceNotFoundError(req.Params.URI)
			}
			raw, err := base64.RawURLEncoding.DecodeString(encoded)
			if err != nil {
				return nil, fmt.Errorf("invalid page resource")
			}
			if kind == "diff" {
				var in diff.PageInput
				if err := json.Unmarshal(raw, &in); err != nil {
					return nil, err
				}
				report, err := a.Diff.Page(ctx, a.RequestStateSigner, a.ProjectRoot, in)
				if err != nil {
					return nil, err
				}
				setDiffContinuation(&report, in)
				return jsonResource(req.Params.URI, report)
			}
			var in locale.PageInput
			if err := json.Unmarshal(raw, &in); err != nil {
				return nil, err
			}
			return localePageResource(ctx, a, req.Params.URI, in)
		})
	}
}

func localePageResource(ctx context.Context, a *app.App, uri string, in locale.PageInput) (*mcp.ReadResourceResult, error) {
	inv, err := a.Locales.Page(ctx, a.RequestStateSigner, a.ProjectRoot, in)
	if err != nil {
		return nil, err
	}
	if inv.Page != nil && inv.Page.Truncated {
		next := in
		next.Cursor = inv.Page.NextCursor
		inv.Page.NextResource = pageURI("locales", next)
	}
	return jsonResource(uri, inv)
}

package mcpserver

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/Ret2Hell/i18n-mcp/internal/locale"
	resourceuri "github.com/Ret2Hell/i18n-mcp/internal/resources"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerResources(s *mcp.Server, a *app.App) {
	s.AddResource(&mcp.Resource{
		URI:         resourceuri.LocalesURI,
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

	s.AddResource(&mcp.Resource{
		URI:         resourceuri.DiffURI,
		Name:        "analysis_diff",
		Title:       "i18n Key Diff Analysis",
		Description: "Latest key diff report with missing, stale, extra, invalid, unknown, and current statuses.",
		MIMEType:    "application/json",
		Annotations: &mcp.Annotations{Audience: []mcp.Role{"assistant"}, Priority: 0.9},
	}, readDiffResource(a))

	s.AddResource(&mcp.Resource{
		URI:         resourceuri.UsageURI,
		Name:        "analysis_usage",
		Title:       "i18n Usage Analysis",
		Description: "Latest static translation key usage scan with evidence and dynamic hints.",
		MIMEType:    "application/json",
		Annotations: &mcp.Annotations{Audience: []mcp.Role{"assistant"}, Priority: 0.8},
	}, readUsageResource(a))

	s.AddResource(&mcp.Resource{
		URI:         resourceuri.DeadKeysURI,
		Name:        "analysis_dead_keys",
		Title:       "i18n Dead-Key Analysis",
		Description: "Latest dead-key classification report with confidence and evidence.",
		MIMEType:    "application/json",
		Annotations: &mcp.Annotations{Audience: []mcp.Role{"assistant"}, Priority: 0.8},
	}, readDeadKeysResource(a))

	s.AddResource(&mcp.Resource{
		URI:         resourceuri.LatestPlanURI,
		Name:        "translation_plan_latest",
		Title:       "Latest i18n Translation Plan",
		Description: "Latest prepared translation batch for missing and stale locale keys.",
		MIMEType:    "application/json",
		Annotations: &mcp.Annotations{Audience: []mcp.Role{"assistant"}, Priority: 0.9},
	}, readTranslationPlanResource(a))

	s.AddResource(&mcp.Resource{
		URI:         resourceuri.LatestReportURI,
		Name:        "reports_latest",
		Title:       "Latest i18n Audit Report",
		Description: "Latest generated i18n audit report with rendered text and structured data.",
		MIMEType:    "application/json",
		Annotations: &mcp.Annotations{Audience: []mcp.Role{"assistant"}, Priority: 0.8},
	}, readLatestReportResource(a))
}

func readLocalesResource(a *app.App) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		if req.Params.URI != resourceuri.LocalesURI {
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
	return resourceuri.ParseLocaleURI(rawURI)
}

func readDiffResource(a *app.App) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		if req.Params.URI != resourceuri.DiffURI {
			return nil, mcp.ResourceNotFoundError(req.Params.URI)
		}
		report, err := a.Diff.Analyze(ctx)
		if err != nil {
			return nil, err
		}
		return jsonResource(req.Params.URI, report)
	}
}

func readUsageResource(a *app.App) mcp.ResourceHandler {
	return latestJSONResource(
		resourceuri.UsageURI,
		func(ctx context.Context) (any, bool, error) { return a.Scanner.Latest(ctx) },
		map[string]any{"report": nil, "message": "no usage scan has been run yet"},
	)
}

func readDeadKeysResource(a *app.App) mcp.ResourceHandler {
	return latestJSONResource(
		resourceuri.DeadKeysURI,
		func(ctx context.Context) (any, bool, error) { return a.DeadKeys.Latest(ctx) },
		map[string]any{"report": nil, "message": "no dead-key report has been run yet"},
	)
}

func readTranslationPlanResource(a *app.App) mcp.ResourceHandler {
	return latestJSONResource(
		resourceuri.LatestPlanURI,
		func(ctx context.Context) (any, bool, error) { return a.Translation.LatestPlan(ctx) },
		map[string]any{"batch": nil, "message": "no translation plan has been created yet"},
	)
}

func readLatestReportResource(a *app.App) mcp.ResourceHandler {
	return latestJSONResource(
		resourceuri.LatestReportURI,
		func(ctx context.Context) (any, bool, error) { return a.Reports.Latest(ctx) },
		map[string]any{"report": nil, "message": "no report has been generated yet"},
	)
}

func latestJSONResource(uri string, latest func(context.Context) (any, bool, error), empty any) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		if req.Params.URI != uri {
			return nil, mcp.ResourceNotFoundError(req.Params.URI)
		}
		value, ok, err := latest(ctx)
		if err != nil {
			return nil, mcp.ResourceNotFoundError(req.Params.URI)
		}
		if !ok {
			value = empty
		}
		return jsonResource(req.Params.URI, value)
	}
}

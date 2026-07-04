package mcpserver

import (
	"context"

	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/Ret2Hell/i18n-mcp/internal/config"
	"github.com/Ret2Hell/i18n-mcp/internal/mcpadapter"
	"github.com/Ret2Hell/i18n-mcp/internal/project"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ProjectDetectInput struct {
	ProjectRoot string `json:"projectRoot,omitzero" jsonschema:"project root to inspect; defaults to configured root and must stay inside configured root"`
	MaxDepth    int    `json:"maxDepth,omitzero" jsonschema:"reserved for deeper scans; zero uses default detection depth"`
}

type ProjectDetectOutput struct {
	ProjectRoot      string                        `json:"projectRoot" jsonschema:"resolved project root"`
	NextJS           project.NextJSHints           `json:"nextjs" jsonschema:"Next.js detection hints from detect4nextjs when applicable"`
	DetectedLibrary  string                        `json:"detectedLibrary,omitzero" jsonschema:"primary detected i18n library"`
	Libraries        []project.LibraryHint         `json:"libraries,omitzero" jsonschema:"all detected i18n libraries"`
	Layouts          []project.LocaleLayout        `json:"layouts,omitzero" jsonschema:"detected locale layouts"`
	LocaleFiles      []project.LocaleFileCandidate `json:"localeFiles,omitzero" jsonschema:"detected locale JSON files"`
	SourceCandidates []string                      `json:"sourceLocaleCandidates,omitzero" jsonschema:"candidate source locales ordered by confidence"`
	TargetLocales    []string                      `json:"targetLocales,omitzero" jsonschema:"detected target locales"`
	ProposedConfig   config.File                   `json:"proposedConfig" jsonschema:"proposed .i18n-mcp.json config; not written to disk"`
	Warnings         []string                      `json:"warnings,omitzero" jsonschema:"non-fatal detection warnings"`
}

func projectDetectTool(a *app.App) func(context.Context, *mcp.CallToolRequest, ProjectDetectInput) (*mcp.CallToolResult, ProjectDetectOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, in ProjectDetectInput) (*mcp.CallToolResult, ProjectDetectOutput, error) {
		d, err := a.Project.Detect(ctx, project.DetectOptions{
			ProjectRoot: in.ProjectRoot,
			MaxDepth:    in.MaxDepth,
			Progress:    mcpadapter.NewMCPProgressReporter(req, a.Logger),
		})
		if err != nil {
			return nil, ProjectDetectOutput{}, err
		}
		return nil, ProjectDetectOutput{
			ProjectRoot:      d.ProjectRoot,
			NextJS:           d.NextJS,
			DetectedLibrary:  d.DetectedLibrary,
			Libraries:        d.Libraries,
			Layouts:          d.Layouts,
			LocaleFiles:      d.LocaleFiles,
			SourceCandidates: d.SourceCandidates,
			TargetLocales:    d.TargetLocales,
			ProposedConfig:   d.ProposedConfig,
			Warnings:         d.Warnings,
		}, nil
	}
}

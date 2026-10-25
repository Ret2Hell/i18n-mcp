package mcpserver

import (
	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerTools(s *mcp.Server, a *app.App) {
	addTool(s, &mcp.Tool{
		Name:        "i18n.health",
		Title:       "i18n MCP Health",
		Description: "Return server version, configured project root, and health status.",
		Annotations: readOnly("i18n MCP Health"),
	}, healthTool(a))

	addTool(s, &mcp.Tool{
		Name:        "i18n.project.detect",
		Title:       "Detect i18n Project",
		Description: "Detect framework hints, i18n libraries, JSON locale layouts, locale candidates, and a proposed i18n MCP config.",
		Annotations: readOnly("Detect i18n Project"),
	}, projectDetectTool(a))

	addTool(s, &mcp.Tool{
		Name:        "i18n.config.get",
		Title:       "Get i18n MCP Config",
		Description: "Return resolved configuration, including defaults and config file origin.",
		Annotations: readOnly("Get i18n MCP Config"),
	}, configGetTool(a))

	addTool(s, &mcp.Tool{
		Name:        "i18n.config.validate",
		Title:       "Validate i18n MCP Config",
		Description: "Validate resolved configuration and return actionable errors and warnings.",
		Annotations: readOnly("Validate i18n MCP Config"),
	}, configValidateTool(a))

	addTool(s, new(mcp.Tool{
		Name:        "i18n.config.write",
		Title:       "Write i18n MCP Config",
		Description: "Preview or write .i18n-mcp.json. Dry-run by default. If preview.truncated is true, retrieve the complete patch resources and review them before applying with previewToken=preview.token.",
		Annotations: writeOp("Write i18n MCP Config", false, true),
	}), configWriteTool(a))

	addTool(s, &mcp.Tool{
		Name:        "i18n.locales.list",
		Title:       "List i18n Locales",
		Description: "List locale files, locales, namespaces, key counts, warnings, and duplicate namespace issues.",
		Annotations: readOnly("List i18n Locales"),
	}, localesListTool(a))

	addTool(s, &mcp.Tool{
		Name:        "i18n.keys.diff",
		Title:       "Diff Locale Keys",
		Description: "Return a bounded filtered diff page. Target defects (invalid) are separate from source-policy failures (source_invalid); sourceIssueCount counts distinct source keys. Follow page.nextCursor or page.nextResource.",
		Annotations: readOnly("Diff Locale Keys"),
	}, keysDiffTool(a))

	// Dead-key features are temporarily unregistered pending safer analysis.

	addTool(s, &mcp.Tool{
		Name:        "i18n.keys.rename",
		Title:       "Rename i18n Keys",
		Description: "Preview or rename exact locale keys and update state. Dry-run by default. Truncated previews require complete patch retrieval/review, then apply true with previewToken=preview.token.",
		Annotations: writeOp("Rename i18n Keys", true, true),
	}, keysRenameTool(a))

	addTool(s, &mcp.Tool{
		Name:        "i18n.translation.plan",
		Title:       "Plan Translations",
		Description: "Build a bounded translation batch (default 50, maximum 200) from missing and stale keys. Follow page.nextCursor or re-plan after apply. Does not generate translations.",
		Annotations: readOnly("Plan Translations"),
	}, translationPlanTool(a))

	addTool(s, &mcp.Tool{
		Name:        "i18n.translation.validate",
		Title:       "Validate Translations",
		Description: "Validate proposed translations against current source values, placeholders, tags, ICU shape, and batch constraints.",
		Annotations: readOnly("Validate Translations"),
	}, translationValidateTool(a))

	addTool(s, &mcp.Tool{
		Name:        "i18n.translation.apply",
		Title:       "Apply Translations",
		Description: "Validate translations and preview locale patches plus planned state-update counts. Dry-run by default. For truncated previews, retrieve/review complete patches before apply true with previewToken=preview.token.",
		Annotations: writeOp("Apply Translations", false, true),
	}, translationApplyTool(a))

	addTool(s, &mcp.Tool{
		Name:        "i18n.state.rebuild",
		Title:       "Rebuild i18n State",
		Description: "Preview state rebuild counts without embedding state entries; write .i18n-mcp/state.json only with apply true.",
		Annotations: writeOp("Rebuild i18n State", false, true),
	}, stateRebuildTool(a))

	addTool(s, &mcp.Tool{
		Name:        "i18n.report.generate",
		Title:       "Generate i18n Audit Report",
		Description: "Generate a compact JSON or Markdown audit summary with a link to bounded diff details. Source usage and dead-key analysis are temporarily disabled.",
		Annotations: readOnly("Generate i18n Audit Report"),
	}, reportGenerateTool(a))
}

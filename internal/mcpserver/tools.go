package mcpserver

import (
	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerTools(s *mcp.Server, a *app.App) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "i18n.health",
		Title:       "i18n MCP Health",
		Description: "Return server version, configured project root, and health status.",
		Annotations: readOnly("i18n MCP Health"),
	}, healthTool(a))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "i18n.project.detect",
		Title:       "Detect i18n Project",
		Description: "Detect framework hints, i18n libraries, JSON locale layouts, locale candidates, and a proposed i18n MCP config.",
		Annotations: readOnly("Detect i18n Project"),
	}, projectDetectTool(a))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "i18n.config.get",
		Title:       "Get i18n MCP Config",
		Description: "Return resolved configuration, including defaults and config file origin.",
		Annotations: readOnly("Get i18n MCP Config"),
	}, configGetTool(a))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "i18n.config.validate",
		Title:       "Validate i18n MCP Config",
		Description: "Validate resolved configuration and return actionable errors and warnings.",
		Annotations: readOnly("Validate i18n MCP Config"),
	}, configValidateTool(a))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "i18n.locales.list",
		Title:       "List i18n Locales",
		Description: "List locale files, locales, namespaces, key counts, warnings, and duplicate namespace issues.",
		Annotations: readOnly("List i18n Locales"),
	}, localesListTool(a))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "i18n.keys.diff",
		Title:       "Diff Locale Keys",
		Description: "Compare source and target locales for missing, stale, extra, invalid, unknown, and current keys.",
		Annotations: readOnly("Diff Locale Keys"),
	}, keysDiffTool(a))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "i18n.translation.plan",
		Title:       "Plan Translations",
		Description: "Build a deterministic translation batch from missing and stale locale keys. Does not generate translations.",
		Annotations: readOnly("Plan Translations"),
	}, translationPlanTool(a))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "i18n.translation.validate",
		Title:       "Validate Translations",
		Description: "Validate proposed translations against current source values, placeholders, tags, ICU shape, and batch constraints.",
		Annotations: readOnly("Validate Translations"),
	}, translationValidateTool(a))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "i18n.state.rebuild",
		Title:       "Rebuild i18n State",
		Description: "Rebuild .i18n-mcp/state.json from existing locale files. Dry-run by default; writes only with apply true.",
		Annotations: writeOp("Rebuild i18n State", false, true),
	}, stateRebuildTool(a))
}

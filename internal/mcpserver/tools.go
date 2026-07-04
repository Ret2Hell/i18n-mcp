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

	mcp.AddTool(s, new(mcp.Tool{
		Name:        "i18n.config.write",
		Title:       "Write i18n MCP Config",
		Description: "Preview or write .i18n-mcp.json from explicit config input. Dry-run by default; writes only with apply true.",
		Annotations: writeOp("Write i18n MCP Config", false, true),
	}), configWriteTool(a))

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
		Name:        "i18n.keys.usage_scan",
		Title:       "Scan i18n Key Usage",
		Description: "Scan TS, TSX, JS, JSX, MJS, and CJS source files for translation key usage evidence and dynamic key hints.",
		Annotations: readOnly("Scan i18n Key Usage"),
	}, usageScanTool(a))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "i18n.keys.dead_report",
		Title:       "Report Dead i18n Keys",
		Description: "Classify source locale keys as used, probably unused, maybe dynamic, ignored, or kept using static usage evidence.",
		Annotations: readOnly("Report Dead i18n Keys"),
	}, deadReportTool(a))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "i18n.keys.prune",
		Title:       "Prune Dead i18n Keys",
		Description: "Remove exact selected dead keys from locale JSON files. Dry-run by default; writes only with apply true.",
		Annotations: writeOp("Prune Dead i18n Keys", true, true),
	}, keysPruneTool(a))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "i18n.keys.rename",
		Title:       "Rename i18n Keys",
		Description: "Rename or move exact locale keys across locale JSON files and update state. Dry-run by default; writes only with apply true.",
		Annotations: writeOp("Rename i18n Keys", true, true),
	}, keysRenameTool(a))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "i18n.translation.plan",
		Title:       "Plan Translations",
		Description: "Build a deterministic translation batch from missing and stale locale keys. Does not generate translations.",
		Annotations: readOnly("Plan Translations"),
	}, translationPlanTool(a))

	mcp.AddTool(s, new(mcp.Tool{
		Name:        "i18n.translation.generate",
		Title:       "Generate Translation Proposals",
		Description: "Generate validated translation proposals using MCP sampling without writing locale files or state.",
		Annotations: readOnly("Generate Translation Proposals"),
	}), translationGenerateTool(a))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "i18n.translation.validate",
		Title:       "Validate Translations",
		Description: "Validate proposed translations against current source values, placeholders, tags, ICU shape, and batch constraints.",
		Annotations: readOnly("Validate Translations"),
	}, translationValidateTool(a))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "i18n.translation.apply",
		Title:       "Apply Translations",
		Description: "Validate translations, preview patches, and write locale files plus state only when apply is true. Dry-run by default.",
		Annotations: writeOp("Apply Translations", false, true),
	}, translationApplyTool(a))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "i18n.state.rebuild",
		Title:       "Rebuild i18n State",
		Description: "Rebuild .i18n-mcp/state.json from existing locale files. Dry-run by default; writes only with apply true.",
		Annotations: writeOp("Rebuild i18n State", false, true),
	}, stateRebuildTool(a))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "i18n.report.generate",
		Title:       "Generate i18n Audit Report",
		Description: "Generate a deterministic JSON or Markdown i18n audit report from config, inventory, diff, usage, and dead-key analysis.",
		Annotations: readOnly("Generate i18n Audit Report"),
	}, reportGenerateTool(a))
}

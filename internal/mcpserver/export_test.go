package mcpserver

import (
	"github.com/Ret2Hell/i18n-mcp/internal/app"
	resourceuri "github.com/Ret2Hell/i18n-mcp/internal/resources"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// RegisterDeadKeyFeaturesForTest preserves coverage of the dormant implementation.
// It is compiled only into tests; production servers cannot enable these features.
func RegisterDeadKeyFeaturesForTest(s *mcp.Server, a *app.App) {
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

	s.AddPrompt(&mcp.Prompt{
		Name:        "i18n_audit_dead_keys",
		Title:       "Audit Dead i18n Keys",
		Description: "Review dead-key candidates and dynamic hints before creating a prune preview.",
		Arguments: []*mcp.PromptArgument{
			{Name: "confidenceThreshold", Title: "Confidence Threshold", Description: "minimum confidence to consider, such as medium or high"},
			{Name: "includeDynamic", Title: "Include Dynamic", Description: "whether maybe_dynamic keys should be included in review"},
		},
	}, auditDeadKeysPrompt(a))

}

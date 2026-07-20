package mcpserver

import (
	"context"
	"fmt"
	"strings"

	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerPrompts(s *mcp.Server, a *app.App) {
	s.AddPrompt(&mcp.Prompt{
		Name:        "i18n_project_bootstrap",
		Title:       "Bootstrap i18n MCP Project",
		Description: "Guide project detection, config creation, and state bootstrap for an existing Next.js i18n project.",
		Arguments: []*mcp.PromptArgument{
			{Name: "projectPath", Title: "Project Path", Description: "project root path or current workspace root"},
		},
	}, projectBootstrapPrompt(a))

	s.AddPrompt(&mcp.Prompt{
		Name:        "i18n_translate_batch",
		Title:       "Translate i18n Batch",
		Description: "Translate a prepared i18n batch while preserving placeholders, tags, ICU arguments, and glossary terms.",
		Arguments: []*mcp.PromptArgument{
			{Name: "locale", Title: "Locale", Description: "target locale to translate", Required: true},
			{Name: "batchId", Title: "Batch ID", Description: "translation batch id from i18n.translation.plan", Required: true},
			{Name: "tone", Title: "Tone", Description: "optional tone guidance"},
			{Name: "domain", Title: "Domain", Description: "optional product or business domain"},
		},
	}, translateBatchPrompt(a))

	s.AddPrompt(&mcp.Prompt{
		Name:        "i18n_review_translations",
		Title:       "Review i18n Translations",
		Description: "Review proposed translations for accuracy, consistency, placeholders, tags, ICU shape, and source drift.",
		Arguments: []*mcp.PromptArgument{
			{Name: "locale", Title: "Locale", Description: "target locale to review", Required: true},
			{Name: "namespace", Title: "Namespace", Description: "namespace to focus on"},
			{Name: "strictness", Title: "Strictness", Description: "review strictness: normal, strict, or blocking"},
		},
	}, reviewTranslationsPrompt(a))

	s.AddPrompt(&mcp.Prompt{
		Name:        "i18n_audit_dead_keys",
		Title:       "Audit Dead i18n Keys",
		Description: "Review dead-key candidates and dynamic hints before creating a prune preview.",
		Arguments: []*mcp.PromptArgument{
			{Name: "confidenceThreshold", Title: "Confidence Threshold", Description: "minimum confidence to consider, such as medium or high"},
			{Name: "includeDynamic", Title: "Include Dynamic", Description: "whether maybe_dynamic keys should be included in review"},
		},
	}, auditDeadKeysPrompt(a))

	s.AddPrompt(new(mcp.Prompt{
		Name:        "i18n_ci_report_summary",
		Title:       "Summarize i18n CI Report",
		Description: "Summarize an i18n audit report for CI, pull requests, or release review.",
		Arguments: []*mcp.PromptArgument{
			{Name: "reportUri", Title: "Report URI", Description: "report resource URI, defaults to i18n://reports/latest"},
			{Name: "audience", Title: "Audience", Description: "summary audience: developer, reviewer, manager, or ci"},
		},
	}), ciReportSummaryPrompt(a))

	s.AddPrompt(&mcp.Prompt{
		Name:        "i18n_add_feature_keys",
		Title:       "Add Feature i18n Keys",
		Description: "Plan source keys and translations for a new feature while preserving locale conventions.",
		Arguments: []*mcp.PromptArgument{
			{Name: "namespace", Title: "Namespace", Description: "namespace for the feature keys", Required: true},
			{Name: "featureDescription", Title: "Feature Description", Description: "short description of the feature", Required: true},
			{Name: "locales", Title: "Locales", Description: "comma-separated target locales to consider"},
		},
	}, addFeatureKeysPrompt(a))
}

func promptArg(req *mcp.GetPromptRequest, name string) string {
	if req == nil || req.Params == nil || req.Params.Arguments == nil {
		return ""
	}
	return strings.TrimSpace(req.Params.Arguments[name])
}

func textPrompt(description string, text string) (*mcp.GetPromptResult, error) {
	return &mcp.GetPromptResult{
		Description: description,
		Messages: []*mcp.PromptMessage{{
			Role:    "user",
			Content: &mcp.TextContent{Text: strings.TrimSpace(text)},
		}},
	}, nil
}

func projectBootstrapPrompt(a *app.App) mcp.PromptHandler {
	return func(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		projectPath := optionalText(promptArg(req, "projectPath"), a.ProjectRoot)

		text := fmt.Sprintf(`Bootstrap this project for i18n MCP management.

Project path: %s

Workflow:
- Call i18n.project.detect and inspect locale layout, source locale candidates, target locale candidates, and framework hints.
- Call i18n.config.get to see resolved defaults and whether .i18n-mcp.json exists.
- If config is missing or wrong, call i18n.config.write with dryRun true first.
- Ask the user to review the config patch before applying it.
- After config is correct, call i18n.config.validate.
- Call i18n.locales.list to verify files, namespaces, and key counts.
- Call i18n.state.rebuild with dry-run first to preview sidecar state entries.
- Apply state rebuild only after the user agrees.
- Finish by calling i18n.keys.diff and summarizing missing, stale, invalid, unknown, and extra keys.

Safety rules:
- Do not write config or state without explicit user confirmation.
- Do not modify locale JSON files during bootstrap.
- Explain any detection ambiguity before choosing sourceLocale or localeFiles patterns.`, projectPath)

		return textPrompt("Bootstrap project configuration and state safely.", text)
	}
}

func addFeatureKeysPrompt(_ *app.App) mcp.PromptHandler {
	return func(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		namespace := promptArg(req, "namespace")
		featureDescription := promptArg(req, "featureDescription")
		locales := optionalText(promptArg(req, "locales"), "configured target locales")

		text := fmt.Sprintf(`Plan i18n keys for a new feature.

Namespace: %s
Feature: %s
Target locales: %s

Workflow:
- Read i18n://locales and the namespace resource for existing naming conventions.
- Propose concise source keys with stable semantic names, not UI copy as keys.
- Group keys by screen or component when useful.
- Avoid duplicating existing keys; call i18n.locales.list and inspect existing namespace keys first.
- If source keys need to be added, describe the exact source locale JSON changes for user review.
- For target translations, prepare proposals in the same shape used by i18n.translation.validate.
- Validate target proposals with i18n.translation.validate before any apply.
- Use i18n.translation.apply dry-run before write mode.

Return a plan with source key additions, target translation proposals, and any open questions about product terminology.`, namespace, featureDescription, locales)

		return textPrompt("Plan new feature keys and translations.", text)
	}
}

func ciReportSummaryPrompt(_ *app.App) mcp.PromptHandler {
	return func(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		reportURI := optionalText(promptArg(req, "reportUri"), "i18n://reports/latest")
		audience := optionalText(promptArg(req, "audience"), "developer")

		text := fmt.Sprintf(`Summarize the i18n audit report for audience: %s.

Report resource: %s

Summary requirements:
- Lead with blocking issues that should fail CI.
- Include counts for missing, stale, invalid, extra, and probably unused keys.
- Separate warnings from errors.
- Mention affected locales and namespaces.
- Include exact tool or command suggestions for remediation.
- Keep the summary concise enough for a pull request comment.

If the report resource is missing, call i18n.report.generate or run the audit command first.`, audience, reportURI)

		return textPrompt("Summarize an i18n audit report.", text)
	}
}

func auditDeadKeysPrompt(_ *app.App) mcp.PromptHandler {
	return func(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		threshold := optionalText(promptArg(req, "confidenceThreshold"), "medium")
		includeDynamic := optionalText(promptArg(req, "includeDynamic"), "false")

		text := fmt.Sprintf(`Audit likely dead i18n keys before pruning.

Inputs:
- Confidence threshold: %s
- Include maybe_dynamic keys in review: %s
- Usage resource: i18n://analysis/usage
- Dead-key resource: i18n://analysis/dead-keys

Workflow:
- If no recent usage report exists, call i18n.keys.usage_scan.
- Call i18n.keys.dead_report with includeUsed false unless the user asks for a full audit.
- Treat probably_unused keys as candidates, not proof.
- Treat maybe_dynamic, ignored, and kept keys as unsafe for pruning by default.
- Inspect dynamic hints before recommending prune.
- For any prune recommendation, call i18n.keys.prune with dry-run first and exact namespace/key pairs only.
- Ask the user to review the patch preview before apply true.

Return a concise audit summary with candidate keys grouped by namespace, confidence, reasons, and recommended next tool call.`, threshold, includeDynamic)

		return textPrompt("Audit dead-key candidates before prune.", text)
	}
}

func reviewTranslationsPrompt(_ *app.App) mcp.PromptHandler {
	return func(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		localeCode := promptArg(req, "locale")
		namespace := promptArg(req, "namespace")
		strictness := optionalText(promptArg(req, "strictness"), "normal")

		text := fmt.Sprintf(`Review proposed i18n translations for locale %s.

Focus:
- Namespace: %s
- Strictness: %s
- Locale inventory: i18n://locales
- Locale namespace resource pattern: i18n://locales/{locale}/{namespace}
- Latest translation plan: i18n://translation/plan/latest

Review checklist:
- Validate every proposal with i18n.translation.validate before apply.
- Reject source drift unless the user explicitly accepts it.
- Check placeholder parity, including repeated placeholders.
- Check HTML-like tag preservation and nesting.
- Check ICU argument names and plural or select shape.
- Check that translations are natural for the locale and not merely copied from source.
- Check glossary and style-guide guidance from the latest batch when present.
- Report risky items separately from stylistic suggestions.

If the proposals pass review, return the same JSON proposal shape expected by i18n.translation.validate. Do not write files.`, localeCode, optionalText(namespace, "all namespaces"), strictness)

		return textPrompt("Review proposed translations before validation and apply.", text)
	}
}

func translateBatchPrompt(a *app.App) mcp.PromptHandler {
	return func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		localeCode := promptArg(req, "locale")
		batchID := promptArg(req, "batchId")
		tone := promptArg(req, "tone")
		domain := promptArg(req, "domain")

		batchSummary := "No latest batch is currently available. Call i18n.translation.plan first, or read i18n://translation/plan/latest after a batch has been prepared."
		if batch, ok, _ := a.Translation.LatestPlan(ctx); ok {
			batchSummary = fmt.Sprintf("Latest batch: %s with %d item(s), source locale %s, target locales %s.", batch.BatchID, len(batch.Items), batch.SourceLocale, strings.Join(batch.TargetLocales, ", "))
		}

		text := fmt.Sprintf(`Translate i18n batch %s into locale %s.

Context:
- %s
- Batch resource: i18n://translation/plan/latest
- Diff resource: i18n://analysis/diff
- Domain guidance: %s
- Tone guidance: %s

Rules:
- Return strict JSON only, with no Markdown fences.
- Return an array of objects with locale, namespace, key, sourceValue, and value.
- Preserve every placeholder exactly, including repeated placeholders.
- Preserve HTML-like rich text tags and nesting.
- Preserve ICU argument names and plural or select structure.
- Use glossary and style-guide context from the batch when present.
- Do not invent keys that are not present in the batch.
- Do not apply translations directly.

After producing JSON proposals, call i18n.translation.validate with batchId %s. Only call i18n.translation.apply after validation succeeds and the user asks to preview or apply changes.`, batchID, localeCode, batchSummary, optionalText(domain, "not specified"), optionalText(tone, "product-neutral"), batchID)

		return textPrompt("Translate a prepared i18n batch safely.", text)
	}
}

func optionalText(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

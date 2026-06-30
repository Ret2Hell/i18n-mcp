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

func translateBatchPrompt(a *app.App) mcp.PromptHandler {
	return func(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		localeCode := promptArg(req, "locale")
		batchID := promptArg(req, "batchId")
		tone := promptArg(req, "tone")
		domain := promptArg(req, "domain")

		batchSummary := "No latest batch is currently available. Call i18n.translation.plan first, or read i18n://translation/plan/latest after a batch has been prepared."
		if batch, ok := a.Translation.LatestPlan(); ok {
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

package openai

import (
	"encoding/json"
	"strings"

	"github.com/Ret2Hell/i18n-mcp/internal/translate"
)

func buildPrompt(req translate.ProviderRequest) (string, error) {
	payload := map[string]any{
		"sourceLocale": req.SourceLocale,
		"targetLocale": req.TargetLocale,
		"items":        req.Items,
		"styleGuide":   req.StyleGuide,
		"glossary":     req.Glossary,
		"responseSchema": []string{
			"locale",
			"namespace",
			"key",
			"sourceValue",
			"value",
		},
	}
	encoded, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("Translate the following i18n items.\n")
	b.WriteString("Return JSON only: an array of proposal objects.\n")
	b.WriteString("Preserve placeholders, ICU arguments, HTML-like tags, and intentional whitespace.\n")
	b.WriteString("Do not invent keys. Do not include explanations.\n\n")
	b.Write(encoded)
	return b.String(), nil
}

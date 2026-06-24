package mcpserver

import (
	"context"

	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/Ret2Hell/i18n-mcp/internal/translate"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type TranslationPlanOutput struct {
	Batch translate.Batch `json:"batch" jsonschema:"translation batch for missing and stale keys"`
}

func translationPlanTool(a *app.App) func(context.Context, *mcp.CallToolRequest, translate.PlanInput) (*mcp.CallToolResult, TranslationPlanOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, in translate.PlanInput) (*mcp.CallToolResult, TranslationPlanOutput, error) {
		_ = req
		batch, err := a.Translation.Plan(ctx, in)
		if err != nil {
			return nil, TranslationPlanOutput{}, err
		}
		return nil, TranslationPlanOutput{Batch: batch}, nil
	}
}

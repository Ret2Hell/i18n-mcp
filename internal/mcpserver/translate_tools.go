package mcpserver

import (
	"cmp"
	"context"
	"fmt"

	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/Ret2Hell/i18n-mcp/internal/diff"
	"github.com/Ret2Hell/i18n-mcp/internal/translate"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TranslationPlanOutput is the output for the translation.plan tool.
type TranslationPlanOutput struct {
	Batch translate.Batch `json:"batch" jsonschema:"translation batch for missing and stale keys"`
}

// TranslationGenerateInput is the input for the translation.generate tool.
type TranslationGenerateInput struct {
	Mode           string   `json:"mode,omitempty"`
	Statuses       []string `json:"statuses,omitzero"`
	Locales        []string `json:"locales,omitzero"`
	Namespaces     []string `json:"namespaces,omitzero"`
	Keys           []string `json:"keys,omitzero"`
	IncludeContext bool     `json:"includeContext,omitempty"`
	MaxItems       int      `json:"maxItems,omitempty"`
	MaxTokens      int64    `json:"maxTokens,omitempty"`
}

// TranslationGenerateOutput is the output for the translation.generate tool.
type TranslationGenerateOutput struct {
	Mode      string                       `json:"mode"`
	Plan      translate.PlanOutput         `json:"plan"`
	Proposals []translate.Proposal         `json:"proposals"`
	Rejected  []translate.RejectedProposal `json:"rejected,omitzero"`
	Warnings  []string                     `json:"warnings,omitzero"`
	Model     string                       `json:"model,omitempty"`
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

func translationGenerateTool(a *app.App) func(context.Context, *mcp.CallToolRequest, TranslationGenerateInput) (*mcp.CallToolResult, TranslationGenerateOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, in TranslationGenerateInput) (*mcp.CallToolResult, TranslationGenerateOutput, error) {
		out, err := translationGenerateHandler(ctx, req, a, in)
		if err != nil {
			return nil, TranslationGenerateOutput{}, err
		}
		return nil, *out, nil
	}
}

func translationGenerateHandler(ctx context.Context, req *mcp.CallToolRequest, a *app.App, in TranslationGenerateInput) (*TranslationGenerateOutput, error) {
	mode, err := resolveTranslationMode(ctx, a, in.Mode)
	if err != nil {
		return nil, err
	}
	statuses := make([]diff.KeyStatus, 0, len(in.Statuses))
	for _, status := range in.Statuses {
		statuses = append(statuses, diff.KeyStatus(status))
	}
	plan, err := a.Translation.Plan(ctx, translate.PlanInput{
		Statuses:       statuses,
		Locales:        in.Locales,
		Namespaces:     in.Namespaces,
		Keys:           in.Keys,
		IncludeContext: in.IncludeContext,
		MaxItems:       in.MaxItems,
	})
	if err != nil {
		return nil, err
	}
	if mode == "provider" {
		cfg, err := a.Config.Resolve(ctx)
		if err != nil {
			return nil, err
		}
		generated, err := a.Translation.GenerateWithProvider(ctx, translate.ProviderGenerateInput{
			ProviderName: cfg.Translation.Provider,
			Plan:         &plan,
			StyleGuide:   plan.StyleGuide,
		})
		if err != nil {
			return nil, err
		}
		return new(TranslationGenerateOutput{
			Mode:      "provider",
			Plan:      plan,
			Proposals: generated.Proposals,
			Rejected:  generated.Rejected,
			Warnings:  generated.Warnings,
		}), nil
	}
	if mode != "sampling" {
		return nil, fmt.Errorf("translation.generate mode %q is not implemented by Epic M", mode)
	}
	if a.Sampling == nil {
		return nil, fmt.Errorf("translation.generate sampling service is not configured")
	}
	sampler := a.Sampling
	if in.MaxTokens > 0 {
		sampler = new(*sampler)
		sampler.MaxTokens = in.MaxTokens
	}
	generated, err := sampler.Generate(ctx, req.Session, translate.SamplingRequest{Plan: &plan})
	if err != nil {
		return nil, err
	}
	return new(TranslationGenerateOutput{
		Mode:      "sampling",
		Plan:      plan,
		Proposals: generated.Proposals,
		Rejected:  generated.Rejected,
		Warnings:  generated.Warnings,
		Model:     generated.Model,
	}), nil
}

func resolveTranslationMode(ctx context.Context, a *app.App, explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	cfg, err := a.Config.Resolve(ctx)
	if err != nil {
		return "", err
	}
	return cmp.Or(cfg.Translation.Mode, "agent"), nil
}

func translationValidateTool(a *app.App) func(context.Context, *mcp.CallToolRequest, translate.ValidationInput) (*mcp.CallToolResult, translate.ValidationOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, in translate.ValidationInput) (*mcp.CallToolResult, translate.ValidationOutput, error) {
		_ = req
		out, err := a.Translation.Validate(ctx, in)
		if err != nil {
			return nil, translate.ValidationOutput{}, err
		}
		if len(out.Rejected) > 0 {
			return &mcp.CallToolResult{IsError: true}, out, nil
		}
		return nil, out, nil
	}
}

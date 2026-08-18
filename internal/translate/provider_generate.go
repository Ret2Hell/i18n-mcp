package translate

import (
	"context"
	"fmt"
	"strings"
)

// ProviderGenerateInput describes a provider-backed proposal generation request.
type ProviderGenerateInput struct {
	ProviderName string
	Plan         *Batch
	StyleGuide   string
	Glossary     string
}

// ProviderGenerateOutput contains provider proposals after normal validation.
type ProviderGenerateOutput struct {
	Provider  string                `json:"provider"`
	Proposals []ProposedTranslation `json:"proposals"`
	Rejected  []RejectedTranslation `json:"rejected,omitzero"`
	Usage     ProviderUsage         `json:"usage,omitzero"`
	Warnings  []string              `json:"warnings,omitzero"`
	Metadata  map[string]any        `json:"metadata,omitzero"`
}

// GenerateWithProvider asks a configured provider for proposals and validates them
// through the same path used by agent-submitted proposals. It does not apply
// accepted proposals to locale files or state.
func (s *Service) GenerateWithProvider(ctx context.Context, in ProviderGenerateInput) (*ProviderGenerateOutput, error) {
	if in.Plan == nil {
		return nil, fmt.Errorf("provider generation requires a translation plan")
	}
	if s.Providers == nil {
		return nil, fmt.Errorf("translation provider registry is not configured")
	}
	provider, err := s.Providers.Get(in.ProviderName)
	if err != nil {
		return nil, err
	}
	cfg, err := s.config.Resolve(ctx)
	if err != nil {
		return nil, err
	}

	combined := new(ProviderResponse)
	for _, req := range providerRequestsFromPlan(cfg.SourceLocale, in.Plan, in.StyleGuide, in.Glossary) {
		resp, err := provider.Generate(ctx, req)
		if err != nil {
			return nil, err
		}
		if resp == nil {
			return nil, fmt.Errorf("translation provider %q returned nil response for locale %q", provider.Name(), req.TargetLocale)
		}
		combined.Proposals = append(combined.Proposals, resp.Proposals...)
		combined.Warnings = append(combined.Warnings, resp.Warnings...)
		combined.Usage.InputTokens += resp.Usage.InputTokens
		combined.Usage.OutputTokens += resp.Usage.OutputTokens
		combined.Usage.TotalTokens += resp.Usage.TotalTokens
		if combined.Metadata == nil {
			combined.Metadata = make(map[string]any, len(resp.Metadata))
		}
		for key, value := range resp.Metadata {
			if _, exists := combined.Metadata[key]; !exists {
				combined.Metadata[key] = value
			}
		}
	}

	validated, err := s.Validate(ctx, ValidationInput{BatchID: in.Plan.BatchID, Translations: combined.Proposals})
	if err != nil {
		return nil, err
	}
	acceptedProposals := make([]ProposedTranslation, 0, len(validated.Accepted))
	for _, accepted := range validated.Accepted {
		acceptedProposals = append(acceptedProposals, accepted.ProposedTranslation)
	}
	return &ProviderGenerateOutput{
		Provider:  provider.Name(),
		Proposals: acceptedProposals,
		Rejected:  validated.Rejected,
		Usage:     combined.Usage,
		Warnings:  combined.Warnings,
		Metadata:  safeProviderMetadata(combined.Metadata),
	}, nil
}

func providerRequestsFromPlan(sourceLocale string, plan *Batch, styleGuide string, glossary string) []ProviderRequest {
	if plan == nil {
		return nil
	}
	itemsByLocale := make(map[string][]ProviderItem, len(plan.TargetLocales))
	localeOrder := make([]string, 0, len(plan.TargetLocales))
	seen := make(map[string]bool, len(plan.TargetLocales))
	for _, locale := range plan.TargetLocales {
		if locale != "" && !seen[locale] {
			seen[locale] = true
			localeOrder = append(localeOrder, locale)
		}
	}
	for _, item := range ProviderItemsFromPlan(plan) {
		if item.Locale == "" {
			continue
		}
		if !seen[item.Locale] {
			seen[item.Locale] = true
			localeOrder = append(localeOrder, item.Locale)
		}
		itemsByLocale[item.Locale] = append(itemsByLocale[item.Locale], item)
	}

	requests := make([]ProviderRequest, 0, len(localeOrder))
	for _, locale := range localeOrder {
		items := itemsByLocale[locale]
		if len(items) == 0 {
			continue
		}
		requests = append(requests, ProviderRequest{
			SourceLocale: sourceLocale,
			TargetLocale: locale,
			Items:        items,
			StyleGuide:   styleGuide,
			Glossary:     glossary,
		})
	}
	return requests
}

func safeProviderMetadata(in map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range in {
		switch strings.ToLower(k) {
		case "apikey", "authorization", "token", "secret":
			continue
		default:
			out[k] = v
		}
	}
	return out
}

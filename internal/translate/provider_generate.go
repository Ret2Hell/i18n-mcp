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
	resp, err := provider.Generate(ctx, ProviderRequest{
		SourceLocale: cfg.SourceLocale,
		TargetLocale: firstTargetLocale(in.Plan),
		Items:        ProviderItemsFromPlan(in.Plan),
		StyleGuide:   in.StyleGuide,
		Glossary:     in.Glossary,
	})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("translation provider %q returned nil response", provider.Name())
	}
	validated, err := s.Validate(ctx, ValidationInput{BatchID: in.Plan.BatchID, Translations: resp.Proposals})
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
		Usage:     resp.Usage,
		Warnings:  resp.Warnings,
		Metadata:  safeProviderMetadata(resp.Metadata),
	}, nil
}

func firstTargetLocale(plan *Batch) string {
	if plan == nil || len(plan.TargetLocales) == 0 {
		return ""
	}
	return plan.TargetLocales[0]
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

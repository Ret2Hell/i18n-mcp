package translate

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Proposal is a translation proposal returned by MCP sampling.
type Proposal = ProposedTranslation

// RejectedProposal is a sampled proposal rejected by validation.
type RejectedProposal = RejectedTranslation

// PlanOutput is the translation plan consumed by sampling.
type PlanOutput = Batch

// ProposalValidator validates sampled proposals through the normal translation validation path.
type ProposalValidator interface {
	Validate(context.Context, ValidationInput) (ValidationOutput, error)
}

// SamplingService generates translation proposals with MCP client sampling.
type SamplingService struct {
	Validator ProposalValidator
	MaxTokens int64
}

// SamplingRequest describes a sampling proposal generation request.
type SamplingRequest struct {
	Plan       *PlanOutput
	StyleGuide string
	Glossary   string
}

// SamplingResponse contains validated sampled translation proposals.
type SamplingResponse struct {
	Proposals []Proposal         `json:"proposals"`
	Rejected  []RejectedProposal `json:"rejected,omitzero"`
	Model     string             `json:"model,omitempty"`
	Warnings  []string           `json:"warnings,omitzero"`
}

// Generate samples translation proposals from the MCP client and validates them.
func (s *SamplingService) Generate(ctx context.Context, session *mcp.ServerSession, req SamplingRequest) (*SamplingResponse, error) {
	if session == nil || session.InitializeParams() == nil {
		return nil, fmt.Errorf("sampling requires an initialized MCP session")
	}
	caps := session.InitializeParams().Capabilities
	if caps == nil || caps.Sampling == nil {
		return nil, fmt.Errorf("client does not support MCP sampling")
	}
	if s.Validator == nil {
		return nil, fmt.Errorf("sampling requires a proposal validator")
	}

	prompt, err := buildSamplingPrompt(req)
	if err != nil {
		return nil, err
	}
	maxTokens := s.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 4096
	}
	res, err := session.CreateMessage(ctx, &mcp.CreateMessageParams{
		SystemPrompt: "You generate JSON translation proposals for i18n locale files. Return only valid JSON.",
		MaxTokens:    maxTokens,
		Temperature:  0.2,
		Messages: []*mcp.SamplingMessage{
			{Role: "user", Content: &mcp.TextContent{Text: prompt}},
		},
	})
	if err != nil {
		return nil, err
	}
	text, ok := res.Content.(*mcp.TextContent)
	if !ok {
		return nil, fmt.Errorf("sampling returned non-text content")
	}
	var proposals []Proposal
	if err := json.Unmarshal([]byte(text.Text), &proposals); err != nil {
		return nil, fmt.Errorf("decoding sampled proposals: %w", err)
	}
	validated, err := s.Validator.Validate(ctx, ValidationInput{BatchID: req.Plan.BatchID, Translations: proposals})
	if err != nil {
		return nil, err
	}
	acceptedProposals := make([]Proposal, 0, len(validated.Accepted))
	for _, accepted := range validated.Accepted {
		acceptedProposals = append(acceptedProposals, accepted.ProposedTranslation)
	}
	return new(SamplingResponse{Proposals: acceptedProposals, Rejected: validated.Rejected, Model: res.Model}), nil
}

func buildSamplingPrompt(req SamplingRequest) (string, error) {
	if req.Plan == nil {
		return "", fmt.Errorf("sampling requires a translation plan")
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Source locale: %s\n", req.Plan.SourceLocale)
	fmt.Fprintf(&b, "Target locales: %s\n", strings.Join(req.Plan.TargetLocales, ", "))
	fmt.Fprintf(&b, "Batch ID: %s\n\n", req.Plan.BatchID)
	b.WriteString("Translate these items:\n")
	for _, item := range req.Plan.Items {
		fmt.Fprintf(&b, "- Target locale: %s\n", item.Locale)
		fmt.Fprintf(&b, "  Namespace: %s\n", item.Namespace)
		fmt.Fprintf(&b, "  Key: %s\n", item.Key)
		fmt.Fprintf(&b, "  Current source value: %q\n", item.SourceValue)
		if item.OldValue != "" {
			fmt.Fprintf(&b, "  Previous target value: %q\n", item.OldValue)
		}
		if len(item.Placeholders) > 0 {
			fmt.Fprintf(&b, "  Placeholders to preserve: %s\n", strings.Join(item.Placeholders, ", "))
		}
		if len(item.Tags) > 0 {
			fmt.Fprintf(&b, "  HTML-like tags to preserve: %s\n", strings.Join(item.Tags, ", "))
		}
	}
	if strings.TrimSpace(req.StyleGuide) != "" {
		fmt.Fprintf(&b, "\nStyle guide:\n%s\n", req.StyleGuide)
	}
	if strings.TrimSpace(req.Glossary) != "" {
		fmt.Fprintf(&b, "\nGlossary:\n%s\n", req.Glossary)
	}
	b.WriteString("\nReturn JSON only. The response must be an array of objects with locale, namespace, key, sourceValue, and value fields.\n")
	b.WriteString("Preserve every placeholder, HTML-like tag, ICU argument, and newline from the source value unless locale grammar requires reordering.\n")
	b.WriteString("Do not add explanations outside JSON.\n")
	b.WriteString("Exact JSON response schema: [{\"locale\":\"<target locale>\",\"namespace\":\"<namespace>\",\"key\":\"<key>\",\"sourceValue\":\"<exact source value>\",\"value\":\"<translated value>\"}]\n")
	return b.String(), nil
}

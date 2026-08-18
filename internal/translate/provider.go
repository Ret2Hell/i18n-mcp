package translate

import (
	"cmp"
	"context"
	"fmt"
	"time"
)

// Provider generates translation proposals for planned translation batches.
type Provider interface {
	Name() string
	Generate(ctx context.Context, req ProviderRequest) (*ProviderResponse, error)
}

// ProviderRequest is the provider-neutral input for generating translations.
type ProviderRequest struct {
	SourceLocale string         `json:"sourceLocale"`
	TargetLocale string         `json:"targetLocale"`
	Items        []ProviderItem `json:"items"`
	StyleGuide   string         `json:"styleGuide,omitempty"`
	Glossary     string         `json:"glossary,omitempty"`
	MaxItems     int            `json:"maxItems,omitempty"`
}

// ProviderItem describes a single translation unit for a provider.
type ProviderItem struct {
	ID           string `json:"id"`
	Locale       string `json:"locale"`
	Namespace    string `json:"namespace"`
	Key          string `json:"key"`
	SourceValue  string `json:"sourceValue"`
	CurrentValue string `json:"currentValue,omitempty"`
	Status       string `json:"status"`
	SourceHash   string `json:"sourceHash"`
	TargetHash   string `json:"targetHash,omitempty"`
	Description  string `json:"description,omitempty"`
}

// ProviderResponse is the provider-neutral output from translation generation.
type ProviderResponse struct {
	Proposals []ProposedTranslation `json:"proposals"`
	Usage     ProviderUsage         `json:"usage,omitzero"`
	Warnings  []string              `json:"warnings,omitzero"`
	Metadata  map[string]any        `json:"metadata,omitzero"`
	StartedAt time.Time             `json:"startedAt,omitzero"`
	EndedAt   time.Time             `json:"endedAt,omitzero"`
}

// ProviderUsage reports token usage returned by a provider.
type ProviderUsage struct {
	InputTokens  int `json:"inputTokens,omitempty"`
	OutputTokens int `json:"outputTokens,omitempty"`
	TotalTokens  int `json:"totalTokens,omitempty"`
}

// ProviderRegistry stores configured translation providers by name.
type ProviderRegistry struct {
	providers   map[string]Provider
	unavailable map[string]error
}

// NewProviderRegistry builds a registry from the provided providers.
func NewProviderRegistry(providers ...Provider) *ProviderRegistry {
	r := &ProviderRegistry{providers: map[string]Provider{}, unavailable: map[string]error{}}
	for _, p := range providers {
		if p != nil {
			r.providers[p.Name()] = p
		}
	}
	return r
}

// MarkUnavailable records why a provider could not be configured at startup.
func (r *ProviderRegistry) MarkUnavailable(name string, err error) {
	if r == nil || name == "" || err == nil {
		return
	}
	if r.unavailable == nil {
		r.unavailable = map[string]error{}
	}
	r.unavailable[name] = err
}

// Get returns a configured provider by name, defaulting to openai-compatible.
func (r *ProviderRegistry) Get(name string) (Provider, error) {
	name = cmp.Or(name, "openai-compatible")
	p, ok := r.providers[name]
	if ok {
		return p, nil
	}
	if err := r.unavailable[name]; err != nil {
		return nil, fmt.Errorf("translation provider %q is unavailable: %w", name, err)
	}
	return nil, fmt.Errorf("translation provider %q is not configured", name)
}

// ProviderItemsFromPlan maps a translation plan to provider request items.
func ProviderItemsFromPlan(plan *Batch) []ProviderItem {
	if plan == nil {
		return nil
	}
	items := make([]ProviderItem, 0, len(plan.Items))
	for _, item := range plan.Items {
		items = append(items, ProviderItem{
			ID:           item.ID,
			Locale:       item.Locale,
			Namespace:    item.Namespace,
			Key:          item.Key,
			SourceValue:  item.SourceValue,
			CurrentValue: item.OldValue,
			Status:       string(item.Status),
			SourceHash:   item.SourceHash,
			TargetHash:   item.TargetHash,
		})
	}
	return items
}

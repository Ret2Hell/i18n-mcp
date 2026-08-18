package translate

import (
	"context"
	"testing"

	"github.com/Ret2Hell/i18n-mcp/internal/diff"
	"github.com/stretchr/testify/require"
)

type testProvider struct {
	name     string
	generate func(context.Context, ProviderRequest) (*ProviderResponse, error)
}

func (p testProvider) Name() string { return p.name }

func (p testProvider) Generate(ctx context.Context, req ProviderRequest) (*ProviderResponse, error) {
	if p.generate != nil {
		return p.generate(ctx, req)
	}
	return &ProviderResponse{}, nil
}

func TestProviderRegistryReturnsProviderByName(t *testing.T) {
	provider := &testProvider{name: "mock"}
	registry := NewProviderRegistry(provider)

	got, err := registry.Get("mock")

	require.NoError(t, err)
	require.Same(t, provider, got)
}

func TestProviderRegistryDefaultsToOpenAICompatible(t *testing.T) {
	provider := &testProvider{name: "openai-compatible"}
	registry := NewProviderRegistry(provider)

	got, err := registry.Get("")

	require.NoError(t, err)
	require.Same(t, provider, got)
}

func TestProviderRegistryUnknownProviderError(t *testing.T) {
	registry := NewProviderRegistry(testProvider{name: "configured"})

	got, err := registry.Get("missing")

	require.Nil(t, got)
	require.EqualError(t, err, `translation provider "missing" is not configured`)
}

func TestProviderItemsFromPlanPreservesPlanFields(t *testing.T) {
	plan := &Batch{Items: []Item{
		{
			ID:          "fr:common:welcome",
			Locale:      "fr",
			Namespace:   "common",
			Key:         "welcome",
			SourceValue: "Welcome {name}",
			OldValue:    "Bienvenue",
			Status:      diff.Stale,
			SourceHash:  "source-hash",
			TargetHash:  "target-hash",
		},
	}}

	items := ProviderItemsFromPlan(plan)

	require.Equal(t, []ProviderItem{
		{
			ID:           "fr:common:welcome",
			Locale:       "fr",
			Namespace:    "common",
			Key:          "welcome",
			SourceValue:  "Welcome {name}",
			CurrentValue: "Bienvenue",
			Status:       "stale",
			SourceHash:   "source-hash",
			TargetHash:   "target-hash",
		},
	}, items)
}

func TestProviderItemsFromPlanNilPlan(t *testing.T) {
	require.Nil(t, ProviderItemsFromPlan(nil))
}

func TestProviderRequestsFromPlanPartitionsItemsByLocale(t *testing.T) {
	plan := &Batch{
		TargetLocales: []string{"de", "fr", "de"},
		Items: []Item{
			{ID: "fr:common:title", Locale: "fr", Namespace: "common", Key: "title", SourceValue: "Title"},
			{ID: "de:common:title", Locale: "de", Namespace: "common", Key: "title", SourceValue: "Title"},
			{ID: "es:common:title", Locale: "es", Namespace: "common", Key: "title", SourceValue: "Title"},
		},
	}

	requests := providerRequestsFromPlan("en", plan, "Be concise.", "title = heading")

	require.Len(t, requests, 3)
	require.Equal(t, []string{"de", "fr", "es"}, []string{requests[0].TargetLocale, requests[1].TargetLocale, requests[2].TargetLocale})
	require.Equal(t, "en", requests[0].SourceLocale)
	require.Equal(t, "Be concise.", requests[0].StyleGuide)
	require.Equal(t, "title = heading", requests[0].Glossary)
	require.Equal(t, "de:common:title", requests[0].Items[0].ID)
	require.Equal(t, "fr:common:title", requests[1].Items[0].ID)
	require.Equal(t, "es:common:title", requests[2].Items[0].ID)
}

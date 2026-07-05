package translate_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Ret2Hell/i18n-mcp/internal/state"
	"github.com/Ret2Hell/i18n-mcp/internal/translate"
	"github.com/stretchr/testify/require"
)

type generateProvider struct {
	name     string
	generate func(context.Context, translate.ProviderRequest) (*translate.ProviderResponse, error)
}

func (p generateProvider) Name() string { return p.name }

func (p generateProvider) Generate(ctx context.Context, req translate.ProviderRequest) (*translate.ProviderResponse, error) {
	return p.generate(ctx, req)
}

func TestGenerateWithProviderValidOutputAccepted(t *testing.T) {
	ctx := t.Context()
	a := newTranslationFixtureApp(t)
	plan, err := a.Translation.Plan(ctx, translate.PlanInput{})
	require.NoError(t, err)
	a.Translation.Providers = translate.NewProviderRegistry(generateProvider{name: "mock", generate: func(_ context.Context, req translate.ProviderRequest) (*translate.ProviderResponse, error) {
		require.Equal(t, "en", req.SourceLocale)
		require.Equal(t, "fr", req.TargetLocale)
		return &translate.ProviderResponse{Proposals: []translate.Proposal{{
			Locale:      "fr",
			Namespace:   "auth",
			Key:         "login.title",
			SourceValue: "Log in",
			Value:       "Connexion",
		}}}, nil
	}})

	out, err := a.Translation.GenerateWithProvider(ctx, translate.ProviderGenerateInput{ProviderName: "mock", Plan: &plan})

	require.NoError(t, err)
	require.Equal(t, "mock", out.Provider)
	require.Len(t, out.Proposals, 1)
	require.Empty(t, out.Rejected)
}

func TestGenerateWithProviderRejectsPlaceholderMismatch(t *testing.T) {
	ctx := t.Context()
	a := newTranslationFixtureApp(t)
	plan, err := a.Translation.Plan(ctx, translate.PlanInput{})
	require.NoError(t, err)
	a.Translation.Providers = providerReturning([]translate.Proposal{{
		Locale:      "fr",
		Namespace:   "auth",
		Key:         "login.subtitle",
		SourceValue: "Welcome {name}",
		Value:       "Bienvenue",
	}})

	out, err := a.Translation.GenerateWithProvider(ctx, translate.ProviderGenerateInput{ProviderName: "mock", Plan: &plan})

	require.NoError(t, err)
	require.Empty(t, out.Proposals)
	require.Len(t, out.Rejected, 1)
	requireIssueCode(t, out.Rejected[0].Issues, "placeholder_missing")
}

func TestGenerateWithProviderRejectsUnknownKeyOrLocale(t *testing.T) {
	ctx := t.Context()
	a := newTranslationFixtureApp(t)
	plan, err := a.Translation.Plan(ctx, translate.PlanInput{})
	require.NoError(t, err)
	a.Translation.Providers = providerReturning([]translate.Proposal{{
		Locale:      "de",
		Namespace:   "auth",
		Key:         "missing",
		SourceValue: "Missing",
		Value:       "Fehlt",
	}})

	out, err := a.Translation.GenerateWithProvider(ctx, translate.ProviderGenerateInput{ProviderName: "mock", Plan: &plan})

	require.NoError(t, err)
	require.Empty(t, out.Proposals)
	require.Len(t, out.Rejected, 1)
	requireIssueCode(t, out.Rejected[0].Issues, "proposal_not_in_batch")
}

func TestGenerateWithProviderRedactsMetadata(t *testing.T) {
	ctx := t.Context()
	a := newTranslationFixtureApp(t)
	plan, err := a.Translation.Plan(ctx, translate.PlanInput{})
	require.NoError(t, err)
	a.Translation.Providers = translate.NewProviderRegistry(generateProvider{name: "mock", generate: func(context.Context, translate.ProviderRequest) (*translate.ProviderResponse, error) {
		return &translate.ProviderResponse{Metadata: map[string]any{"apiKey": "hidden", "authorization": "hidden", "token": "hidden", "secret": "hidden", "model": "kept"}}, nil
	}})

	out, err := a.Translation.GenerateWithProvider(ctx, translate.ProviderGenerateInput{ProviderName: "mock", Plan: &plan})

	require.NoError(t, err)
	require.Equal(t, map[string]any{"model": "kept"}, out.Metadata)
}

func TestGenerateWithProviderErrorDoesNotWriteFilesOrState(t *testing.T) {
	ctx := t.Context()
	a := newTranslationFixtureApp(t)
	plan, err := a.Translation.Plan(ctx, translate.PlanInput{})
	require.NoError(t, err)
	beforeLocale := readFixtureFile(t, a.ProjectRoot, "messages/fr/auth.json")
	beforeState := readFixtureFile(t, a.ProjectRoot, state.DefaultStatePath)
	a.Translation.Providers = translate.NewProviderRegistry(generateProvider{name: "mock", generate: func(context.Context, translate.ProviderRequest) (*translate.ProviderResponse, error) {
		return nil, errors.New("provider unavailable")
	}})

	out, err := a.Translation.GenerateWithProvider(ctx, translate.ProviderGenerateInput{ProviderName: "mock", Plan: &plan})

	require.Nil(t, out)
	require.EqualError(t, err, "provider unavailable")
	require.Equal(t, beforeLocale, readFixtureFile(t, a.ProjectRoot, "messages/fr/auth.json"))
	require.Equal(t, beforeState, readFixtureFile(t, a.ProjectRoot, state.DefaultStatePath))
}

func providerReturning(proposals []translate.Proposal) *translate.ProviderRegistry {
	return translate.NewProviderRegistry(generateProvider{name: "mock", generate: func(context.Context, translate.ProviderRequest) (*translate.ProviderResponse, error) {
		return &translate.ProviderResponse{Proposals: proposals}, nil
	}})
}

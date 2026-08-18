package translate_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Ret2Hell/i18n-mcp/internal/state"
	"github.com/Ret2Hell/i18n-mcp/internal/translate"
	"github.com/stretchr/testify/require"
)

type fakeProvider struct {
	name      string
	response  *translate.ProviderResponse
	err       error
	lastInput translate.ProviderRequest
}

func (p *fakeProvider) Name() string { return p.name }

func (p *fakeProvider) Generate(ctx context.Context, req translate.ProviderRequest) (*translate.ProviderResponse, error) {
	p.lastInput = req
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if p.err != nil {
		return nil, p.err
	}
	return p.response, nil
}

func TestProviderGenerateSuccess(t *testing.T) {
	fake := &fakeProvider{name: "fake", response: &translate.ProviderResponse{
		Proposals: []translate.ProposedTranslation{{Locale: "fr", Namespace: "auth", Key: "login.title", SourceValue: "Log in", Value: "Connexion"}},
	}}
	service, plan := newProviderTestService(t, fake)

	out, err := service.GenerateWithProvider(t.Context(), translate.ProviderGenerateInput{ProviderName: "fake", Plan: plan})

	require.NoError(t, err)
	require.Len(t, out.Proposals, 1)
	require.Empty(t, out.Rejected)
	require.Equal(t, "en", fake.lastInput.SourceLocale)
	require.Equal(t, "fr", fake.lastInput.TargetLocale)
	require.NotEmpty(t, fake.lastInput.Items)
}

func TestProviderGenerateValidationFailure(t *testing.T) {
	fake := &fakeProvider{name: "fake", response: &translate.ProviderResponse{
		Proposals: []translate.ProposedTranslation{{Locale: "fr", Namespace: "auth", Key: "login.subtitle", SourceValue: "Welcome {name}", Value: "Bienvenue"}},
	}}
	service, plan := newProviderTestService(t, fake)

	out, err := service.GenerateWithProvider(t.Context(), translate.ProviderGenerateInput{ProviderName: "fake", Plan: plan})

	require.NoError(t, err)
	require.Empty(t, out.Proposals)
	require.NotEmpty(t, out.Rejected)
}

func TestProviderGenerateError(t *testing.T) {
	fake := &fakeProvider{name: "fake", err: errors.New("provider unavailable")}
	app := newTranslationFixtureApp(t)
	plan, err := app.Translation.Plan(t.Context(), translate.PlanInput{})
	require.NoError(t, err)
	app.Translation.Providers = translate.NewProviderRegistry(fake)
	beforeLocale := readFixtureFile(t, app.ProjectRoot, "messages/fr/auth.json")
	beforeState := readFixtureFile(t, app.ProjectRoot, state.DefaultStatePath)

	_, err = app.Translation.GenerateWithProvider(t.Context(), translate.ProviderGenerateInput{ProviderName: "fake", Plan: new(plan)})

	require.ErrorContains(t, err, "provider unavailable")
	require.Equal(t, beforeLocale, readFixtureFile(t, app.ProjectRoot, "messages/fr/auth.json"))
	require.Equal(t, beforeState, readFixtureFile(t, app.ProjectRoot, state.DefaultStatePath))
}

func TestProviderGenerateCancellation(t *testing.T) {
	fake := &fakeProvider{name: "fake", response: &translate.ProviderResponse{}}
	service, plan := newProviderTestService(t, fake)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := service.GenerateWithProvider(ctx, translate.ProviderGenerateInput{ProviderName: "fake", Plan: plan})

	require.ErrorIs(t, err, context.Canceled)
}

func TestProviderGenerateRedactsSensitiveMetadata(t *testing.T) {
	fake := &fakeProvider{name: "fake", response: &translate.ProviderResponse{Metadata: map[string]any{
		"apiKey": "sk-test-secret", "authorization": "Bearer hidden", "token": "hidden", "secret": "hidden", "model": "kept",
	}}}
	service, plan := newProviderTestService(t, fake)

	out, err := service.GenerateWithProvider(t.Context(), translate.ProviderGenerateInput{ProviderName: "fake", Plan: plan})

	require.NoError(t, err)
	require.Equal(t, map[string]any{"model": "kept"}, out.Metadata)
}

func newProviderTestService(t *testing.T, provider translate.Provider) (*translate.Service, *translate.Batch) {
	t.Helper()
	a := newTranslationFixtureApp(t)
	plan, err := a.Translation.Plan(t.Context(), translate.PlanInput{})
	require.NoError(t, err)
	a.Translation.Providers = translate.NewProviderRegistry(provider)
	return a.Translation, new(plan)
}

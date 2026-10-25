package translate_test

import (
	"context"
	"github.com/Ret2Hell/i18n-mcp/internal/translate"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestTranslationDetectsDriftAtWriteBoundary(t *testing.T) {
	a := newTranslationFixtureApp(t)
	batch, err := a.Translation.Plan(t.Context(), translate.PlanInput{Keys: []string{"login.title"}})
	require.NoError(t, err)
	external := "{\"login\":{\"subtitle\":\"External change\"}}"
	out, err := a.Translation.Apply(t.Context(), translate.ApplyInput{Apply: true, BatchID: batch.BatchID, Translations: []translate.ProposedTranslation{{Locale: "fr", Namespace: "auth", Key: "login.title", SourceValue: "Log in", Value: "Connexion"}}, BeforeWrite: func(context.Context, translate.ApplyOutput) error {
		writeFixtureFile(t, a.ProjectRoot, "messages/fr/auth.json", external)
		return nil
	}})
	require.ErrorContains(t, err, "changed since preview")
	require.Zero(t, out.Applied)
	require.Equal(t, external, readFixtureFile(t, a.ProjectRoot, "messages/fr/auth.json"))
}

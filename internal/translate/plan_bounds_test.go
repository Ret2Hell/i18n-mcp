package translate_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/Ret2Hell/i18n-mcp/internal/pagination"
	"github.com/Ret2Hell/i18n-mcp/internal/translate"
	"github.com/stretchr/testify/require"
)

func TestTranslationPlanIsBoundedAndPaginates(t *testing.T) {
	a := newTranslationFixtureApp(t)
	values := map[string]string{}
	for i := range 230 {
		values[fmt.Sprintf("key%03d", i)] = "Translate me"
	}
	raw, err := json.Marshal(values)
	require.NoError(t, err)
	writeFixtureFile(t, a.ProjectRoot, "messages/en/auth.json", string(raw))
	writeFixtureFile(t, a.ProjectRoot, "messages/fr/auth.json", `{}`)
	in := translate.PlanInput{}
	seen := map[string]bool{}
	for {
		batch, err := a.Translation.Plan(t.Context(), in)
		require.NoError(t, err)
		require.LessOrEqual(t, len(batch.Items), pagination.DefaultLimit)
		require.Equal(t, 230, batch.Page.TotalMatched)
		for _, item := range batch.Items {
			require.False(t, seen[item.ID])
			seen[item.ID] = true
		}
		_, err = a.Translation.ResolveBatch(t.Context(), batch.BatchID)
		require.NoError(t, err)
		if !batch.Page.Truncated {
			break
		}
		in.Cursor = batch.Page.NextCursor
	}
	require.Len(t, seen, 230)
	_, err = a.Translation.Plan(t.Context(), translate.PlanInput{MaxItems: 201})
	require.Error(t, err)
}

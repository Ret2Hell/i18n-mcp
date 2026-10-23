package pagination

import (
	"context"
	"strings"
	"testing"

	"github.com/Ret2Hell/i18n-mcp/internal/security"
	"github.com/stretchr/testify/require"
)

func TestPagesAreCompleteAndSnapshotBound(t *testing.T) {
	signer, err := security.NewRandomRequestStateSigner()
	require.NoError(t, err)
	ctx := t.Context()
	input := Input{Limit: 2}
	var all []string
	for {
		b, err := New[string](ctx, signer, "test", "project", "filter", input)
		require.NoError(t, err)
		for _, item := range []string{"a", "b", "c", "d", "e"} {
			require.NoError(t, b.Add(item))
		}
		meta, err := b.Finish()
		require.NoError(t, err)
		require.Equal(t, 5, meta.TotalMatched)
		all = append(all, b.Items...)
		if !meta.Truncated {
			break
		}
		input.Cursor = meta.NextCursor
	}
	require.Equal(t, []string{"a", "b", "c", "d", "e"}, all)
	b, err := New[string](ctx, signer, "test", "project", "filter", Input{Limit: 2})
	require.NoError(t, err)
	require.NoError(t, b.Add("a"))
	require.NoError(t, b.Add("b"))
	require.NoError(t, b.Add("c"))
	meta, err := b.Finish()
	require.NoError(t, err)
	_, err = New[string](ctx, signer, "test", "other project", "filter", Input{Limit: 2, Cursor: meta.NextCursor})
	require.Error(t, err)
	_, err = New[string](ctx, signer, "test", "project", "other filter", Input{Limit: 2, Cursor: meta.NextCursor})
	require.Error(t, err)
	_, err = New[string](ctx, signer, "test", "project", "filter", Input{Limit: 3, Cursor: meta.NextCursor})
	require.Error(t, err)
	b, err = New[string](ctx, signer, "test", "project", "filter", Input{Limit: 2, Cursor: meta.NextCursor})
	require.NoError(t, err)
	require.NoError(t, b.Add("changed"))
	_, err = b.Finish()
	require.ErrorContains(t, err, "stale")
}

func TestByteBudgetAndCancellation(t *testing.T) {
	signer, err := security.NewRandomRequestStateSigner()
	require.NoError(t, err)
	b, err := New[string](t.Context(), signer, "test", "project", nil, Input{})
	require.NoError(t, err)
	item := strings.Repeat("a", ItemBudget/2)
	require.NoError(t, b.Add(item))
	require.NoError(t, b.Add(item))
	meta, err := b.Finish()
	require.NoError(t, err)
	require.Equal(t, 1, meta.Returned)
	require.True(t, meta.Truncated)
	b, err = New[string](t.Context(), signer, "test", "project", nil, Input{})
	require.NoError(t, err)
	require.Error(t, b.Add(strings.Repeat("a", ItemBudget)))
	_, err = New[string](t.Context(), signer, "test", "project", nil, Input{Limit: MaxLimit + 1})
	require.Error(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	b, err = New[string](ctx, signer, "test", "project", nil, Input{})
	require.NoError(t, err)
	require.ErrorIs(t, b.Add("a"), context.Canceled)
}

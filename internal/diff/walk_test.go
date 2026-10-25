package diff

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/Ret2Hell/i18n-mcp/internal/config"
	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
	"github.com/Ret2Hell/i18n-mcp/internal/locale"
	"github.com/Ret2Hell/i18n-mcp/internal/state"
	"github.com/Ret2Hell/i18n-mcp/internal/validate"
	"github.com/stretchr/testify/require"
)

func walkFixture(t *testing.T) (*Service, locale.Inventory, string) {
	t.Helper()
	root := makeDiffFixture(t)
	guard, err := fsutil.NewGuard(root)
	require.NoError(t, err)
	locales := locale.NewService(guard, config.NewService(guard, ""))
	service := NewService(locales, state.NewService(state.NewStore(guard), locales), validate.NewService())
	inv, err := locales.Inventory(t.Context())
	require.NoError(t, err)
	return service, inv, root
}

func TestWalkInventoryFiltersAndSummary(t *testing.T) {
	s, inv, _ := walkFixture(t)
	for _, tc := range []struct {
		name   string
		filter Filter
		total  int
	}{
		{"all", Filter{}, 11},
		{"locale", Filter{Locales: []string{"fr"}}, 6},
		{"namespace", Filter{Namespaces: []string{"absent"}}, 0},
		{"keys and prefix", Filter{Keys: []string{"missing", "current"}, KeyPrefix: "miss"}, 2},
		{"status", Filter{Statuses: []KeyStatus{Invalid, Stale}}, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var items []KeyDiff
			got, err := s.WalkInventory(t.Context(), inv, tc.filter, func(item KeyDiff) error { items = append(items, item); return nil })
			require.NoError(t, err)
			require.Len(t, items, tc.total)
			require.Equal(t, Summarize(items), got.Summary)
			require.Empty(t, got.Items)
			for i := 1; i < len(items); i++ {
				require.True(t, lessKeyDiff(items[i-1], items[i]))
			}
			counts, err := s.WalkInventory(t.Context(), inv, tc.filter, nil)
			require.NoError(t, err)
			require.Equal(t, got, counts)
		})
	}
}

func TestWalkInventoryPropagatesErrors(t *testing.T) {
	s, inv, root := walkFixture(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := s.WalkInventory(ctx, inv, Filter{}, nil)
	require.ErrorIs(t, err, context.Canceled)
	_, err = s.WalkInventory(t.Context(), inv, Filter{Statuses: []KeyStatus{"bogus"}}, nil)
	require.ErrorContains(t, err, `unknown diff status "bogus"`)
	want := errors.New("visitor failed")
	calls := 0
	got, err := s.WalkInventory(t.Context(), inv, Filter{}, func(KeyDiff) error { calls++; return want })
	require.ErrorIs(t, err, want)
	require.Equal(t, 1, calls)
	require.Equal(t, Report{}, got)
	ctx, cancel = context.WithCancel(t.Context())
	defer cancel()
	calls = 0
	_, err = s.WalkInventory(ctx, inv, Filter{}, func(KeyDiff) error { calls++; cancel(); return nil })
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, 1, calls)
	writeDiffFile(t, root, state.DefaultStatePath, `{`)
	_, err = s.WalkInventory(t.Context(), inv, Filter{}, nil)
	require.Error(t, err)
}

func TestWalkInventoryCountsInvalidSourcesOnceAndBoundsSamples(t *testing.T) {
	s, _, _ := walkFixture(t)
	inv := locale.Inventory{SourceLocale: "en", TargetLocales: []string{"de", "fr"}, Validation: config.ValidationConfig{MessageFormat: "icu"}}
	for i := 0; i < 25; i++ {
		inv.Units = append(inv.Units, locale.Unit{Locale: "en", Namespace: "common", Key: fmt.Sprintf("key%02d", i), Value: `{n, plural, one {broken}}`})
	}
	var items []KeyDiff
	got, err := s.WalkInventory(t.Context(), inv, Filter{}, func(item KeyDiff) error { items = append(items, item); return nil })
	require.NoError(t, err)
	require.Equal(t, 25, got.SourceIssueCount)
	require.Len(t, got.SourceIssues, 20)
	require.Equal(t, 50, got.Summary.SourceInvalid)
	for _, item := range items {
		require.Equal(t, SourceInvalid, item.Status)
		require.NotEmpty(t, item.SourceValidation)
		require.Equal(t, "en", item.SourceValidation[0].Locale)
		require.Equal(t, item.Key, item.SourceValidation[0].Key)
	}
	// Status filtering must not hide source-level diagnostics.
	got, err = s.WalkInventory(t.Context(), inv, Filter{Statuses: []KeyStatus{Missing}}, nil)
	require.NoError(t, err)
	require.Zero(t, got.Summary.Total)
	require.Equal(t, 25, got.SourceIssueCount)
}

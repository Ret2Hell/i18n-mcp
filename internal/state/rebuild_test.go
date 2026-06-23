package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Ret2Hell/i18n-mcp/internal/config"
	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
	"github.com/Ret2Hell/i18n-mcp/internal/locale"
	"github.com/stretchr/testify/require"
)

func TestServiceRebuildDryRunAndApplySemantics(t *testing.T) {
	fixedNow := time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)
	store, service := newRebuildTestService(t, fixedNow)

	existing := NewFile("en", fixedNow.Add(-time.Hour))
	existing.Entries[EntryKey("fr", "common", "reviewed")] = Entry{
		Locale:             "fr",
		Namespace:          "common",
		Key:                "reviewed",
		SourceHash:         SourceHash("Old"),
		TranslatedFromHash: SourceHash("Old"),
		TargetHash:         TargetHash("Ancien"),
		Status:             StatusCurrent,
		Reviewed:           true,
		UpdatedAt:          fixedNow.Add(-time.Hour),
	}
	require.NoError(t, store.Save(t.Context(), existing))

	dryRun, err := service.Rebuild(t.Context(), RebuildOptions{})
	require.NoError(t, err)
	require.True(t, dryRun.DryRun)
	require.False(t, dryRun.Applied)
	require.Equal(t, DefaultStatePath, dryRun.StatePath)
	require.Equal(t, "en", dryRun.SourceLocale)
	require.Equal(t, 2, dryRun.Entries)
	require.Equal(t, 1, dryRun.Created)
	require.Equal(t, 1, dryRun.Updated)
	require.Equal(t, 3, dryRun.Skipped)
	require.Len(t, dryRun.PreviewState.Entries, 2)
	require.True(t, dryRun.PreviewState.Entries[EntryKey("fr", "common", "reviewed")].Reviewed)
	require.Equal(t, fixedNow, dryRun.PreviewState.Entries[EntryKey("fr", "common", "hello")].UpdatedAt)

	loadedAfterDryRun, err := store.Load(t.Context())
	require.NoError(t, err)
	require.Len(t, loadedAfterDryRun.Entries, 1)
	require.Equal(t, TargetHash("Ancien"), loadedAfterDryRun.Entries[EntryKey("fr", "common", "reviewed")].TargetHash)

	applied, err := service.Rebuild(t.Context(), RebuildOptions{Apply: true})
	require.NoError(t, err)
	require.False(t, applied.DryRun)
	require.True(t, applied.Applied)
	require.Equal(t, 2, applied.Entries)
	require.Equal(t, 1, applied.Created)
	require.Equal(t, 1, applied.Updated)
	require.Equal(t, 3, applied.Skipped)
	require.NotContains(t, applied.Assumptions, "placeholder, rich-text tag, and ICU validation will be added in Epic D")
	require.Contains(t, applied.Assumptions, "rebuild records current state only; placeholder, rich-text tag, and ICU checks are reported by validation tools")

	loadedAfterApply, err := store.Load(t.Context())
	require.NoError(t, err)
	require.Len(t, loadedAfterApply.Entries, 2)
	hello := loadedAfterApply.Entries[EntryKey("fr", "common", "hello")]
	require.Equal(t, SourceHash("Hello"), hello.SourceHash)
	require.Equal(t, SourceHash("Hello"), hello.TranslatedFromHash)
	require.Equal(t, TargetHash("Bonjour"), hello.TargetHash)
	require.Equal(t, StatusCurrent, hello.Status)
	require.Equal(t, "state.rebuild", hello.UpdatedBy)

	reviewed := loadedAfterApply.Entries[EntryKey("fr", "common", "reviewed")]
	require.True(t, reviewed.Reviewed)
	require.Equal(t, SourceHash("Reviewed"), reviewed.SourceHash)
	require.Equal(t, TargetHash("Revu"), reviewed.TargetHash)
}

func TestServiceRebuildReturnsInventoryError(t *testing.T) {
	root := t.TempDir()
	writeRebuildFile(t, root, ".i18n-mcp.json", `{
  "sourceLocale": "en",
  "targetLocales": ["fr"],
  "localeFiles": ["../outside/{locale}.json"],
  "defaultNamespace": "common"
}`)
	guard, err := fsutil.NewGuard(root)
	require.NoError(t, err)
	store := NewStore(guard)
	service := NewService(store, locale.NewService(guard, config.NewService(guard, "")))

	_, err = service.Rebuild(t.Context(), RebuildOptions{})
	require.Error(t, err)
}

func newRebuildTestService(t *testing.T, now time.Time) (*Store, *Service) {
	t.Helper()
	root := t.TempDir()
	writeRebuildFile(t, root, ".i18n-mcp.json", `{
  "sourceLocale": "en",
  "targetLocales": ["fr"],
  "localeFiles": ["messages/{locale}.json"],
  "defaultNamespace": "common"
}`)
	writeRebuildFile(t, root, "messages/en.json", `{
  "empty": "Empty",
  "hello": "Hello",
  "onlySource": "Only source",
  "reviewed": "Reviewed"
}`)
	writeRebuildFile(t, root, "messages/fr.json", `{
  "empty": "   ",
  "hello": "Bonjour",
  "orphan": "Orphelin",
  "reviewed": "Revu"
}`)
	writeRebuildFile(t, root, "messages/es.json", `{
  "hello": "Hola"
}`)

	guard, err := fsutil.NewGuard(root)
	require.NoError(t, err)
	store := NewStore(guard)
	service := NewService(store, locale.NewService(guard, config.NewService(guard, "")))
	service.now = func() time.Time { return now }
	return store, service
}

func writeRebuildFile(t *testing.T, root string, relPath string, contents string) {
	t.Helper()
	path := filepath.Join(root, relPath)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))
}

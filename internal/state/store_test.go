package state

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
	"github.com/stretchr/testify/require"
)

func TestLoadMissingStateReturnsEmptyFile(t *testing.T) {
	store := newTestStore(t)

	file, err := store.Load(context.Background())
	require.NoError(t, err)
	require.Equal(t, CurrentVersion, file.Version)
	require.Empty(t, file.Entries)
}

func TestLoadValidState(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".i18n-mcp"), 0o700))
	stateFile := NewFile("en", time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC))
	stateFile.Entries[EntryKey("fr", "common", "hello")] = Entry{
		Locale:             "fr",
		Namespace:          "common",
		Key:                "hello",
		SourceHash:         SourceHash("Hello"),
		TranslatedFromHash: SourceHash("Hello"),
		Status:             StatusCurrent,
		UpdatedAt:          time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
	}
	data, err := json.Marshal(stateFile)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(root, DefaultStatePath), data, 0o600))

	guard, err := fsutil.NewGuard(root)
	require.NoError(t, err)
	loaded, err := NewStore(guard).Load(context.Background())
	require.NoError(t, err)
	require.Equal(t, "en", loaded.SourceLocale)
	require.Len(t, loaded.Entries, 1)
}

func TestLoadCorruptStateReturnsError(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".i18n-mcp"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, DefaultStatePath), []byte(`{"version":`), 0o600))

	guard, err := fsutil.NewGuard(root)
	require.NoError(t, err)
	_, err = NewStore(guard).Load(context.Background())
	require.Error(t, err)
}

func TestSaveWritesStateAtomically(t *testing.T) {
	store := newTestStore(t)
	file := NewFile("en", time.Now())
	file.Entries[EntryKey("fr", "common", "hello")] = Entry{
		Locale:             "fr",
		Namespace:          "common",
		Key:                "hello",
		SourceHash:         SourceHash("Hello"),
		TranslatedFromHash: SourceHash("Hello"),
		TargetHash:         TargetHash("Bonjour"),
		Status:             StatusCurrent,
		UpdatedAt:          time.Now().UTC(),
	}

	require.NoError(t, store.Save(context.Background(), file))

	loaded, err := store.Load(context.Background())
	require.NoError(t, err)
	require.Len(t, loaded.Entries, 1)

	absStateDir := filepath.Join(store.guard.Root(), ".i18n-mcp")
	entries, err := os.ReadDir(absStateDir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, "state.json", entries[0].Name())
}

func TestSaveRejectsPathOutsideRoot(t *testing.T) {
	root := t.TempDir()
	guard, err := fsutil.NewGuard(root)
	require.NoError(t, err)
	store := NewStoreAt(guard, "../state.json")

	err = store.Save(context.Background(), NewFile("en", time.Now()))
	require.Error(t, err)
}

func TestSaveRejectsStateFileSymlink(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".i18n-mcp"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, "real-state.json"), []byte(`{}`), 0o600))
	require.NoError(t, os.Symlink(filepath.Join(root, "real-state.json"), filepath.Join(root, DefaultStatePath)))

	guard, err := fsutil.NewGuard(root)
	require.NoError(t, err)
	err = NewStore(guard).Save(context.Background(), NewFile("en", time.Now()))
	require.Error(t, err)
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	guard, err := fsutil.NewGuard(t.TempDir())
	require.NoError(t, err)
	return NewStore(guard)
}

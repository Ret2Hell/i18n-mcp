package watch

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Ret2Hell/i18n-mcp/internal/config"
	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
	"github.com/Ret2Hell/i18n-mcp/internal/locale"
	"github.com/fsnotify/fsnotify"
	"github.com/stretchr/testify/require"
)

type recordingNotifier struct{ updates chan []string }

func (n recordingNotifier) Updated(_ context.Context, uris ...string) { n.updates <- uris }

func TestLocaleWatcherDebouncesAndIgnoresUnmappedEvents(t *testing.T) {
	n := recordingNotifier{make(chan []string, 10)}
	w, err := NewLocaleWatcher(n)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- w.Run(ctx, func(path string) []string {
			if path == "ignored" {
				return nil
			}
			return []string{path}
		})
	}()
	for _, ev := range []fsnotify.Event{
		{Name: "a", Op: fsnotify.Chmod},
		{Name: "a", Op: fsnotify.Write},
		{Name: "a", Op: fsnotify.Create},
		{Name: "b", Op: fsnotify.Rename},
		{Name: "ignored", Op: fsnotify.Remove},
	} {
		w.watcher.Events <- ev
	}
	require.Eventually(t, func() bool { return len(n.updates) == 2 }, 5*time.Second, time.Millisecond)
	got := [][]string{<-n.updates, <-n.updates}
	require.ElementsMatch(t, [][]string{{"a"}, {"b"}}, got)
	cancel()
	require.ErrorIs(t, <-done, context.Canceled)
}

func TestLocaleWatcherStopsPendingNotifications(t *testing.T) {
	n := recordingNotifier{make(chan []string, 1)}
	w, err := NewLocaleWatcher(n)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- w.Run(ctx, func(string) []string { return []string{"changed"} }) }()
	w.watcher.Events <- fsnotify.Event{Name: "a", Op: fsnotify.Write}
	cancel()
	require.ErrorIs(t, <-done, context.Canceled)
	time.Sleep(2 * w.debounce)
	require.Empty(t, n.updates)
	require.Empty(t, w.watcher.WatchList())
}

func TestLocaleWatcherReturnsBackendErrorAndHandlesClose(t *testing.T) {
	for _, closed := range []bool{false, true} {
		t.Run(map[bool]string{false: "error", true: "closed"}[closed], func(t *testing.T) {
			w, err := NewLocaleWatcher(recordingNotifier{make(chan []string, 1)})
			require.NoError(t, err)
			done := make(chan error, 1)
			go func() { done <- w.Run(t.Context(), func(string) []string { return nil }) }()
			if closed {
				require.NoError(t, w.Close())
				select {
				case err := <-done:
					require.NoError(t, err)
				case <-time.After(5 * time.Second):
					t.Fatal("Run did not stop on close")
				}
			} else {
				want := errors.New("watch backend failed")
				w.watcher.Errors <- want
				select {
				case err := <-done:
					require.ErrorIs(t, err, want)
				case <-time.After(5 * time.Second):
					t.Fatal("Run did not stop on error")
				}
			}
		})
	}
}

func TestLocaleWatcherAddDir(t *testing.T) {
	w, err := NewLocaleWatcher(recordingNotifier{make(chan []string, 1)})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, w.Close()) })
	dir := t.TempDir()
	require.NoError(t, w.AddDir(dir))
	require.Equal(t, []string{dir}, w.watcher.WatchList())
	require.Error(t, w.AddDir(filepath.Join(dir, "missing")))
}

func TestNewLocaleMapper(t *testing.T) {
	guard, err := fsutil.NewGuard(t.TempDir())
	require.NoError(t, err)
	for _, tc := range []struct {
		name, pattern, namespace, path string
		want                           []string
	}{
		{"namespaced", "locales/{locale}/{namespace}.json", "", "locales/fr/admin.json", []string{"i18n://locales/fr/admin", "i18n://analysis/diff"}},
		{"default", "messages/{locale}.json", "site", "messages/fr.json", []string{"i18n://locales/fr/site", "i18n://analysis/diff"}},
		{"fallback", "messages/{locale}.json", "", "messages/fr.json", []string{"i18n://locales/fr/common", "i18n://analysis/diff"}},
		{"escaped", "locales/{locale}/{namespace}.json", "", "locales/fr/admin space.json", []string{"i18n://locales/fr/admin%20space", "i18n://analysis/diff"}},
		{"unmatched", "messages/{locale}.json", "", "README.md", nil},
		{"outside", "messages/{locale}.json", "", "../messages/fr.json", nil},
		{"invalid pattern", "[/{locale}.json", "", "messages/fr.json", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mapper := NewLocaleMapper(guard, config.Resolved{LocaleFiles: []string{tc.pattern}, DefaultNamespace: tc.namespace})
			require.Equal(t, tc.want, mapper(filepath.Join(guard.Root(), tc.path)))
		})
	}
}

func TestGuardedLocaleDirsDeduplicatesAndPropagatesErrors(t *testing.T) {
	root := t.TempDir()
	guard, err := fsutil.NewGuard(root)
	require.NoError(t, err)
	cfg := config.NewService(guard, "")
	service := locale.NewService(guard, cfg)
	require.NoError(t, os.WriteFile(filepath.Join(root, ".i18n-mcp.json"), []byte(`{`), 0600))
	_, err = GuardedLocaleDirs(t.Context(), guard, service)
	require.Error(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(root, ".i18n-mcp.json"), []byte(`{"sourceLocale":"en","localeFiles":["messages/{locale}.json"]}`), 0600))
	require.NoError(t, os.Mkdir(filepath.Join(root, "messages"), 0700))
	for _, code := range []string{"en", "fr"} {
		require.NoError(t, os.WriteFile(filepath.Join(root, "messages", code+".json"), []byte(`{"hello":"Hello"}`), 0600))
	}
	dirs, err := GuardedLocaleDirs(t.Context(), guard, service)
	require.NoError(t, err)
	require.Equal(t, []string{filepath.Join(root, "messages")}, dirs)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = GuardedLocaleDirs(ctx, guard, service)
	require.ErrorIs(t, err, context.Canceled)
}

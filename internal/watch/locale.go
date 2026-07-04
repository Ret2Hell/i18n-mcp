package watch

import (
	"cmp"
	"context"
	"errors"
	"maps"
	"path/filepath"
	"slices"
	"time"

	"github.com/Ret2Hell/i18n-mcp/internal/config"
	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
	"github.com/Ret2Hell/i18n-mcp/internal/locale"
	"github.com/fsnotify/fsnotify"
)

// Notifier emits update notifications for changed resource URIs.
type Notifier interface {
	Updated(ctx context.Context, uris ...string)
}

// LocaleWatcher watches locale directories and debounces file change notifications.
type LocaleWatcher struct {
	watcher  *fsnotify.Watcher
	notifier Notifier
	debounce time.Duration
}

// NewLocaleWatcher creates a locale file watcher.
func NewLocaleWatcher(notifier Notifier) (*LocaleWatcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &LocaleWatcher{watcher: w, notifier: notifier, debounce: 250 * time.Millisecond}, nil
}

// AddDir adds a guarded locale directory to the watcher.
func (w *LocaleWatcher) AddDir(path string) error {
	return w.watcher.Add(path)
}

// Run watches for locale file changes until ctx is cancelled.
func (w *LocaleWatcher) Run(ctx context.Context, mapper func(string) []string) (err error) {
	timers := map[string]*time.Timer{}
	defer func() {
		err = errors.Join(err, w.watcher.Close())
	}()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-w.watcher.Errors:
			if err != nil {
				return err
			}
		case ev := <-w.watcher.Events:
			if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) == 0 {
				continue
			}
			if timers[ev.Name] != nil {
				timers[ev.Name].Stop()
			}
			name := ev.Name
			timers[name] = time.AfterFunc(w.debounce, func() {
				w.notifier.Updated(context.Background(), mapper(name)...)
			})
		}
	}
}

// GuardedLocaleDirs returns existing non-symlink directories that contain locale files.
func GuardedLocaleDirs(ctx context.Context, guard *fsutil.Guard, service *locale.Service) ([]string, error) {
	inv, err := service.Inventory(ctx)
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	for _, file := range inv.Files {
		dir := filepath.Dir(file.Path)
		abs, err := guard.ResolveExisting(dir)
		if err != nil {
			return nil, err
		}
		seen[abs] = struct{}{}
	}
	return slices.Sorted(maps.Keys(seen)), nil
}

// NewLocaleMapper maps changed filesystem paths to affected MCP resource URIs.
func NewLocaleMapper(guard *fsutil.Guard, cfg config.Resolved) func(string) []string {
	return func(path string) []string {
		rel, err := guard.Rel(path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		for _, pattern := range cfg.LocaleFiles {
			ref, ok, err := locale.MatchPattern(pattern, rel)
			if err != nil || !ok {
				continue
			}
			namespace := cmp.Or(ref.Namespace, cfg.DefaultNamespace, "common")
			return []string{
				"i18n://locales/" + ref.Locale + "/" + namespace,
				"i18n://analysis/diff",
			}
		}
		return nil
	}
}

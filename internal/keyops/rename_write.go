package keyops

import (
	"bytes"
	"context"
	"os"
	"slices"
	"time"

	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
	"github.com/Ret2Hell/i18n-mcp/internal/state"
)

type writeReport struct {
	Written   []string
	Unchanged []string
	Skipped   []string
}

func (s *Service) writeEdits(ctx context.Context, edits []fileEdit) (writeReport, error) {
	_ = ctx
	var report writeReport
	for _, edit := range edits {
		if bytes.Equal(edit.Before, edit.After) {
			report.Unchanged = append(report.Unchanged, edit.Path)
			continue
		}
		perm := os.FileMode(0o600)
		absPath, err := s.guard.Resolve(edit.Path)
		if err != nil {
			report.Skipped = appendRemaining(report.Skipped, edits, edit.Path)
			return report, err
		}
		if info, err := os.Stat(absPath); err == nil {
			perm = info.Mode().Perm()
		}
		if err := fsutil.AtomicWriteFile(s.guard, edit.Path, edit.After, perm); err != nil {
			report.Skipped = appendRemaining(report.Skipped, edits, edit.Path)
			return report, err
		}
		report.Written = append(report.Written, edit.Path)
	}
	return report, nil
}

func appendRemaining(out []string, edits []fileEdit, failedPath string) []string {
	idx := slices.IndexFunc(edits, func(edit fileEdit) bool {
		return edit.Path == failedPath
	})
	if idx == -1 {
		return out
	}
	for _, edit := range edits[idx:] {
		out = append(out, edit.Path)
	}
	return out
}

func (s *Service) applyStateUpdates(ctx context.Context, updates []StateUpdate) error {
	if len(updates) == 0 {
		return nil
	}
	stateFile, err := s.state.Load(ctx)
	if err != nil {
		return err
	}
	if stateFile.Entries == nil {
		stateFile.Entries = map[string]state.Entry{}
	}
	now := time.Now().UTC()
	for _, update := range updates {
		oldKey := state.EntryKey(update.Locale, update.Namespace, update.FromKey)
		entry, ok := stateFile.Entries[oldKey]
		if !ok {
			continue
		}
		delete(stateFile.Entries, oldKey)
		entry.Key = update.ToKey
		entry.UpdatedAt = now
		entry.UpdatedBy = "keys.rename"
		stateFile.Entries[state.EntryKey(update.Locale, update.Namespace, update.ToKey)] = entry
	}
	return s.state.Save(ctx, stateFile)
}

func markWritten(files []ChangedFile, written []string) []ChangedFile {
	set := make(map[string]bool, len(written))
	for _, path := range written {
		set[path] = true
	}
	for i := range files {
		files[i].Written = set[files[i].Path]
	}
	return files
}

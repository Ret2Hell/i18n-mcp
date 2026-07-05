package translate

import (
	"context"
	"os"

	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
)

// WriteReport summarizes locale file writes.
type WriteReport struct {
	Written   []string `json:"written"`
	Unchanged []string `json:"unchanged"`
	Skipped   []string `json:"skipped"`
}

// WriteEdits writes locale file edits to disk.
func (s *Service) WriteEdits(ctx context.Context, edits []FileEdit) (WriteReport, error) {
	_ = ctx
	var report WriteReport
	for _, edit := range edits {
		if string(edit.Before) == string(edit.After) {
			report.Unchanged = append(report.Unchanged, edit.Path)
			continue
		}

		perm := os.FileMode(0o600)
		resolved, err := s.guard.Resolve(edit.Path)
		if err != nil {
			report.Skipped = appendRemaining(report.Skipped, edits, edit.Path)
			return report, err
		}
		if info, err := os.Stat(resolved); err == nil {
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

func appendRemaining(out []string, edits []FileEdit, failedPath string) []string {
	started := false
	for _, edit := range edits {
		if edit.Path == failedPath {
			started = true
		}
		if started {
			out = append(out, edit.Path)
		}
	}
	return out
}

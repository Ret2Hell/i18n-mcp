package translate

import (
	"bytes"
	"context"
	"os"
	"slices"

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
	var report WriteReport
	for _, edit := range edits {
		if err := ctx.Err(); err != nil {
			return report, err
		}
		if edit.CheckBefore {
			if err := fsutil.VerifyFileVersion(s.guard, edit.Path, edit.Before, edit.BeforeMissing); err != nil {
				for _, pending := range edits {
					report.Skipped = append(report.Skipped, pending.Path)
				}
				return report, err
			}
		}
	}
	for _, edit := range edits {
		if err := ctx.Err(); err != nil {
			report.Skipped = appendRemaining(report.Skipped, edits, edit.Path)
			return report, err
		}
		if bytes.Equal(edit.Before, edit.After) {
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

		if edit.CheckBefore {
			if err := fsutil.VerifyFileVersion(s.guard, edit.Path, edit.Before, edit.BeforeMissing); err != nil {
				report.Skipped = appendRemaining(report.Skipped, edits, edit.Path)
				return report, err
			}
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
	failed := slices.IndexFunc(edits, func(edit FileEdit) bool {
		return edit.Path == failedPath
	})
	if failed == -1 {
		return out
	}
	for _, edit := range edits[failed:] {
		out = append(out, edit.Path)
	}
	return out
}

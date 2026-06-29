package keyops

import (
	"context"
	"errors"
	"slices"

	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
)

func (s *Service) Rename(ctx context.Context, in RenameInput) (RenameOutput, error) {
	dryRun := in.DryRunValue()
	plan, err := s.PlanRename(ctx, in)
	if err != nil {
		return RenameOutput{}, err
	}
	out := RenameOutput{
		DryRun:       dryRun,
		Planned:      len(plan.Edits),
		ChangedFiles: previewEdits(plan.Edits),
		StateUpdates: slices.Clone(plan.StateUpdates),
		Conflicts:    slices.Clone(plan.Conflicts),
		Warnings:     slices.Clone(plan.Warnings),
	}
	if len(out.Conflicts) > 0 {
		return out, nil
	}
	if dryRun {
		return out, nil
	}
	return out, errors.New("rename write mode is not enabled until KAN-098")
}

func previewEdits(edits []fileEdit) []ChangedFile {
	out := make([]ChangedFile, 0, len(edits))
	for _, edit := range edits {
		diffText := fsutil.UnifiedDiff(edit.Path, edit.Before, edit.After)
		out = append(out, ChangedFile{Path: edit.Path, Diff: diffText, Changed: diffText != ""})
	}
	return out
}

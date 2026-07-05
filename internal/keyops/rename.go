package keyops

import (
	"context"
	"fmt"
	"slices"

	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
	"github.com/Ret2Hell/i18n-mcp/internal/resources"
)

// Rename renames a translation key across locale files and state.
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
	if len(out.Conflicts) > 0 || dryRun {
		return out, nil
	}

	writeReport, err := s.writeEdits(ctx, plan.Edits)
	out.ChangedFiles = markWritten(out.ChangedFiles, writeReport.Written)
	s.notifyRenamed(ctx, plan.Edits, writeReport.Written)
	if err != nil {
		out.Warnings = append(out.Warnings, fmt.Sprintf("rename write failed after writing %d file(s): %v", len(writeReport.Written), err))
		return out, err
	}
	if err := s.applyStateUpdates(ctx, plan.StateUpdates); err != nil {
		out.Warnings = append(out.Warnings, "locale files were written but state update failed: "+err.Error())
		return out, err
	}
	out.Renamed = len(plan.Edits)
	return out, nil
}

func (s *Service) notifyRenamed(ctx context.Context, edits []fileEdit, written []string) {
	if s.Notifier == nil || len(written) == 0 {
		return
	}
	writtenSet := make(map[string]struct{}, len(written))
	for _, path := range written {
		writtenSet[path] = struct{}{}
	}
	uris := make([]string, 0, len(written)+3)
	for _, edit := range edits {
		if _, ok := writtenSet[edit.Path]; ok {
			uris = append(uris, resources.LocaleURI(edit.Locale, edit.Namespace))
		}
	}
	uris = append(uris, resources.DiffURI, resources.UsageURI, resources.DeadKeysURI)
	s.Notifier.Updated(ctx, uris...)
}

func previewEdits(edits []fileEdit) []ChangedFile {
	out := make([]ChangedFile, 0, len(edits))
	for _, edit := range edits {
		diffText := fsutil.UnifiedDiff(edit.Path, edit.Before, edit.After)
		out = append(out, ChangedFile{Path: edit.Path, Diff: diffText, Changed: diffText != ""})
	}
	return out
}

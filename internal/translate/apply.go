package translate

import (
	"context"
	"fmt"
	"time"

	"github.com/Ret2Hell/i18n-mcp/internal/resources"
	"github.com/Ret2Hell/i18n-mcp/internal/state"
)

func (s *Service) Apply(ctx context.Context, in ApplyInput) (ApplyOutput, error) {
	dryRun := in.DryRunValue()
	validation, err := s.Validate(ctx, ValidationInput{
		BatchID:          in.BatchID,
		Translations:     in.Translations,
		AllowSourceDrift: in.AllowSourceDrift,
	})
	if err != nil {
		return ApplyOutput{}, err
	}

	out := ApplyOutput{DryRun: dryRun, Rejected: validation.Rejected}
	if len(validation.Rejected) > 0 {
		return out, nil
	}

	edits, err := s.BuildEdits(ctx, validation.Accepted)
	if err != nil {
		return ApplyOutput{}, err
	}
	out.ChangedFiles = PreviewEdits(edits)
	if dryRun {
		return out, nil
	}

	writeReport, err := s.WriteEdits(ctx, edits)
	out.ChangedFiles = markWritten(out.ChangedFiles, writeReport.Written)
	s.notifyApplied(ctx, validation.Accepted, writeReport.Written)
	if err != nil {
		out.Warnings = append(out.Warnings, fmt.Sprintf("locale write failed after writing %d file(s): %v", len(writeReport.Written), err))
		return out, err
	}

	updates, err := s.updateStateAfterApply(ctx, validation.Accepted)
	if err != nil {
		out.Warnings = append(out.Warnings, "locale files were written but state update failed: "+err.Error())
		return out, err
	}
	out.Applied = len(validation.Accepted)
	out.StateUpdates = updates
	return out, nil
}

func (s *Service) notifyApplied(ctx context.Context, accepted []ValidatedTranslation, written []string) {
	if s.Notifier == nil || len(written) == 0 {
		return
	}
	writtenSet := make(map[string]struct{}, len(written))
	for _, path := range written {
		writtenSet[path] = struct{}{}
	}
	uris := make([]string, 0, len(accepted)+1)
	for _, tr := range accepted {
		if _, ok := writtenSet[tr.TargetFilePath]; ok {
			uris = append(uris, resources.LocaleURI(tr.Locale, tr.Namespace))
		}
	}
	uris = append(uris, resources.DiffURI)
	s.Notifier.Updated(ctx, uris...)
}

func markWritten(files []ChangedFile, written []string) []ChangedFile {
	set := map[string]bool{}
	for _, path := range written {
		set[path] = true
	}
	for i := range files {
		files[i].Written = set[files[i].Path]
	}
	return files
}

func (s *Service) updateStateAfterApply(ctx context.Context, accepted []ValidatedTranslation) (int, error) {
	if len(accepted) == 0 {
		return 0, nil
	}
	stateFile, err := s.state.Load(ctx)
	if err != nil {
		return 0, err
	}
	if stateFile.Entries == nil {
		stateFile.Entries = map[string]state.Entry{}
	}
	now := time.Now().UTC()
	for _, tr := range accepted {
		key := state.EntryKey(tr.Locale, tr.Namespace, tr.Key)
		stateFile.Entries[key] = state.Entry{
			Locale:             tr.Locale,
			Namespace:          tr.Namespace,
			Key:                tr.Key,
			SourceHash:         tr.SourceHash,
			TranslatedFromHash: tr.SourceHash,
			TargetHash:         tr.TargetHash,
			Status:             state.StatusCurrent,
			Reviewed:           true,
			UpdatedAt:          now,
			UpdatedBy:          "translation.apply",
		}
	}
	if err := s.state.Save(ctx, stateFile); err != nil {
		return 0, err
	}
	return len(accepted), nil
}

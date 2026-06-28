package deadkey

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
	"github.com/Ret2Hell/i18n-mcp/internal/jsonedit"
	"github.com/Ret2Hell/i18n-mcp/internal/locale"
)

type pruneEdit struct {
	Path   string
	Before []byte
	After  []byte
}

func (s *Service) Prune(ctx context.Context, in PruneInput) (PruneOutput, error) {
	dryRun := in.DryRunValue()
	edits, rejected, err := s.BuildPruneEdits(ctx, in)
	if err != nil {
		return PruneOutput{}, err
	}
	out := PruneOutput{DryRun: dryRun, Rejected: rejected}
	if len(rejected) > 0 {
		return out, nil
	}
	out.ChangedFiles = previewPruneEdits(edits)
	if dryRun {
		return out, nil
	}
	return out, errors.New("prune write mode is not enabled until KAN-093")
}

func (s *Service) BuildPruneEdits(ctx context.Context, in PruneInput) ([]pruneEdit, []PruneReject, error) {
	if len(in.Keys) == 0 {
		return nil, []PruneReject{{Reason: "at least one exact namespace and key is required"}}, nil
	}
	report, err := s.Report(ctx, ReportInput{RefreshUsage: false, IncludeUsed: true})
	if err != nil {
		return nil, nil, err
	}
	allowed, rejected := validatePruneSelection(report, in)
	if len(rejected) > 0 {
		return nil, rejected, nil
	}
	inv, err := s.locales.Inventory(ctx)
	if err != nil {
		return nil, nil, err
	}
	return s.buildPruneEditsForInventory(ctx, inv, allowed)
}

func validatePruneSelection(report Report, in PruneInput) (map[string]PruneKey, []PruneReject) {
	items := map[string]Item{}
	for _, item := range report.Items {
		items[pruneIdentity(item.Namespace, item.Key)] = item
	}
	allowed := map[string]PruneKey{}
	var rejected []PruneReject
	for _, key := range in.Keys {
		identity := pruneIdentity(key.Namespace, key.Key)
		if key.Namespace == "" || key.Key == "" {
			rejected = append(rejected, PruneReject{Namespace: key.Namespace, Key: key.Key, Reason: "namespace and key are required"})
			continue
		}
		item, ok := items[identity]
		if !ok {
			rejected = append(rejected, PruneReject{Namespace: key.Namespace, Key: key.Key, Reason: "key was not found in source locale"})
			continue
		}
		if item.Status != StatusProbablyUnused && !in.AllowUnsafe {
			rejected = append(rejected, PruneReject{Namespace: key.Namespace, Key: key.Key, Status: item.Status, Reason: "key is not classified as probably_unused; set allowUnsafe only after human review"})
			continue
		}
		allowed[identity] = key
	}
	return allowed, rejected
}

func pruneIdentity(namespace string, key string) string {
	return namespace + "\x00" + key
}

func (s *Service) buildPruneEditsForInventory(ctx context.Context, inv locale.Inventory, keys map[string]PruneKey) ([]pruneEdit, []PruneReject, error) {
	_ = ctx
	keysByNamespace := map[string][]PruneKey{}
	for _, key := range keys {
		keysByNamespace[key.Namespace] = append(keysByNamespace[key.Namespace], key)
	}
	for namespace := range keysByNamespace {
		slices.SortFunc(keysByNamespace[namespace], func(a, b PruneKey) int {
			return cmp.Compare(a.Key, b.Key)
		})
	}

	var edits []pruneEdit
	for _, file := range inv.Files {
		selected := keysByNamespace[file.Namespace]
		if len(selected) == 0 {
			continue
		}
		before, doc, err := s.readPruneDocument(ctx, file.Path)
		if err != nil {
			return nil, nil, err
		}
		for _, key := range selected {
			if _, err := doc.Delete(strings.Split(key.Key, ".")); err != nil {
				return nil, nil, fmt.Errorf("delete %s in %s: %w", key.Key, file.Path, err)
			}
		}
		after, err := doc.Render()
		if err != nil {
			return nil, nil, err
		}
		if !bytes.Equal(before, after) {
			edits = append(edits, pruneEdit{Path: file.Path, Before: before, After: after})
		}
	}
	slices.SortFunc(edits, func(a, b pruneEdit) int { return cmp.Compare(a.Path, b.Path) })
	return edits, nil, nil
}

func (s *Service) readPruneDocument(ctx context.Context, relPath string) ([]byte, *jsonedit.Document, error) {
	cfg, err := s.config.Resolve(ctx)
	if err != nil {
		return nil, nil, err
	}
	absPath, err := s.guard.ResolveExisting(relPath)
	if err != nil {
		return nil, nil, err
	}
	data, err := os.ReadFile(absPath)
	if errors.Is(err, os.ErrNotExist) {
		doc := jsonedit.NewObject(cfg.Format.Indent, cfg.Format.SortKeys)
		return []byte("{}\n"), doc, nil
	}
	if err != nil {
		return nil, nil, err
	}
	doc, err := jsonedit.Parse(data, cfg.Format.Indent, cfg.Format.SortKeys)
	if err != nil {
		return nil, nil, fmt.Errorf("parse %s: %w", relPath, err)
	}
	doc.Format.SortKeys = cfg.Format.SortKeys
	doc.Format.TrailingNewline = cfg.Format.TrailingNewline
	return data, doc, nil
}

func previewPruneEdits(edits []pruneEdit) []ChangedFile {
	out := make([]ChangedFile, 0, len(edits))
	for _, edit := range edits {
		diffText := fsutil.UnifiedDiff(edit.Path, edit.Before, edit.After)
		out = append(out, ChangedFile{Path: edit.Path, Diff: diffText, Changed: diffText != ""})
	}
	return out
}

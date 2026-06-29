package keyops

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/Ret2Hell/i18n-mcp/internal/config"
	"github.com/Ret2Hell/i18n-mcp/internal/jsonedit"
	"github.com/Ret2Hell/i18n-mcp/internal/locale"
	"github.com/Ret2Hell/i18n-mcp/internal/state"
)

func (s *Service) PlanRename(ctx context.Context, in RenameInput) (Plan, error) {
	plan := Plan{Input: in}
	if err := validateRenameInput(in); err != nil {
		plan.Conflicts = append(plan.Conflicts, Conflict{Reason: err.Error()})
		return plan, nil
	}
	cfg, err := s.config.Resolve(ctx)
	if err != nil {
		return Plan{}, err
	}
	inv, err := s.locales.Inventory(ctx)
	if err != nil {
		return Plan{}, err
	}
	stateFile, err := s.state.Load(ctx)
	if err != nil {
		return Plan{}, err
	}

	locales := selectedLocales(inv, in.Locales)
	units := unitIndex(inv)
	files := fileIndex(inv)
	policy := in.NormalizedConflictPolicy()

	for _, localeCode := range locales {
		file, ok := files[fileIdentity(localeCode, in.Namespace)]
		if !ok {
			plan.Warnings = append(plan.Warnings, fmt.Sprintf("no locale file for %s/%s", localeCode, in.Namespace))
			continue
		}
		fromUnit, hasFrom := units[unitIdentity(localeCode, in.Namespace, in.FromKey)]
		if !hasFrom {
			if localeCode == inv.SourceLocale {
				plan.Conflicts = append(plan.Conflicts, Conflict{Locale: localeCode, Namespace: in.Namespace, FromKey: in.FromKey, FilePath: file.Path, Reason: "source key does not exist"})
			} else {
				plan.Warnings = append(plan.Warnings, fmt.Sprintf("target key %s/%s/%s does not exist; skipping", localeCode, in.Namespace, in.FromKey))
			}
			continue
		}
		_, hasTo := units[unitIdentity(localeCode, in.Namespace, in.ToKey)]
		if hasTo && policy == ConflictReject {
			plan.Conflicts = append(plan.Conflicts, Conflict{Locale: localeCode, Namespace: in.Namespace, FromKey: in.FromKey, ToKey: in.ToKey, FilePath: file.Path, Reason: "destination key already exists"})
			continue
		}
		if hasTo && policy == ConflictSkip {
			plan.Warnings = append(plan.Warnings, fmt.Sprintf("destination key %s/%s/%s exists; skipping", localeCode, in.Namespace, in.ToKey))
			continue
		}
		edit, changed, err := s.planFileRename(cfg, file.Path, fromUnit.Key, in.ToKey, policy == ConflictOverwrite)
		if err != nil {
			if errors.Is(err, jsonedit.ErrPathExists) {
				plan.Conflicts = append(plan.Conflicts, Conflict{Locale: localeCode, Namespace: in.Namespace, FromKey: in.FromKey, ToKey: in.ToKey, FilePath: file.Path, Reason: "destination path already exists"})
				continue
			}
			if errors.Is(err, jsonedit.ErrAncestorDescendantPath) {
				plan.Conflicts = append(plan.Conflicts, Conflict{Locale: localeCode, Namespace: in.Namespace, FromKey: in.FromKey, ToKey: in.ToKey, FilePath: file.Path, Reason: err.Error()})
				continue
			}
			return Plan{}, err
		}
		if changed {
			edit.Locale = localeCode
			plan.Edits = append(plan.Edits, edit)
		}
	}

	plan.StateUpdates = planStateUpdates(stateFile, locales, in, policy, &plan)
	sortPlan(&plan)
	return plan, nil
}

func (s *Service) planFileRename(cfg config.Resolved, relPath string, fromKey string, toKey string, overwrite bool) (fileEdit, bool, error) {
	before, doc, err := s.readDocument(relPath, cfg)
	if err != nil {
		return fileEdit{}, false, err
	}
	renamed, err := doc.RenameString(strings.Split(fromKey, "."), strings.Split(toKey, "."), overwrite)
	if err != nil || !renamed {
		return fileEdit{}, false, err
	}
	after, err := doc.Render()
	if err != nil {
		return fileEdit{}, false, err
	}
	changed := !bytes.Equal(before, after)
	return fileEdit{Path: relPath, Before: before, After: after, Changed: changed}, changed, nil
}

func (s *Service) readDocument(relPath string, cfg config.Resolved) ([]byte, *jsonedit.Document, error) {
	absPath, err := s.guard.ResolveExisting(relPath)
	if err != nil {
		return nil, nil, err
	}
	data, err := os.ReadFile(absPath)
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

func validateRenameInput(in RenameInput) error {
	if strings.TrimSpace(in.Namespace) == "" {
		return fmt.Errorf("namespace is required")
	}
	if strings.TrimSpace(in.FromKey) == "" {
		return fmt.Errorf("fromKey is required")
	}
	if strings.TrimSpace(in.ToKey) == "" {
		return fmt.Errorf("toKey is required")
	}
	if in.FromKey == in.ToKey {
		return fmt.Errorf("fromKey and toKey must be different")
	}
	return nil
}

func selectedLocales(inv locale.Inventory, requested []string) []string {
	if len(requested) > 0 {
		out := slices.Clone(requested)
		slices.Sort(out)
		return out
	}
	out := slices.Clone(inv.Locales)
	slices.Sort(out)
	return out
}

func unitIndex(inv locale.Inventory) map[string]locale.Unit {
	out := make(map[string]locale.Unit, len(inv.Units))
	for _, unit := range inv.Units {
		out[unitIdentity(unit.Locale, unit.Namespace, unit.Key)] = unit
	}
	return out
}

func fileIndex(inv locale.Inventory) map[string]locale.FileSummary {
	out := make(map[string]locale.FileSummary, len(inv.Files))
	for _, file := range inv.Files {
		out[fileIdentity(file.Locale, file.Namespace)] = file
	}
	return out
}

func planStateUpdates(stateFile state.File, locales []string, in RenameInput, policy ConflictPolicy, plan *Plan) []StateUpdate {
	updates := make([]StateUpdate, 0, len(locales))
	for _, localeCode := range locales {
		oldKey := state.EntryKey(localeCode, in.Namespace, in.FromKey)
		if _, ok := stateFile.Entries[oldKey]; !ok {
			continue
		}
		newKey := state.EntryKey(localeCode, in.Namespace, in.ToKey)
		if _, exists := stateFile.Entries[newKey]; exists && policy == ConflictReject {
			plan.Conflicts = append(plan.Conflicts, Conflict{Locale: localeCode, Namespace: in.Namespace, FromKey: in.FromKey, ToKey: in.ToKey, Reason: "destination state entry already exists"})
			continue
		}
		if _, exists := stateFile.Entries[newKey]; exists && policy == ConflictSkip {
			plan.Warnings = append(plan.Warnings, fmt.Sprintf("destination state entry %s/%s/%s exists; skipping", localeCode, in.Namespace, in.ToKey))
			continue
		}
		updates = append(updates, StateUpdate{Locale: localeCode, Namespace: in.Namespace, FromKey: in.FromKey, ToKey: in.ToKey})
	}
	return updates
}

func unitIdentity(localeCode string, namespace string, key string) string {
	return localeCode + "\x00" + namespace + "\x00" + key
}

func fileIdentity(localeCode string, namespace string) string {
	return localeCode + "\x00" + namespace
}

func sortPlan(plan *Plan) {
	slices.SortFunc(plan.Edits, func(a, b fileEdit) int {
		return cmp.Compare(a.Path, b.Path)
	})
	slices.SortFunc(plan.StateUpdates, stateUpdateCompare)
	slices.SortFunc(plan.Conflicts, conflictCompare)
	slices.Sort(plan.Warnings)
}

func stateUpdateCompare(a StateUpdate, b StateUpdate) int {
	return cmp.Or(
		cmp.Compare(a.Locale, b.Locale),
		cmp.Compare(a.Namespace, b.Namespace),
		cmp.Compare(a.FromKey, b.FromKey),
	)
}

func conflictCompare(a Conflict, b Conflict) int {
	return cmp.Or(
		cmp.Compare(a.Locale, b.Locale),
		cmp.Compare(a.Namespace, b.Namespace),
		cmp.Compare(a.FilePath, b.FilePath),
		cmp.Compare(a.Reason, b.Reason),
	)
}

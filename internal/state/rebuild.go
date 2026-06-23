package state

import (
	"context"
	"strings"
	"time"

	"github.com/Ret2Hell/i18n-mcp/internal/locale"
)

type Service struct {
	store   *Store
	locales *locale.Service
	now     func() time.Time
}

type RebuildOptions struct {
	Apply bool `json:"apply"`
}

type RebuildResult struct {
	DryRun            bool             `json:"dryRun"`
	Applied           bool             `json:"applied"`
	StatePath         string           `json:"statePath"`
	SourceLocale      string           `json:"sourceLocale"`
	Entries           int              `json:"entries"`
	Created           int              `json:"created"`
	Updated           int              `json:"updated"`
	Skipped           int              `json:"skipped"`
	Assumptions       []string         `json:"assumptions"`
	InventoryWarnings []locale.Warning `json:"inventoryWarnings,omitzero"`
	PreviewState      File             `json:"previewState"`
}

func NewService(store *Store, locales *locale.Service) *Service {
	return &Service{store: store, locales: locales, now: func() time.Time { return time.Now().UTC() }}
}

func (s *Service) Rebuild(ctx context.Context, opts RebuildOptions) (RebuildResult, error) {
	existing, err := s.store.Load(ctx)
	if err != nil {
		return RebuildResult{}, err
	}
	inv, err := s.locales.Inventory(ctx)
	if err != nil {
		return RebuildResult{}, err
	}

	now := s.now()
	next := NewFile(inv.SourceLocale, now)
	sourceByKey := make(map[string]locale.Unit, len(inv.Units))
	targetLocales := make(map[string]struct{}, len(inv.TargetLocales))
	for _, localeCode := range inv.TargetLocales {
		targetLocales[localeCode] = struct{}{}
	}

	for _, unit := range inv.Units {
		if unit.Locale == inv.SourceLocale {
			sourceByKey[unitIdentity(unit.Namespace, unit.Key)] = unit
		}
	}

	var created int
	var updated int
	var skipped int
	for _, unit := range inv.Units {
		if unit.Locale == inv.SourceLocale {
			continue
		}
		if _, ok := targetLocales[unit.Locale]; !ok {
			skipped++
			continue
		}
		if strings.TrimSpace(unit.Value) == "" {
			skipped++
			continue
		}

		sourceUnit, ok := sourceByKey[unitIdentity(unit.Namespace, unit.Key)]
		if !ok {
			skipped++
			continue
		}

		sourceHash := SourceHash(sourceUnit.Value)
		entryKey := EntryKey(unit.Locale, unit.Namespace, unit.Key)
		old, existed := existing.Entries[entryKey]
		entry := Entry{
			Locale:             unit.Locale,
			Namespace:          unit.Namespace,
			Key:                unit.Key,
			SourceHash:         sourceHash,
			TranslatedFromHash: sourceHash,
			TargetHash:         TargetHash(unit.Value),
			Status:             StatusCurrent,
			Reviewed:           old.Reviewed,
			UpdatedAt:          now,
			UpdatedBy:          "state.rebuild",
		}
		next.Entries[entryKey] = entry
		if existed {
			updated++
		} else {
			created++
		}
	}

	result := RebuildResult{
		DryRun:            !opts.Apply,
		Applied:           false,
		StatePath:         s.store.Path(),
		SourceLocale:      inv.SourceLocale,
		Entries:           len(next.Entries),
		Created:           created,
		Updated:           updated,
		Skipped:           skipped,
		Assumptions:       rebuildAssumptions(),
		InventoryWarnings: inv.Warnings,
		PreviewState:      next,
	}
	if opts.Apply {
		if err := s.store.Save(ctx, next); err != nil {
			return RebuildResult{}, err
		}
		result.Applied = true
		result.DryRun = false
	}
	return result, nil
}

func unitIdentity(namespace string, key string) string {
	return namespace + "\x00" + key
}

func rebuildAssumptions() []string {
	return []string{
		"existing non-empty target strings are treated as accepted current translations",
		"target keys without a matching source key are skipped",
		"empty target strings are skipped",
		"rebuild records current state only; placeholder, rich-text tag, and ICU checks are reported by validation tools",
	}
}

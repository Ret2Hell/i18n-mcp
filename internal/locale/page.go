package locale

import (
	"context"
	"slices"
	"strings"

	"github.com/Ret2Hell/i18n-mcp/internal/pagination"
	"github.com/Ret2Hell/i18n-mcp/internal/security"
)

// PageInput controls network-facing locale inventory detail.
type PageInput struct {
	pagination.Input
	IncludeUnits bool     `json:"includeUnits,omitzero" jsonschema:"include a bounded page of units; default returns metadata only"`
	Locales      []string `json:"locales,omitzero"`
	Namespaces   []string `json:"namespaces,omitzero"`
	KeyPrefix    string   `json:"keyPrefix,omitzero"`
}

// Page returns metadata and optionally a bounded unit page, never raw locale JSON.
func (s *Service) Page(ctx context.Context, signer *security.RequestStateSigner, project string, in PageInput) (Inventory, error) {
	inv, err := s.InventoryFiltered(ctx, in.Locales, in.Namespaces)
	if err != nil {
		return Inventory{}, err
	}
	units := inv.Units
	inv.Units = nil
	inv.WarningsTotal = len(inv.Warnings)
	if len(inv.Warnings) > 20 {
		inv.Warnings = inv.Warnings[:20]
	}
	if !in.IncludeUnits {
		return inv, nil
	}
	query := in
	query.Cursor = ""
	builder, err := pagination.New[Unit](ctx, signer, "i18n.locale.page", project, query, in.Input)
	if err != nil {
		return Inventory{}, err
	}
	for _, unit := range units {
		if err := ctx.Err(); err != nil {
			return Inventory{}, err
		}
		if len(in.Locales) > 0 && !slices.Contains(in.Locales, unit.Locale) {
			continue
		}
		if len(in.Namespaces) > 0 && !slices.Contains(in.Namespaces, unit.Namespace) {
			continue
		}
		if !strings.HasPrefix(unit.Key, in.KeyPrefix) {
			continue
		}
		if err := builder.Add(unit); err != nil {
			return Inventory{}, err
		}
	}
	meta, err := builder.Finish()
	if err != nil {
		return Inventory{}, err
	}
	inv.Units = builder.Items
	inv.Page = &meta
	return inv, nil
}

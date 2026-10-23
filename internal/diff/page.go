package diff

import (
	"context"
	"github.com/Ret2Hell/i18n-mcp/internal/pagination"
	"github.com/Ret2Hell/i18n-mcp/internal/security"
)

// PageInput combines analysis filters with paging controls.
type PageInput struct {
	Filter
	pagination.Input
}

// Page computes a bounded page; no full-detail result cache is retained.
func (s *Service) Page(ctx context.Context, signer *security.RequestStateSigner, project string, in PageInput) (Report, error) {
	b, err := pagination.New[KeyDiff](ctx, signer, "i18n.diff.page", project, in.Filter, in.Input)
	if err != nil {
		return Report{}, err
	}
	report, err := s.Walk(ctx, in.Filter, b.Add)
	if err != nil {
		return Report{}, err
	}
	meta, err := b.Finish()
	if err != nil {
		return Report{}, err
	}
	report.Items = b.Items
	report.Page = &meta
	report.WarningsTotal = len(report.Warnings)
	if len(report.Warnings) > 20 {
		report.Warnings = report.Warnings[:20]
	}
	return report, nil
}

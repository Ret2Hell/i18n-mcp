package scanner

import (
	"context"

	"github.com/Ret2Hell/i18n-mcp/internal/config"
	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
	"github.com/Ret2Hell/i18n-mcp/internal/security"
)

const latestReportID = "latest"

// Service scans source files for translation usage.
type Service struct {
	guard  *fsutil.Guard
	config *config.Service
	latest security.Store[Report]
}

// NewService creates a scanner service scoped by guard and configService.
func NewService(guard *fsutil.Guard, configService *config.Service) *Service {
	return &Service{guard: guard, config: configService}
}

// Latest returns the current subject's most recent scan report, if any.
func (s *Service) Latest(ctx context.Context) (Report, bool, error) {
	report, ok, err := s.latest.Get(ctx, latestReportID)
	return cloneReport(report), ok, err
}

func (s *Service) storeLatest(ctx context.Context, report Report) {
	s.latest.Put(ctx, latestReportID, cloneReport(report))
}

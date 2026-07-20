package deadkey

import (
	"context"

	"github.com/Ret2Hell/i18n-mcp/internal/config"
	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
	"github.com/Ret2Hell/i18n-mcp/internal/locale"
	"github.com/Ret2Hell/i18n-mcp/internal/scanner"
	"github.com/Ret2Hell/i18n-mcp/internal/security"
)

// Notifier publishes resource update notifications.
type Notifier interface {
	Updated(ctx context.Context, uris ...string)
}

// Service classifies and prunes unused translation keys.
type Service struct {
	config   *config.Service
	guard    *fsutil.Guard
	locales  *locale.Service
	scanner  *scanner.Service
	Notifier Notifier

	latest security.Store[Report]
}

// NewService creates a dead-key service.
func NewService(configService *config.Service, guard *fsutil.Guard, localeService *locale.Service, scannerService *scanner.Service) *Service {
	return &Service{config: configService, guard: guard, locales: localeService, scanner: scannerService}
}

// Latest returns the current subject's most recent dead-key report, if any.
func (s *Service) Latest(ctx context.Context) (Report, bool, error) {
	report, ok, err := s.latest.Get(ctx, "latest")
	return cloneReport(report), ok, err
}

func (s *Service) storeLatest(ctx context.Context, report Report) {
	s.latest.Put(ctx, "latest", cloneReport(report))
}

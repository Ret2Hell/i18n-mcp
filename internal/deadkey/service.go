package deadkey

import (
	"context"
	"sync"

	"github.com/Ret2Hell/i18n-mcp/internal/config"
	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
	"github.com/Ret2Hell/i18n-mcp/internal/locale"
	"github.com/Ret2Hell/i18n-mcp/internal/scanner"
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

	latestMu sync.RWMutex
	latest   *Report
}

// NewService creates a dead-key service.
func NewService(configService *config.Service, guard *fsutil.Guard, localeService *locale.Service, scannerService *scanner.Service) *Service {
	return &Service{config: configService, guard: guard, locales: localeService, scanner: scannerService}
}

// Latest returns the most recent dead-key report, if any.
func (s *Service) Latest() (Report, bool) {
	s.latestMu.RLock()
	defer s.latestMu.RUnlock()
	if s.latest == nil {
		return Report{}, false
	}
	return cloneReport(*s.latest), true
}

func (s *Service) storeLatest(report Report) {
	s.latestMu.Lock()
	defer s.latestMu.Unlock()
	s.latest = new(cloneReport(report))
}

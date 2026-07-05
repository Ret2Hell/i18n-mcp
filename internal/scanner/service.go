package scanner

import (
	"sync"

	"github.com/Ret2Hell/i18n-mcp/internal/config"
	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
)

// Service scans source files for translation usage.
type Service struct {
	guard  *fsutil.Guard
	config *config.Service

	latestMu sync.RWMutex
	latest   *Report
}

// NewService creates a scanner service scoped by guard and configService.
func NewService(guard *fsutil.Guard, configService *config.Service) *Service {
	return &Service{guard: guard, config: configService}
}

// Latest returns the most recent scan report, if any.
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

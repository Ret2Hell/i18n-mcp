package scanner

import (
	"sync"

	"github.com/Ret2Hell/i18n-mcp/internal/config"
	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
)

type Service struct {
	guard  *fsutil.Guard
	config *config.Service

	latestMu sync.RWMutex
	latest   *Report
}

func NewService(guard *fsutil.Guard, configService *config.Service) *Service {
	return &Service{guard: guard, config: configService}
}

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
	copy := cloneReport(report)
	s.latest = &copy
}

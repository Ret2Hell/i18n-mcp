package deadkey

import (
	"sync"

	"github.com/Ret2Hell/i18n-mcp/internal/config"
	"github.com/Ret2Hell/i18n-mcp/internal/locale"
	"github.com/Ret2Hell/i18n-mcp/internal/scanner"
)

type Service struct {
	config  *config.Service
	locales *locale.Service
	scanner *scanner.Service

	latestMu sync.RWMutex
	latest   *Report
}

func NewService(configService *config.Service, localeService *locale.Service, scannerService *scanner.Service) *Service {
	return &Service{config: configService, locales: localeService, scanner: scannerService}
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

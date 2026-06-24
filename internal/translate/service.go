package translate

import (
	"sync"

	"github.com/Ret2Hell/i18n-mcp/internal/config"
	"github.com/Ret2Hell/i18n-mcp/internal/diff"
	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
	"github.com/Ret2Hell/i18n-mcp/internal/locale"
	"github.com/Ret2Hell/i18n-mcp/internal/state"
	"github.com/Ret2Hell/i18n-mcp/internal/validate"
)

type Service struct {
	config    *config.Service
	guard     *fsutil.Guard
	locales   *locale.Service
	state     *state.Service
	diff      *diff.Service
	validator *validate.Service

	latestMu sync.RWMutex
	latest   *Batch
}

func NewService(configService *config.Service, guard *fsutil.Guard, locales *locale.Service, stateService *state.Service, diffService *diff.Service, validator *validate.Service) *Service {
	return &Service{
		config:    configService,
		guard:     guard,
		locales:   locales,
		state:     stateService,
		diff:      diffService,
		validator: validator,
	}
}

func (s *Service) LatestPlan() (Batch, bool) {
	s.latestMu.RLock()
	defer s.latestMu.RUnlock()
	if s.latest == nil {
		return Batch{}, false
	}
	return cloneBatch(*s.latest), true
}

func (s *Service) storeLatest(batch Batch) {
	s.latestMu.Lock()
	defer s.latestMu.Unlock()
	copy := cloneBatch(batch)
	s.latest = &copy
}

func cloneBatch(batch Batch) Batch {
	batch.TargetLocales = append([]string(nil), batch.TargetLocales...)
	batch.Items = append([]Item(nil), batch.Items...)
	batch.Glossary = append([]GlossaryEntry(nil), batch.Glossary...)
	batch.GlossaryReferences = append([]string(nil), batch.GlossaryReferences...)
	batch.ContextFiles = append([]ContextFileRef(nil), batch.ContextFiles...)
	batch.ValidationRules = append([]string(nil), batch.ValidationRules...)
	batch.ResourceLinks = append([]string(nil), batch.ResourceLinks...)
	batch.Warnings = append([]string(nil), batch.Warnings...)
	for i := range batch.Items {
		batch.Items[i].Placeholders = append([]string(nil), batch.Items[i].Placeholders...)
		batch.Items[i].Tags = append([]string(nil), batch.Items[i].Tags...)
		batch.Items[i].Notes = append([]string(nil), batch.Items[i].Notes...)
	}
	return batch
}

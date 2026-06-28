package translate

import (
	"slices"
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
	batch.TargetLocales = slices.Clone(batch.TargetLocales)
	batch.Items = slices.Clone(batch.Items)
	batch.Glossary = slices.Clone(batch.Glossary)
	batch.GlossaryReferences = slices.Clone(batch.GlossaryReferences)
	batch.ContextFiles = slices.Clone(batch.ContextFiles)
	batch.ValidationRules = slices.Clone(batch.ValidationRules)
	batch.ResourceLinks = slices.Clone(batch.ResourceLinks)
	batch.Warnings = slices.Clone(batch.Warnings)
	for i := range batch.Items {
		batch.Items[i].Placeholders = slices.Clone(batch.Items[i].Placeholders)
		batch.Items[i].Tags = slices.Clone(batch.Items[i].Tags)
		batch.Items[i].Notes = slices.Clone(batch.Items[i].Notes)
	}
	return batch
}

package keyops

import (
	"context"

	"github.com/Ret2Hell/i18n-mcp/internal/config"
	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
	"github.com/Ret2Hell/i18n-mcp/internal/locale"
	"github.com/Ret2Hell/i18n-mcp/internal/state"
)

type Notifier interface {
	Updated(ctx context.Context, uris ...string)
}

type Service struct {
	config   *config.Service
	guard    *fsutil.Guard
	locales  *locale.Service
	state    *state.Service
	Notifier Notifier
}

func NewService(configService *config.Service, guard *fsutil.Guard, localeService *locale.Service, stateService *state.Service) *Service {
	return &Service{config: configService, guard: guard, locales: localeService, state: stateService}
}

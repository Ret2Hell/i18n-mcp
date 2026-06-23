package app

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"github.com/Ret2Hell/i18n-mcp/internal/config"
	"github.com/Ret2Hell/i18n-mcp/internal/diff"
	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
	"github.com/Ret2Hell/i18n-mcp/internal/locale"
	"github.com/Ret2Hell/i18n-mcp/internal/project"
	"github.com/Ret2Hell/i18n-mcp/internal/state"
	"github.com/Ret2Hell/i18n-mcp/internal/validate"
	"github.com/rs/zerolog"
)

type App struct {
	Options     Options
	Logger      *slog.Logger
	ProjectRoot string
	Guard       *fsutil.Guard
	Config      *config.Service
	Project     *project.Service
	Locales     *locale.Service
	State       *state.Service
	Validator   *validate.Service
	Diff        *diff.Service
}

func New(ctx context.Context, opts Options) (*App, error) {
	_ = ctx
	guard, err := fsutil.NewGuard(opts.ProjectRoot)
	if err != nil {
		return nil, err
	}

	opts.ProjectRoot = guard.Root()
	logger := zerolog.New(os.Stderr).
		Level(parseLevel(opts.LogLevel)).
		With().
		Timestamp().
		Logger()

	configService := config.NewService(guard, opts.ConfigPath)
	projectService := project.NewService(guard)
	localeService := locale.NewService(guard, configService)
	stateStore := state.NewStore(guard)
	stateService := state.NewService(stateStore, localeService)
	validatorService := validate.NewService()
	diffService := diff.NewService(localeService, stateService, validatorService)

	return &App{
		Options:     opts,
		Logger:      slog.New(zerolog.NewSlogHandler(logger)),
		ProjectRoot: guard.Root(),
		Guard:       guard,
		Config:      configService,
		Project:     projectService,
		Locales:     localeService,
		State:       stateService,
		Validator:   validatorService,
		Diff:        diffService,
	}, nil
}

func parseLevel(level string) zerolog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		return zerolog.WarnLevel
	}
}

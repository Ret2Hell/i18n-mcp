package app

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Ret2Hell/i18n-mcp/internal/config"
	"github.com/Ret2Hell/i18n-mcp/internal/deadkey"
	"github.com/Ret2Hell/i18n-mcp/internal/diff"
	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
	"github.com/Ret2Hell/i18n-mcp/internal/keyops"
	"github.com/Ret2Hell/i18n-mcp/internal/locale"
	"github.com/Ret2Hell/i18n-mcp/internal/mcpadapter"
	"github.com/Ret2Hell/i18n-mcp/internal/project"
	"github.com/Ret2Hell/i18n-mcp/internal/report"
	"github.com/Ret2Hell/i18n-mcp/internal/scanner"
	"github.com/Ret2Hell/i18n-mcp/internal/state"
	"github.com/Ret2Hell/i18n-mcp/internal/translate"
	"github.com/Ret2Hell/i18n-mcp/internal/validate"
	"github.com/rs/zerolog"
)

// App wires together the application services used by the CLI and MCP server.
type App struct {
	Options       Options
	Logger        *slog.Logger
	ProjectRoot   string
	Guard         *fsutil.Guard
	Config        *config.Service
	Project       *project.Service
	Locales       *locale.Service
	State         *state.Service
	Validator     *validate.Service
	Diff          *diff.Service
	Translation   *translate.Service
	Providers     *translate.ProviderRegistry
	Scanner       *scanner.Service
	DeadKeys      *deadkey.Service
	Reports       *report.Service
	KeyOps        *keyops.Service
	Subscriptions *mcpadapter.SubscriptionRegistry
}

// New constructs an App with all services initialized from opts.
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
	translationService := translate.NewService(configService, guard, localeService, stateService, diffService, validatorService)
	providerRegistry := BuildProviderRegistry(os.LookupEnv, &http.Client{Timeout: 60 * time.Second})
	translationService.Providers = providerRegistry
	scannerService := scanner.NewService(guard, configService)
	deadKeyService := deadkey.NewService(configService, guard, localeService, scannerService)
	reportService := report.NewService(guard.Root(), configService, localeService, diffService, scannerService, deadKeyService)
	keyOpsService := keyops.NewService(configService, guard, localeService, stateService)

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
		Translation: translationService,
		Providers:   providerRegistry,
		Scanner:     scannerService,
		DeadKeys:    deadKeyService,
		Reports:     reportService,
		KeyOps:      keyOpsService,
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

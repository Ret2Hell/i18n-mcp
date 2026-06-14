package app

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
	"github.com/rs/zerolog"
)

type App struct {
	Options     Options
	Logger      *slog.Logger
	ProjectRoot string
	Guard       *fsutil.Guard
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

	return &App{
		Options:     opts,
		Logger:      slog.New(zerolog.NewSlogHandler(logger)),
		ProjectRoot: guard.Root(),
		Guard:       guard,
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

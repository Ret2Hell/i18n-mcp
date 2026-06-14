package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog"
)

type App struct {
	Options     Options
	Logger      *slog.Logger
	ProjectRoot string
}

func New(ctx context.Context, opts Options) (*App, error) {
	_ = ctx
	root := opts.ProjectRoot
	if root == "" {
		root = "."
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("stat project root: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("project root is not a directory: %s", abs)
	}

	opts.ProjectRoot = abs
	logger := zerolog.New(os.Stderr).
		Level(parseLevel(opts.LogLevel)).
		With().
		Timestamp().
		Logger()

	return &App{
		Options:     opts,
		Logger:      slog.New(zerolog.NewSlogHandler(logger)),
		ProjectRoot: abs,
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

package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
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
	return &App{
		Options:     opts,
		Logger:      slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: parseLevel(opts.LogLevel)})),
		ProjectRoot: abs,
	}, nil
}

func parseLevel(level string) slog.Leveler {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "error":
		return slog.LevelError
	default:
		return slog.LevelWarn
	}
}

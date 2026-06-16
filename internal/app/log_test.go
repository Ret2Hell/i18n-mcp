package app

import (
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		name  string
		level string
		want  zerolog.Level
	}{
		{name: "debug", level: "debug", want: zerolog.DebugLevel},
		{name: "info", level: "info", want: zerolog.InfoLevel},
		{name: "error", level: "error", want: zerolog.ErrorLevel},
		{name: "warn default", level: "warn", want: zerolog.WarnLevel},
		{name: "unknown default", level: "verbose", want: zerolog.WarnLevel},
		{name: "case insensitive", level: "DEBUG", want: zerolog.DebugLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, parseLevel(tt.level))
		})
	}
}

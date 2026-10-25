package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Ret2Hell/i18n-mcp/internal/app"
	"github.com/Ret2Hell/i18n-mcp/internal/mcpadapter"
	"github.com/Ret2Hell/i18n-mcp/internal/mcpserver"
	"github.com/stretchr/testify/require"
)

func TestStartLocaleWatcher(t *testing.T) {
	for _, tc := range []struct {
		name, config, locale string
		wantError            bool
	}{
		{name: "malformed config", config: `{`, wantError: true},
		{name: "invalid locale JSON", config: `{"sourceLocale":"en","localeFiles":["messages/{locale}.json"]}`, locale: `{`, wantError: true},
		{name: "starts and cancels", config: `{"sourceLocale":"en","localeFiles":["messages/{locale}.json"]}`, locale: `{"hello":"Hello"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			if tc.config != "" {
				require.NoError(t, os.WriteFile(filepath.Join(root, ".i18n-mcp.json"), []byte(tc.config), 0600))
			}
			if tc.locale != "" {
				require.NoError(t, os.Mkdir(filepath.Join(root, "messages"), 0700))
				require.NoError(t, os.WriteFile(filepath.Join(root, "messages/en.json"), []byte(tc.locale), 0600))
			}
			a, err := app.New(t.Context(), app.Options{ProjectRoot: root})
			require.NoError(t, err)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			err = startLocaleWatcher(ctx, a, mcpadapter.ResourceNotifier{Server: mcpserver.New(a), Logger: a.Logger})
			if tc.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestRunStdioRejectsInvalidProject(t *testing.T) {
	err := runStdio(t.Context(), &RootOptions{Project: filepath.Join(t.TempDir(), "missing")}, true)
	require.Error(t, err)
}

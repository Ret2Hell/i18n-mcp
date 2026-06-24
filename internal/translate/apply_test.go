package translate

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Ret2Hell/i18n-mcp/internal/config"
	"github.com/Ret2Hell/i18n-mcp/internal/diff"
	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
	"github.com/Ret2Hell/i18n-mcp/internal/locale"
	"github.com/Ret2Hell/i18n-mcp/internal/state"
	"github.com/Ret2Hell/i18n-mcp/internal/validate"
	"github.com/stretchr/testify/require"
)

func TestApplyDryRunDoesNotWriteFilesOrState(t *testing.T) {
	root := setupApplyProject(t, `{"hello":"Hello"}`)
	svc, _ := newApplyTestService(t, root)

	out, err := svc.Apply(context.Background(), ApplyInput{
		Apply: false,
		Translations: []ProposedTranslation{{
			Locale: "fr", Namespace: "common", Key: "hello", SourceValue: "Hello", Value: "Bonjour",
		}},
	})

	require.NoError(t, err)
	require.True(t, out.DryRun)
	require.Zero(t, out.Applied)
	require.Len(t, out.ChangedFiles, 1)
	require.True(t, out.ChangedFiles[0].Changed)
	require.False(t, out.ChangedFiles[0].Written)
	require.NoFileExists(t, filepath.Join(root, "locales", "fr.json"))
	require.NoFileExists(t, filepath.Join(root, ".i18n-mcp", "state.json"))
}

func TestApplyWriteUpdatesLocaleThenState(t *testing.T) {
	root := setupApplyProject(t, `{"hello":"Hello"}`)
	svc, stateSvc := newApplyTestService(t, root)

	out, err := svc.Apply(context.Background(), ApplyInput{
		Apply: true,
		Translations: []ProposedTranslation{{
			Locale: "fr", Namespace: "common", Key: "hello", SourceValue: "Hello", Value: "Bonjour",
		}},
	})

	require.NoError(t, err)
	require.False(t, out.DryRun)
	require.Equal(t, 1, out.Applied)
	require.Equal(t, 1, out.StateUpdates)
	require.Len(t, out.ChangedFiles, 1)
	require.Equal(t, "locales/fr.json", out.ChangedFiles[0].Path)
	require.True(t, out.ChangedFiles[0].Written)
	assertFileContent(t, filepath.Join(root, "locales", "fr.json"), "{\n  \"hello\": \"Bonjour\"\n}\n")

	stateFile, err := stateSvc.Load(context.Background())
	require.NoError(t, err)
	entry := stateFile.Entries[state.EntryKey("fr", "common", "hello")]
	require.Equal(t, state.SourceHash("Hello"), entry.SourceHash)
	require.Equal(t, state.SourceHash("Hello"), entry.TranslatedFromHash)
	require.Equal(t, state.TargetHash("Bonjour"), entry.TargetHash)
	require.Equal(t, state.StatusCurrent, entry.Status)
	require.True(t, entry.Reviewed)
	require.Equal(t, "translation.apply", entry.UpdatedBy)
}

func TestApplyRejectsInvalidTranslationsBeforeWrites(t *testing.T) {
	root := setupApplyProject(t, `{"hello":"Hello {name}"}`)
	svc, _ := newApplyTestService(t, root)

	out, err := svc.Apply(context.Background(), ApplyInput{
		Apply: true,
		Translations: []ProposedTranslation{{
			Locale: "fr", Namespace: "common", Key: "hello", SourceValue: "Hello {name}", Value: "Bonjour",
		}},
	})

	require.NoError(t, err)
	require.False(t, out.DryRun)
	require.Zero(t, out.Applied)
	require.Len(t, out.Rejected, 1)
	require.Empty(t, out.ChangedFiles)
	require.NoFileExists(t, filepath.Join(root, "locales", "fr.json"))
	require.NoFileExists(t, filepath.Join(root, ".i18n-mcp", "state.json"))
}

func TestApplyRejectsSourceDriftBeforeWrites(t *testing.T) {
	root := setupApplyProject(t, `{"hello":"Hello"}`)
	svc, _ := newApplyTestService(t, root)

	out, err := svc.Apply(context.Background(), ApplyInput{
		Apply: true,
		Translations: []ProposedTranslation{{
			Locale: "fr", Namespace: "common", Key: "hello", SourceValue: "Old hello", Value: "Bonjour",
		}},
	})

	require.NoError(t, err)
	require.False(t, out.DryRun)
	require.Zero(t, out.Applied)
	require.Len(t, out.Rejected, 1)
	require.Empty(t, out.ChangedFiles)
	require.NoFileExists(t, filepath.Join(root, "locales", "fr.json"))
	require.NoFileExists(t, filepath.Join(root, ".i18n-mcp", "state.json"))
}

func setupApplyProject(t *testing.T, sourceJSON string) string {
	t.Helper()
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "locales"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, "locales", "en.json"), []byte(sourceJSON+"\n"), 0o600))

	cfg := config.Defaults()
	cfg.TargetLocales = []string{"fr"}
	cfg.LocaleFiles = []string{"locales/{locale}.json"}
	data, err := json.Marshal(cfg)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".i18n-mcp"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, config.DefaultConfigFile), data, 0o600))
	return root
}

func newApplyTestService(t *testing.T, root string) (*Service, *state.Service) {
	t.Helper()
	guard, err := fsutil.NewGuard(root)
	require.NoError(t, err)
	configSvc := config.NewService(guard, "")
	localeSvc := locale.NewService(guard, configSvc)
	stateSvc := state.NewService(state.NewStore(guard), localeSvc)
	validatorSvc := validate.NewService()
	diffSvc := diff.NewService(localeSvc, stateSvc, validatorSvc)
	return NewService(configSvc, guard, localeSvc, stateSvc, diffSvc, validatorSvc), stateSvc
}

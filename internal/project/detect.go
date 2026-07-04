package project

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"slices"

	"github.com/Ret2Hell/detect4nextjs"
	"github.com/Ret2Hell/i18n-mcp/internal/config"
	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
)

type ProgressReporter interface {
	Step(ctx context.Context, message string, current int, total int)
}

type DetectOptions struct {
	ProjectRoot string           `json:"projectRoot,omitzero"`
	MaxDepth    int              `json:"maxDepth,omitzero"`
	Progress    ProgressReporter `json:"-"`
}

type Detection struct {
	ProjectRoot      string                `json:"projectRoot"`
	Root             RootInfo              `json:"root"`
	NextJS           NextJSHints           `json:"nextjs"`
	Libraries        []LibraryHint         `json:"libraries,omitzero"`
	DetectedLibrary  string                `json:"detectedLibrary,omitzero"`
	Layouts          []LocaleLayout        `json:"layouts,omitzero"`
	LocaleFiles      []LocaleFileCandidate `json:"localeFiles,omitzero"`
	SourceCandidates []string              `json:"sourceLocaleCandidates,omitzero"`
	TargetLocales    []string              `json:"targetLocales,omitzero"`
	ProposedConfig   config.File           `json:"proposedConfig"`
	Warnings         []string              `json:"warnings,omitzero"`
}

type NextJSHints = detect4nextjs.Hints

type packageJSON struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

func (s *Service) Detect(ctx context.Context, opts DetectOptions) (Detection, error) {
	progress := opts.Progress
	if progress == nil {
		progress = noopProgress{}
	}

	progress.Step(ctx, "checking project files", 1, 4)
	guard, root, err := s.guardFor(ctx, opts.ProjectRoot)
	if err != nil {
		return Detection{}, err
	}

	d := Detection{
		ProjectRoot: root.ProjectRoot,
		Root:        root,
		Warnings:    nil,
	}
	pkg, err := readPackageJSON(guard)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		d.Warnings = append(d.Warnings, err.Error())
	}
	progress.Step(ctx, "checking i18n dependencies", 2, 4)
	nextResult, err := detect4nextjs.Detect(ctx, detect4nextjs.Options{ProjectRoot: guard.Root()})
	if err != nil {
		return Detection{}, err
	}
	d.NextJS = nextResult.Hints
	d.Warnings = append(d.Warnings, nextResult.Warnings...)

	d.Libraries = detectLibraries(pkg)
	d.DetectedLibrary = primaryLibrary(d.Libraries)
	if d.DetectedLibrary == "" {
		d.Warnings = append(d.Warnings, "no supported i18n library dependency was detected")
	}

	progress.Step(ctx, "checking locale layouts", 3, 4)
	layouts, layoutWarnings := detectLocaleLayouts(ctx, guard)
	d.Layouts = layouts
	d.Warnings = append(d.Warnings, layoutWarnings...)
	for _, layout := range layouts {
		d.LocaleFiles = append(d.LocaleFiles, layout.Files...)
	}
	locales := uniqueLocales(layouts)
	d.SourceCandidates = bestSourceCandidates(locales)
	if len(d.SourceCandidates) > 0 {
		source := d.SourceCandidates[0]
		for _, locale := range locales {
			if locale != source {
				d.TargetLocales = append(d.TargetLocales, locale)
			}
		}
	}
	if len(layouts) == 0 {
		d.Warnings = append(d.Warnings, "no common JSON locale layout was detected")
	}

	progress.Step(ctx, "building proposed config", 4, 4)
	d.ProposedConfig = proposeConfig(d)
	if len(d.SourceCandidates) == 0 {
		d.Warnings = append(d.Warnings, "could not infer source locale")
	}
	if len(d.TargetLocales) == 0 {
		d.Warnings = append(d.Warnings, "could not infer target locales")
	}

	slices.Sort(d.Warnings)
	return d, nil
}

type noopProgress struct{}

func (noopProgress) Step(context.Context, string, int, int) {}

func readPackageJSON(guard *fsutil.Guard) (packageJSON, error) {
	path, err := guard.Resolve("package.json")
	if err != nil {
		return packageJSON{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return packageJSON{}, err
	}
	var pkg packageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return packageJSON{}, err
	}
	return pkg, nil
}

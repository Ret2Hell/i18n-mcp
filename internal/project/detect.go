package project

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Ret2Hell/i18n-mcp/internal/config"
	"github.com/Ret2Hell/i18n-mcp/internal/fsutil"
)

type DetectOptions struct {
	ProjectRoot string `json:"projectRoot,omitzero"`
	MaxDepth    int    `json:"maxDepth,omitzero"`
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

type NextJSHints struct {
	PackageJSON                bool     `json:"packageJson"`
	PackageJSONPath            string   `json:"packageJsonPath,omitzero"`
	NextDependency             string   `json:"nextDependency,omitzero"`
	ReactDependency            string   `json:"reactDependency,omitzero"`
	ReactDOMDependency         string   `json:"reactDomDependency,omitzero"`
	NextScripts                []string `json:"nextScripts,omitzero"`
	NextConfigFiles            []string `json:"nextConfigFiles,omitzero"`
	UnsupportedNextConfigFiles []string `json:"unsupportedNextConfigFiles,omitzero"`
	NextEnvDTS                 bool     `json:"nextEnvDts"`
	ProxyFiles                 []string `json:"proxyFiles,omitzero"`
	InstrumentationFiles       []string `json:"instrumentationFiles,omitzero"`
	AppDir                     bool     `json:"appDir"`
	PagesDir                   bool     `json:"pagesDir"`
	SrcDir                     bool     `json:"srcDir"`
	AppRouter                  bool     `json:"appRouter"`
	PagesRouter                bool     `json:"pagesRouter"`
	LooksLikeNextJS            bool     `json:"looksLikeNextjs"`
}

type packageJSON struct {
	Scripts         map[string]string `json:"scripts"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

func (s *Service) Detect(ctx context.Context, opts DetectOptions) (Detection, error) {
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
	d.NextJS = detectNextJS(guard, pkg)
	if !d.NextJS.LooksLikeNextJS {
		d.Warnings = append(d.Warnings, "project root does not look like a Next.js app")
	}

	d.Libraries = detectLibraries(pkg)
	d.DetectedLibrary = primaryLibrary(d.Libraries)
	if d.DetectedLibrary == "" {
		d.Warnings = append(d.Warnings, "no supported i18n library dependency was detected")
	}

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

	d.ProposedConfig = proposeConfig(d)
	if len(d.SourceCandidates) == 0 {
		d.Warnings = append(d.Warnings, "could not infer source locale")
	}
	if len(d.TargetLocales) == 0 {
		d.Warnings = append(d.Warnings, "could not infer target locales")
	}

	sort.Strings(d.Warnings)
	return d, nil
}

func detectNextJS(guard *fsutil.Guard, pkg packageJSON) NextJSHints {
	hints := NextJSHints{}

	if path, err := guard.Resolve("package.json"); err == nil {
		if _, statErr := os.Stat(path); statErr == nil {
			hints.PackageJSON = true
			hints.PackageJSONPath = path
		}
	}

	hints.NextConfigFiles = existingPaths(guard, []string{"next.config.js", "next.config.mjs", "next.config.ts"})
	// Next.js 16 documentation explicitly notes that next.config.cjs and
	// next.config.cts are not supported. Keep reporting them separately so users
	// can diagnose a near-miss without treating them as valid config evidence.
	hints.UnsupportedNextConfigFiles = existingPaths(guard, []string{"next.config.cjs", "next.config.cts"})
	hints.ProxyFiles = existingPaths(guard, []string{"proxy.js", "proxy.ts", filepath.Join("src", "proxy.js"), filepath.Join("src", "proxy.ts")})
	hints.InstrumentationFiles = existingPaths(guard, []string{"instrumentation.js", "instrumentation.ts", filepath.Join("src", "instrumentation.js"), filepath.Join("src", "instrumentation.ts")})

	hints.AppDir = isDir(guard, "app") || isDir(guard, filepath.Join("src", "app"))
	hints.PagesDir = isDir(guard, "pages") || isDir(guard, filepath.Join("src", "pages"))
	hints.SrcDir = isDir(guard, "src")
	hints.NextEnvDTS = isFile(guard, "next-env.d.ts")
	hints.AppRouter = hints.AppDir
	hints.PagesRouter = hints.PagesDir
	hints.NextDependency = dependencyVersion(pkg, "next")
	hints.ReactDependency = dependencyVersion(pkg, "react")
	hints.ReactDOMDependency = dependencyVersion(pkg, "react-dom")
	hints.NextScripts = nextScripts(pkg)
	hints.LooksLikeNextJS = hints.NextDependency != "" || len(hints.NextScripts) > 0 || len(hints.NextConfigFiles) > 0 || hints.NextEnvDTS || hints.AppDir || hints.PagesDir
	return hints
}

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

func dependencyVersion(pkg packageJSON, name string) string {
	if pkg.Dependencies != nil {
		if version := pkg.Dependencies[name]; version != "" {
			return version
		}
	}
	if pkg.DevDependencies != nil {
		if version := pkg.DevDependencies[name]; version != "" {
			return version
		}
	}
	return ""
}

func nextScripts(pkg packageJSON) []string {
	if len(pkg.Scripts) == 0 {
		return nil
	}
	matches := make([]string, 0, len(pkg.Scripts))
	for name, script := range pkg.Scripts {
		if scriptUsesNext(script) {
			matches = append(matches, name)
		}
	}
	sort.Strings(matches)
	return matches
}

func scriptUsesNext(script string) bool {
	for field := range strings.FieldsSeq(script) {
		field = strings.Trim(field, "'\"")
		if field == "next" || strings.HasSuffix(field, "/next") || strings.HasSuffix(field, string(filepath.Separator)+"next") {
			return true
		}
	}
	return false
}

func existingPaths(guard *fsutil.Guard, paths []string) []string {
	matches := make([]string, 0, len(paths))
	for _, path := range paths {
		if isFile(guard, path) {
			resolved, err := guard.Resolve(path)
			if err != nil {
				continue
			}
			matches = append(matches, resolved)
		}
	}
	sort.Strings(matches)
	return matches
}

func isFile(guard *fsutil.Guard, path string) bool {
	info, ok := stat(guard, path)
	return ok && !info.IsDir()
}

func isDir(guard *fsutil.Guard, path string) bool {
	info, ok := stat(guard, path)
	return ok && info.IsDir()
}

func stat(guard *fsutil.Guard, path string) (os.FileInfo, bool) {
	resolved, err := guard.Resolve(path)
	if err != nil {
		return nil, false
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return nil, false
	}
	return info, true
}

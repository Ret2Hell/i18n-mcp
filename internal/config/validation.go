package config

import (
	"context"
	"errors"
	"fmt"
	"os"

	json "github.com/Ret2Hell/i18n-mcp/internal/jsonutil"
)

// ValidationConfig selects the project's message syntax, independently of prose.
type ValidationConfig struct {
	MessageFormat string               `json:"messageFormat,omitzero" jsonschema:"auto (default), icu, interpolation, i18next, or raw; explicit format overrides library detection"`
	Overrides     []ValidationOverride `json:"overrides,omitzero" jsonschema:"explicit scoped policies; exact keys outrank longest prefixes and namespace-only rules"`
	Printf        bool                 `json:"printf,omitzero" jsonschema:"validate printf arguments in addition to message syntax; disabled by default"`
}

// ResolveValidation resolves auto once per operation from guarded package metadata.
// Unknown projects retain source-structure auto detection, never printf guessing.
func (s *Service) ResolveValidation(ctx context.Context, options ValidationConfig) (ValidationConfig, error) {
	if err := ctx.Err(); err != nil {
		return options, err
	}
	needsDetection := options.MessageFormat == "" || options.MessageFormat == "auto"
	for _, rule := range options.Overrides {
		needsDetection = needsDetection || rule.MessageFormat == "auto"
	}
	if !needsDetection {
		return options, nil
	}
	path, err := s.guard.ResolveExisting("package.json")
	if errors.Is(err, os.ErrNotExist) {
		return options, nil
	}
	if err != nil {
		return options, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return options, nil
	}
	if err != nil {
		return options, err
	}
	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return options, fmt.Errorf("detect validation message format: %w", err)
	}
	has := func(name string) bool { return pkg.Dependencies[name] != "" || pkg.DevDependencies[name] != "" }
	detected := "auto"
	switch {
	case has("next-intl"):
		detected = "icu"
	case has("next-i18next") || has("react-i18next") || has("i18next"):
		detected = "i18next"
	}
	if options.MessageFormat == "" || options.MessageFormat == "auto" {
		options.MessageFormat = detected
	}
	options.Overrides = append([]ValidationOverride(nil), options.Overrides...)
	for i := range options.Overrides {
		if options.Overrides[i].MessageFormat == "auto" {
			options.Overrides[i].MessageFormat = detected
		}
	}
	return options, nil
}

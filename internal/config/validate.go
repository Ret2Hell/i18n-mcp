package config

import (
	"context"
	"fmt"
	"slices"
	"strings"
)

type Diagnostic struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitzero"`
}

type ValidationResult struct {
	Valid    bool         `json:"valid"`
	Errors   []Diagnostic `json:"errors,omitzero"`
	Warnings []Diagnostic `json:"warnings,omitzero"`
}

func (s *Service) Validate(ctx context.Context, cfg Resolved) ValidationResult {
	_ = ctx
	var result ValidationResult

	validateLocales(cfg, &result)
	validateLocaleFiles(cfg, &result)
	validateFormat(cfg, &result)
	validateTranslationMode(cfg, &result)
	s.validateTranslationPaths(cfg, &result)

	result.Valid = len(result.Errors) == 0
	return result
}

func validateLocales(cfg Resolved, result *ValidationResult) {
	if strings.TrimSpace(cfg.SourceLocale) == "" {
		result.Errors = append(result.Errors, Diagnostic{Code: "source_locale_required", Field: "sourceLocale", Message: "sourceLocale is required"})
	}

	if slices.Contains(cfg.TargetLocales, cfg.SourceLocale) {
		result.Errors = append(result.Errors, Diagnostic{Code: "target_contains_source", Field: "targetLocales", Message: "targetLocales must not include sourceLocale"})
	}
}

func validateLocaleFiles(cfg Resolved, result *ValidationResult) {
	if len(cfg.LocaleFiles) == 0 {
		result.Errors = append(result.Errors, Diagnostic{Code: "locale_files_required", Field: "localeFiles", Message: "at least one locale file pattern is required"})
	}
	for i, pattern := range cfg.LocaleFiles {
		field := fmt.Sprintf("localeFiles[%d]", i)
		if !strings.Contains(pattern, "{locale}") {
			result.Errors = append(result.Errors, Diagnostic{Code: "locale_pattern_missing_locale", Field: field, Message: "locale file pattern must contain {locale}"})
		}
		if strings.Contains(pattern, "..") {
			result.Errors = append(result.Errors, Diagnostic{Code: "locale_pattern_traversal", Field: field, Message: "locale file pattern must not contain .."})
		}
	}
}

func validateFormat(cfg Resolved, result *ValidationResult) {
	if cfg.Format.Indent < 0 || cfg.Format.Indent > 8 {
		result.Errors = append(result.Errors, Diagnostic{Code: "invalid_indent", Field: "format.indent", Message: "format.indent must be between 0 and 8"})
	}
}

func validateTranslationMode(cfg Resolved, result *ValidationResult) {
	switch cfg.Translation.Mode {
	case "agent", "provider", "sampling":
	case "":
		result.Errors = append(result.Errors, Diagnostic{Code: "translation_mode_required", Field: "translation.mode", Message: "translation.mode is required"})
	default:
		result.Errors = append(result.Errors, Diagnostic{Code: "invalid_translation_mode", Field: "translation.mode", Message: "translation.mode must be agent, provider, or sampling"})
	}
}

func (s *Service) validateTranslationPaths(cfg Resolved, result *ValidationResult) {
	s.validateProjectPath("translation.styleGuidePath", cfg.Translation.StyleGuidePath, result)
	s.validateProjectPath("translation.glossaryPath", cfg.Translation.GlossaryPath, result)
}

func (s *Service) validateProjectPath(field string, value string, result *ValidationResult) {
	if value == "" {
		return
	}
	if _, err := s.guard.Resolve(value); err != nil {
		result.Errors = append(result.Errors, Diagnostic{Code: "path_escapes_project", Field: field, Message: err.Error()})
	}
}

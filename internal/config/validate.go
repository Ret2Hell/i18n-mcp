package config

import (
	"context"
	"fmt"
	"slices"
	"strings"
)

// Diagnostic describes a configuration validation problem.
type Diagnostic struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitzero"`
}

// ValidationResult contains configuration validation errors and warnings.
type ValidationResult struct {
	Valid    bool         `json:"valid"`
	Errors   []Diagnostic `json:"errors,omitzero"`
	Warnings []Diagnostic `json:"warnings,omitzero"`
}

// Validate checks a resolved configuration for errors and warnings.
func (s *Service) Validate(ctx context.Context, cfg Resolved) ValidationResult {
	_ = ctx
	var result ValidationResult

	validateLocales(cfg, &result)
	validateLocaleFiles(cfg, &result)
	validateFormat(cfg, &result)
	if !slices.Contains([]string{"", "auto", "icu", "interpolation", "i18next", "raw"}, cfg.Validation.MessageFormat) {
		result.Errors = append(result.Errors, Diagnostic{Code: "invalid_message_format", Field: "validation.messageFormat", Message: "messageFormat must be auto, icu, interpolation, i18next, or raw"})
	}
	if len(cfg.Validation.Overrides) > 256 {
		result.Errors = append(result.Errors, Diagnostic{Code: "too_many_validation_overrides", Field: "validation.overrides", Message: "at most 256 overrides are allowed"})
	}
	seen := map[string]bool{}
	for i, rule := range cfg.Validation.Overrides {
		field := fmt.Sprintf("validation.overrides[%d]", i)
		invalid := (rule.Namespace == "" && rule.Key == "" && rule.KeyPrefix == "") || (rule.Key != "" && rule.KeyPrefix != "") || (rule.MessageFormat == "" && rule.Printf == nil) || !slices.Contains([]string{"", "auto", "icu", "i18next", "interpolation", "raw"}, rule.MessageFormat)
		id := rule.Namespace + "\x00" + rule.Key + "\x00" + rule.KeyPrefix
		if invalid || seen[id] {
			result.Errors = append(result.Errors, Diagnostic{Code: "invalid_validation_override", Field: field, Message: "provide one unambiguous key/keyPrefix or namespace scope and a supported policy; duplicate scopes are rejected"})
		}
		seen[id] = true
	}
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

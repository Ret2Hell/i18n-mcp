package config

import (
	"bytes"
	"fmt"
	"io"

	json "github.com/Ret2Hell/i18n-mcp/internal/jsonutil"
)

// FileInput is the presence-aware disk/MCP configuration contract. Only the
// source locale and locale patterns are required; absent optional values default.
// It is a replacement with defaults, never a patch against an existing config.
type FileInput struct {
	SourceLocale         string            `json:"sourceLocale" jsonschema:"source locale used as translation source"`
	LocaleFiles          []string          `json:"localeFiles" jsonschema:"locale file patterns containing {locale}"`
	Schema               *string           `json:"$schema,omitzero"`
	TargetLocales        *[]string         `json:"targetLocales,omitzero"`
	DefaultNamespace     *string           `json:"defaultNamespace,omitzero"`
	TranslationFunctions *[]string         `json:"translationFunctions,omitzero"`
	NamespaceFunctions   *[]string         `json:"namespaceFunctions,omitzero"`
	IgnoredKeyPatterns   *[]string         `json:"ignoredKeyPatterns,omitzero"`
	KeptKeyPatterns      *[]string         `json:"keptKeyPatterns,omitzero"`
	DynamicKeyHints      *[]string         `json:"dynamicKeyHints,omitzero"`
	Format               *FormatInput      `json:"format,omitzero"`
	Translation          *TranslationInput `json:"translation,omitzero"`
	Validation           *ValidationInput  `json:"validation,omitzero"`
	CI                   *CIInput          `json:"ci,omitzero"`
}

// FormatInput preserves explicit false and zero formatting settings.
type FormatInput struct {
	SortKeys        *bool `json:"sortKeys,omitzero"`
	Indent          *int  `json:"indent,omitzero"`
	TrailingNewline *bool `json:"trailingNewline,omitzero"`
}

// TranslationInput configures optional translation context files.
type TranslationInput struct {
	StyleGuidePath *string `json:"styleGuidePath,omitzero"`
	GlossaryPath   *string `json:"glossaryPath,omitzero"`
}

// ValidationInput preserves optional message syntax and printf policy.
type ValidationInput struct {
	Overrides     *[]ValidationOverride `json:"overrides,omitzero"`
	MessageFormat *string               `json:"messageFormat,omitzero" jsonschema:"auto, icu, interpolation, i18next, or raw"`
	Printf        *bool                 `json:"printf,omitzero" jsonschema:"explicit opt-in to portable printf validation"`
}

// CIInput preserves explicitly disabled CI policies when applying defaults.
type CIInput struct {
	FailOnMissing  *bool `json:"failOnMissing,omitzero"`
	FailOnStale    *bool `json:"failOnStale,omitzero"`
	FailOnInvalid  *bool `json:"failOnInvalid,omitzero"`
	FailOnDeadKeys *bool `json:"failOnDeadKeys,omitzero"`
}

func provided[T any](dst *T, value *T) {
	if value != nil {
		*dst = *value
	}
}

// NormalizeInput overlays present values on defaults, including explicit false,
// zero and empty arrays. Invalid values remain invalid for semantic validation.
func NormalizeInput(in FileInput) File {
	cfg := Defaults()
	cfg.SourceLocale = in.SourceLocale
	cfg.LocaleFiles = in.LocaleFiles
	provided(&cfg.Schema, in.Schema)
	provided(&cfg.TargetLocales, in.TargetLocales)
	provided(&cfg.DefaultNamespace, in.DefaultNamespace)
	provided(&cfg.TranslationFunctions, in.TranslationFunctions)
	provided(&cfg.NamespaceFunctions, in.NamespaceFunctions)
	provided(&cfg.IgnoredKeyPatterns, in.IgnoredKeyPatterns)
	provided(&cfg.KeptKeyPatterns, in.KeptKeyPatterns)
	provided(&cfg.DynamicKeyHints, in.DynamicKeyHints)
	if in.Format != nil {
		provided(&cfg.Format.SortKeys, in.Format.SortKeys)
		provided(&cfg.Format.Indent, in.Format.Indent)
		provided(&cfg.Format.TrailingNewline, in.Format.TrailingNewline)
	}
	if in.Translation != nil {
		provided(&cfg.Translation.StyleGuidePath, in.Translation.StyleGuidePath)
		provided(&cfg.Translation.GlossaryPath, in.Translation.GlossaryPath)
	}
	if in.Validation != nil {
		provided(&cfg.Validation.Overrides, in.Validation.Overrides)
		provided(&cfg.Validation.MessageFormat, in.Validation.MessageFormat)
		provided(&cfg.Validation.Printf, in.Validation.Printf)
	}
	if in.CI != nil {
		provided(&cfg.CI.FailOnMissing, in.CI.FailOnMissing)
		provided(&cfg.CI.FailOnStale, in.CI.FailOnStale)
		provided(&cfg.CI.FailOnInvalid, in.CI.FailOnInvalid)
		provided(&cfg.CI.FailOnDeadKeys, in.CI.FailOnDeadKeys)
	}
	return cfg
}

func decodeInput(data []byte) (FileInput, error) {
	var in FileInput
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&in); err != nil {
		return in, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return in, fmt.Errorf("configuration must contain exactly one JSON object")
	}
	return in, nil
}

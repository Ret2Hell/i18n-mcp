package translate

import (
	"time"

	"github.com/Ret2Hell/i18n-mcp/internal/diff"
	"github.com/Ret2Hell/i18n-mcp/internal/validate"
)

type PlanInput struct {
	Locales        []string         `json:"locales,omitzero" jsonschema:"target locales; empty means all configured target locales"`
	Namespaces     []string         `json:"namespaces,omitzero" jsonschema:"namespaces to include; empty means all namespaces"`
	Keys           []string         `json:"keys,omitzero" jsonschema:"translation keys to include; empty means all keys"`
	Statuses       []diff.KeyStatus `json:"statuses,omitzero" jsonschema:"diff statuses to include; defaults to missing and stale"`
	MaxItems       int              `json:"maxItems,omitzero" jsonschema:"maximum translation units to include"`
	IncludeContext bool             `json:"includeContext,omitzero" jsonschema:"include configured style guide and glossary references"`
}

type Batch struct {
	BatchID            string           `json:"batchId"`
	SourceLocale       string           `json:"sourceLocale"`
	TargetLocales      []string         `json:"targetLocales"`
	Items              []Item           `json:"items"`
	StyleGuide         string           `json:"styleGuide,omitzero"`
	Glossary           []GlossaryEntry  `json:"glossary,omitzero"`
	GlossaryReferences []string         `json:"glossaryReferences,omitzero"`
	ContextFiles       []ContextFileRef `json:"contextFiles,omitzero"`
	ValidationRules    []string         `json:"validationRules"`
	ResourceLinks      []string         `json:"resourceLinks,omitzero"`
	Warnings           []string         `json:"warnings,omitzero"`
	CreatedAt          time.Time        `json:"createdAt"`
}

type Item struct {
	ID             string         `json:"id"`
	Locale         string         `json:"locale"`
	Namespace      string         `json:"namespace"`
	Key            string         `json:"key"`
	Status         diff.KeyStatus `json:"status"`
	SourceValue    string         `json:"sourceValue"`
	OldValue       string         `json:"oldValue,omitzero"`
	SourceHash     string         `json:"sourceHash"`
	Placeholders   []string       `json:"placeholders,omitzero"`
	Tags           []string       `json:"tags,omitzero"`
	Notes          []string       `json:"notes,omitzero"`
	SourceFilePath string         `json:"sourceFilePath,omitzero"`
	TargetFilePath string         `json:"targetFilePath,omitzero"`
}

type GlossaryEntry struct {
	Term        string `json:"term"`
	Translation string `json:"translation,omitzero"`
	Locale      string `json:"locale,omitzero"`
	Notes       string `json:"notes,omitzero"`
}

type ContextFileRef struct {
	Kind string `json:"kind"`
	Path string `json:"path"`
}

type ProposedTranslation struct {
	ID          string `json:"id,omitzero"`
	BatchID     string `json:"batchId,omitzero"`
	Locale      string `json:"locale"`
	Namespace   string `json:"namespace"`
	Key         string `json:"key"`
	SourceValue string `json:"sourceValue"`
	Value       string `json:"value"`
}

type ValidationInput struct {
	BatchID          string                `json:"batchId,omitzero" jsonschema:"expected translation batch id"`
	Translations     []ProposedTranslation `json:"translations" jsonschema:"proposed translations to validate"`
	AllowSourceDrift bool                  `json:"allowSourceDrift,omitzero" jsonschema:"allow proposals generated from stale source text"`
}

type ValidationOutput struct {
	BatchID  string                 `json:"batchId,omitzero"`
	Accepted []ValidatedTranslation `json:"accepted"`
	Rejected []RejectedTranslation  `json:"rejected,omitzero"`
	Summary  ValidationSummary      `json:"summary"`
}

type ValidatedTranslation struct {
	ProposedTranslation
	SourceHash     string           `json:"sourceHash"`
	TargetHash     string           `json:"targetHash"`
	TargetFilePath string           `json:"targetFilePath,omitzero"`
	Warnings       []validate.Issue `json:"warnings,omitzero"`
}

type RejectedTranslation struct {
	ProposedTranslation
	Issues []validate.Issue `json:"issues"`
}

type ValidationSummary struct {
	Total    int `json:"total"`
	Accepted int `json:"accepted"`
	Rejected int `json:"rejected"`
}

type ApplyInput struct {
	BatchID          string                `json:"batchId,omitzero" jsonschema:"expected translation batch id"`
	Translations     []ProposedTranslation `json:"translations" jsonschema:"translations to validate and apply"`
	DryRun           *bool                 `json:"dryRun,omitzero" jsonschema:"when true, preview changes without writing"`
	Apply            bool                  `json:"apply,omitzero" jsonschema:"must be true to write locale files and state"`
	AllowSourceDrift bool                  `json:"allowSourceDrift,omitzero" jsonschema:"allow proposals generated from stale source text"`
}

type ApplyOutput struct {
	DryRun       bool                  `json:"dryRun"`
	Applied      int                   `json:"applied"`
	ChangedFiles []ChangedFile         `json:"changedFiles"`
	Rejected     []RejectedTranslation `json:"rejected,omitzero"`
	StateUpdates int                   `json:"stateUpdates,omitzero"`
	Warnings     []string              `json:"warnings,omitzero"`
}

type ChangedFile struct {
	Path    string `json:"path"`
	Diff    string `json:"diff,omitzero"`
	Changed bool   `json:"changed"`
	Written bool   `json:"written,omitzero"`
}

func (in ApplyInput) DryRunValue() bool {
	if in.DryRun != nil {
		return *in.DryRun || !in.Apply
	}
	return !in.Apply
}

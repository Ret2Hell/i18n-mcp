package config

// File describes the on-disk i18n MCP configuration file.
type File struct {
	Schema               string            `json:"$schema,omitzero" jsonschema:"schema URL for editor support"`
	SourceLocale         string            `json:"sourceLocale" jsonschema:"source locale used as translation source"`
	TargetLocales        []string          `json:"targetLocales" jsonschema:"target locales to maintain"`
	LocaleFiles          []string          `json:"localeFiles" jsonschema:"locale file patterns containing {locale} and optional {namespace}"`
	DefaultNamespace     string            `json:"defaultNamespace,omitzero" jsonschema:"namespace used when file layout has no namespace segment"`
	TranslationFunctions []string          `json:"translationFunctions,omitzero" jsonschema:"reserved while source usage scanning is disabled"`
	NamespaceFunctions   []string          `json:"namespaceFunctions,omitzero" jsonschema:"reserved while source usage scanning is disabled"`
	IgnoredKeyPatterns   []string          `json:"ignoredKeyPatterns,omitzero" jsonschema:"reserved while dead-key analysis is disabled"`
	KeptKeyPatterns      []string          `json:"keptKeyPatterns,omitzero" jsonschema:"reserved while dead-key analysis is disabled"`
	DynamicKeyHints      []string          `json:"dynamicKeyHints,omitzero" jsonschema:"reserved while dead-key analysis is disabled"`
	Format               FormatConfig      `json:"format" jsonschema:"locale JSON formatting preferences"`
	Validation           ValidationConfig  `json:"validation,omitzero" jsonschema:"message format and optional printf validation"`
	Translation          TranslationConfig `json:"translation" jsonschema:"optional translation context files"`
	CI                   CIConfig          `json:"ci" jsonschema:"CI audit failure policy"`
}

// FormatConfig controls locale JSON formatting.
type FormatConfig struct {
	SortKeys        bool `json:"sortKeys" jsonschema:"sort JSON object keys when writing locale files"`
	Indent          int  `json:"indent" jsonschema:"number of spaces for JSON indentation"`
	TrailingNewline bool `json:"trailingNewline" jsonschema:"write trailing newline at end of JSON files"`
}

// TranslationConfig specifies optional context files for translation plans.
type TranslationConfig struct {
	StyleGuidePath string `json:"styleGuidePath,omitzero" jsonschema:"optional project-relative style guide path"`
	GlossaryPath   string `json:"glossaryPath,omitzero" jsonschema:"optional project-relative glossary path"`
}

// CIConfig controls audit failure policy for CI runs.
type CIConfig struct {
	FailOnMissing  bool `json:"failOnMissing" jsonschema:"fail audit when missing translations are detected"`
	FailOnStale    bool `json:"failOnStale" jsonschema:"fail audit when stale translations are detected"`
	FailOnInvalid  bool `json:"failOnInvalid" jsonschema:"fail audit when invalid translations are detected"`
	FailOnDeadKeys bool `json:"failOnDeadKeys" jsonschema:"reserved for compatibility; ignored while dead-key analysis is disabled"`
}

// Resolved is a configuration file plus project-specific resolution metadata.
type Resolved struct {
	File
	ProjectRoot string `json:"projectRoot"`
	ConfigPath  string `json:"configPath,omitzero"`
	Exists      bool   `json:"exists"`
}

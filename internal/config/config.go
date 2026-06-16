package config

type File struct {
	Schema               string            `json:"$schema,omitzero" jsonschema:"schema URL for editor support"`
	SourceLocale         string            `json:"sourceLocale" jsonschema:"source locale used as translation source"`
	TargetLocales        []string          `json:"targetLocales" jsonschema:"target locales to maintain"`
	LocaleFiles          []string          `json:"localeFiles" jsonschema:"locale file patterns containing {locale} and optional {namespace}"`
	DefaultNamespace     string            `json:"defaultNamespace,omitzero" jsonschema:"namespace used when file layout has no namespace segment"`
	TranslationFunctions []string          `json:"translationFunctions,omitzero" jsonschema:"translation function names to scan in app code"`
	NamespaceFunctions   []string          `json:"namespaceFunctions,omitzero" jsonschema:"functions that bind a translation namespace"`
	IgnoredKeyPatterns   []string          `json:"ignoredKeyPatterns,omitzero" jsonschema:"dead-key patterns to ignore"`
	KeptKeyPatterns      []string          `json:"keptKeyPatterns,omitzero" jsonschema:"dead-key patterns to always keep"`
	DynamicKeyHints      []string          `json:"dynamicKeyHints,omitzero" jsonschema:"patterns that may be used dynamically"`
	Format               FormatConfig      `json:"format" jsonschema:"locale JSON formatting preferences"`
	Translation          TranslationConfig `json:"translation" jsonschema:"translation mode and optional context files"`
}

type FormatConfig struct {
	SortKeys        bool `json:"sortKeys" jsonschema:"sort JSON object keys when writing locale files"`
	Indent          int  `json:"indent" jsonschema:"number of spaces for JSON indentation"`
	TrailingNewline bool `json:"trailingNewline" jsonschema:"write trailing newline at end of JSON files"`
}

type TranslationConfig struct {
	Mode           string `json:"mode" jsonschema:"translation mode: agent, provider, or sampling"`
	Provider       string `json:"provider,omitzero" jsonschema:"optional provider name for provider mode"`
	StyleGuidePath string `json:"styleGuidePath,omitzero" jsonschema:"optional project-relative style guide path"`
	GlossaryPath   string `json:"glossaryPath,omitzero" jsonschema:"optional project-relative glossary path"`
}

type Resolved struct {
	File
	ProjectRoot string `json:"projectRoot"`
	ConfigPath  string `json:"configPath,omitzero"`
	Exists      bool   `json:"exists"`
}

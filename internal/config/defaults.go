package config

// DefaultConfigFile is the default project-relative configuration filename.
const DefaultConfigFile = ".i18n-mcp.json"

// Defaults returns the default configuration values.
func Defaults() File {
	return File{
		SourceLocale:         "en",
		TargetLocales:        nil,
		LocaleFiles:          []string{"messages/{locale}.json", "locales/{locale}.json"},
		DefaultNamespace:     "common",
		TranslationFunctions: []string{"t", "i18n.t"},
		NamespaceFunctions:   []string{"useTranslations", "getTranslations"},
		Format: FormatConfig{
			SortKeys:        false,
			Indent:          2,
			TrailingNewline: true,
		},
		Translation: TranslationConfig{Mode: "agent"},
		CI: CIConfig{
			FailOnMissing:  true,
			FailOnStale:    false,
			FailOnInvalid:  true,
			FailOnDeadKeys: false,
		},
	}
}

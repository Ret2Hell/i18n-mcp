package config

const DefaultConfigFile = ".i18n-mcp.json"

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
	}
}

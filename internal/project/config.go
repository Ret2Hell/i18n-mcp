package project

import (
	"slices"

	"github.com/Ret2Hell/i18n-mcp/internal/config"
)

func proposeConfig(d Detection) config.File {
	cfg := config.Defaults()

	if len(d.SourceCandidates) > 0 {
		cfg.SourceLocale = d.SourceCandidates[0]
	}
	cfg.TargetLocales = slices.Clone(d.TargetLocales)

	if len(d.Layouts) > 0 {
		cfg.LocaleFiles = []string{d.Layouts[0].Pattern}
	}

	switch d.DetectedLibrary {
	case "next-intl":
		cfg.TranslationFunctions = []string{"t"}
		cfg.NamespaceFunctions = []string{"useTranslations", "getTranslations"}
	case "next-i18next", "react-i18next":
		cfg.TranslationFunctions = []string{"t", "i18n.t"}
		cfg.NamespaceFunctions = []string{"useTranslation"}
	case "i18next":
		cfg.TranslationFunctions = []string{"t", "i18n.t"}
		cfg.NamespaceFunctions = nil
	case "next-translate":
		cfg.TranslationFunctions = []string{"t"}
		cfg.NamespaceFunctions = []string{"useTranslation"}
	}

	return cfg
}

package project

import (
	"cmp"
	"slices"
)

type LibraryHint struct {
	Name       string `json:"name"`
	Version    string `json:"version,omitzero"`
	Source     string `json:"source"`
	Confidence string `json:"confidence"`
}

var knownLibraries = []string{
	"next-intl",
	"next-i18next",
	"react-i18next",
	"i18next",
	"next-translate",
}

func detectLibraries(pkg packageJSON) []LibraryHint {
	var hints []LibraryHint
	for _, name := range knownLibraries {
		if version := cmp.Or(pkg.Dependencies[name], pkg.DevDependencies[name]); version != "" {
			hints = append(hints, LibraryHint{
				Name:       name,
				Version:    version,
				Source:     "package.json",
				Confidence: "high",
			})
		}
	}
	slices.SortFunc(hints, func(a, b LibraryHint) int {
		return cmp.Compare(libraryRank(a.Name), libraryRank(b.Name))
	})
	return hints
}

func primaryLibrary(hints []LibraryHint) string {
	if len(hints) == 0 {
		return ""
	}
	return hints[0].Name
}

func libraryRank(name string) int {
	switch name {
	case "next-intl":
		return 0
	case "next-i18next":
		return 1
	case "next-translate":
		return 2
	case "react-i18next":
		return 3
	case "i18next":
		return 4
	default:
		return 100
	}
}

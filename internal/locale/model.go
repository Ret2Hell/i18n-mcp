package locale

type FileRef struct {
	Locale    string `json:"locale"`
	Namespace string `json:"namespace,omitzero"`
	Path      string `json:"path"`
	Pattern   string `json:"pattern"`
}

type Unit struct {
	Locale    string   `json:"locale"`
	Namespace string   `json:"namespace"`
	Key       string   `json:"key"`
	Value     string   `json:"value"`
	FilePath  string   `json:"filePath"`
	JSONPath  []string `json:"jsonPath"`
}

type Warning struct {
	Code      string   `json:"code"`
	Message   string   `json:"message"`
	Locale    string   `json:"locale,omitzero"`
	Namespace string   `json:"namespace,omitzero"`
	FilePath  string   `json:"filePath,omitzero"`
	Key       string   `json:"key,omitzero"`
	JSONPath  []string `json:"jsonPath,omitzero"`
}

type FlattenResult struct {
	Units    []Unit    `json:"units"`
	Warnings []Warning `json:"warnings,omitzero"`
}

type Inventory struct {
	SourceLocale      string                    `json:"sourceLocale"`
	TargetLocales     []string                  `json:"targetLocales"`
	Locales           []string                  `json:"locales"`
	Namespaces        []string                  `json:"namespaces"`
	Files             []FileSummary             `json:"files"`
	Units             []Unit                    `json:"units,omitzero"`
	CountsByLocale    map[string]int            `json:"countsByLocale"`
	CountsByNamespace map[string]int            `json:"countsByNamespace"`
	Warnings          []Warning                 `json:"warnings,omitzero"`
	Duplicates        []DuplicateNamespaceIssue `json:"duplicates,omitzero"`
}

type FileSummary struct {
	FileRef
	StringKeys int `json:"stringKeys"`
	Bytes      int `json:"bytes"`
}

type DuplicateNamespaceIssue struct {
	Locale    string   `json:"locale"`
	Namespace string   `json:"namespace"`
	Paths     []string `json:"paths"`
}

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

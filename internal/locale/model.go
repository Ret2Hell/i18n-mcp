package locale

type FileRef struct {
	Locale    string `json:"locale"`
	Namespace string `json:"namespace,omitzero"`
	Path      string `json:"path"`
	Pattern   string `json:"pattern"`
}

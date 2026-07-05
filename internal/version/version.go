package version

// AppName is the CLI and application name.
const AppName = "i18n-mcp"

// Info describes build and version metadata.
type Info struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Built   string `json:"built"`
}

// Build metadata set by release tooling.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// Get returns the current build and version metadata.
func Get() Info {
	return Info{
		Name:    AppName,
		Version: Version,
		Commit:  Commit,
		Built:   Date,
	}
}

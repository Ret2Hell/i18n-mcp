package version

const AppName = "i18n-mcp"

type Info struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Built   string `json:"built"`
}

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func Get() Info {
	return Info{
		Name:    AppName,
		Version: Version,
		Commit:  Commit,
		Built:   Date,
	}
}

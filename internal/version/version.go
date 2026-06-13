package version

type Info struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
	Dirty   string `json:"dirty"`
}

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
	Dirty   = "unknown"
)

func Get() Info {
	return Info{
		Version: Version,
		Commit:  Commit,
		Date:    Date,
		Dirty:   Dirty,
	}
}

package state

import "time"

const (
	CurrentVersion   = 1
	DefaultStatePath = ".i18n-mcp/state.json"
)

type Status string

const (
	StatusCurrent Status = "current"
	StatusMissing Status = "missing"
	StatusStale   Status = "stale"
	StatusExtra   Status = "extra"
	StatusInvalid Status = "invalid"
	StatusUnknown Status = "unknown"
)

type File struct {
	Version      int              `json:"version"`
	SourceLocale string           `json:"sourceLocale"`
	Entries      map[string]Entry `json:"entries"`
	UpdatedAt    time.Time        `json:"updatedAt"`
}

type Entry struct {
	Locale             string    `json:"locale"`
	Namespace          string    `json:"namespace"`
	Key                string    `json:"key"`
	SourceHash         string    `json:"sourceHash"`
	TranslatedFromHash string    `json:"translatedFromHash"`
	TargetHash         string    `json:"targetHash,omitzero"`
	Status             Status    `json:"status"`
	Reviewed           bool      `json:"reviewed,omitzero"`
	UpdatedAt          time.Time `json:"updatedAt"`
	UpdatedBy          string    `json:"updatedBy,omitzero"`
}

func NewFile(sourceLocale string, now time.Time) File {
	return File{
		Version:      CurrentVersion,
		SourceLocale: sourceLocale,
		Entries:      map[string]Entry{},
		UpdatedAt:    now.UTC(),
	}
}

func EmptyFile() File {
	return File{Version: CurrentVersion, Entries: map[string]Entry{}}
}

func (f *File) Normalize() {
	if f.Version == 0 {
		f.Version = CurrentVersion
	}
	if f.Entries == nil {
		f.Entries = map[string]Entry{}
	}
}

func EntryKey(locale string, namespace string, key string) string {
	return locale + "\x00" + namespace + "\x00" + key
}

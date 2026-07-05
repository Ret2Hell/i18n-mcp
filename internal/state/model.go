package state

import "time"

// State file constants.
const (
	CurrentVersion   = 1
	DefaultStatePath = ".i18n-mcp/state.json"
)

// Status records translation state for a tracked key.
type Status string

// Translation state statuses.
const (
	StatusCurrent Status = "current"
	StatusMissing Status = "missing"
	StatusStale   Status = "stale"
	StatusExtra   Status = "extra"
	StatusInvalid Status = "invalid"
	StatusUnknown Status = "unknown"
)

// File is the persisted translation state file.
type File struct {
	Version      int              `json:"version"`
	SourceLocale string           `json:"sourceLocale"`
	Entries      map[string]Entry `json:"entries"`
	UpdatedAt    time.Time        `json:"updatedAt"`
}

// Entry records translation state for one locale key.
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

// NewFile creates an initialized state file.
func NewFile(sourceLocale string, now time.Time) File {
	return File{
		Version:      CurrentVersion,
		SourceLocale: sourceLocale,
		Entries:      map[string]Entry{},
		UpdatedAt:    now.UTC(),
	}
}

// EmptyFile returns an empty initialized state file.
func EmptyFile() File {
	return File{Version: CurrentVersion, Entries: map[string]Entry{}}
}

// Normalize fills missing state file defaults.
func (f *File) Normalize() {
	if f.Version == 0 {
		f.Version = CurrentVersion
	}
	if f.Entries == nil {
		f.Entries = map[string]Entry{}
	}
}

// EntryKey returns the stable map key for a state entry.
func EntryKey(locale string, namespace string, key string) string {
	return locale + "\x00" + namespace + "\x00" + key
}

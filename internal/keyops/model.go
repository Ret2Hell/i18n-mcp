package keyops

// ConflictPolicy controls rename behavior when the destination exists.
type ConflictPolicy string

// Conflict policies for key rename operations.
const (
	ConflictReject    ConflictPolicy = "reject"
	ConflictOverwrite ConflictPolicy = "overwrite"
	ConflictSkip      ConflictPolicy = "skip"
)

// RenameInput configures a key rename operation.
type RenameInput struct {
	Namespace      string         `json:"namespace" jsonschema:"namespace containing the key to rename"`
	FromKey        string         `json:"fromKey" jsonschema:"existing flattened key path"`
	ToKey          string         `json:"toKey" jsonschema:"new flattened key path"`
	Locales        []string       `json:"locales,omitzero" jsonschema:"locales to update; empty means all locales"`
	DryRun         *bool          `json:"dryRun,omitzero" jsonschema:"when true, preview changes without writing"`
	Apply          bool           `json:"apply,omitzero" jsonschema:"must be true to write locale files and state"`
	ConflictPolicy ConflictPolicy `json:"conflictPolicy,omitzero" jsonschema:"destination conflict policy: reject, overwrite, or skip"`
}

// RenameOutput describes the result of a key rename operation.
type RenameOutput struct {
	DryRun       bool          `json:"dryRun"`
	Planned      int           `json:"planned"`
	Renamed      int           `json:"renamed"`
	ChangedFiles []ChangedFile `json:"changedFiles"`
	StateUpdates []StateUpdate `json:"stateUpdates,omitzero"`
	Conflicts    []Conflict    `json:"conflicts,omitzero"`
	Warnings     []string      `json:"warnings,omitzero"`
}

// ChangedFile describes changes made or previewed for a locale file.
type ChangedFile struct {
	Path    string `json:"path"`
	Diff    string `json:"diff,omitzero"`
	Changed bool   `json:"changed"`
	Written bool   `json:"written,omitzero"`
}

// StateUpdate describes a state entry key rename.
type StateUpdate struct {
	Locale    string `json:"locale"`
	Namespace string `json:"namespace"`
	FromKey   string `json:"fromKey"`
	ToKey     string `json:"toKey"`
}

// Conflict describes a reason a rename could not proceed.
type Conflict struct {
	Locale    string `json:"locale,omitzero"`
	Namespace string `json:"namespace,omitzero"`
	FromKey   string `json:"fromKey,omitzero"`
	ToKey     string `json:"toKey,omitzero"`
	FilePath  string `json:"filePath,omitzero"`
	Reason    string `json:"reason"`
}

// Plan contains planned file edits and state updates for a rename.
type Plan struct {
	Input        RenameInput
	Edits        []fileEdit
	StateUpdates []StateUpdate
	Conflicts    []Conflict
	Warnings     []string
}

type fileEdit struct {
	Path      string
	Before    []byte
	After     []byte
	Locale    string
	Namespace string
	Changed   bool
}

// DryRunValue returns the effective dry-run setting.
func (in RenameInput) DryRunValue() bool {
	if in.DryRun != nil {
		return *in.DryRun || !in.Apply
	}
	return !in.Apply
}

// NormalizedConflictPolicy returns the requested policy or the default.
func (in RenameInput) NormalizedConflictPolicy() ConflictPolicy {
	switch in.ConflictPolicy {
	case ConflictOverwrite, ConflictSkip:
		return in.ConflictPolicy
	default:
		return ConflictReject
	}
}

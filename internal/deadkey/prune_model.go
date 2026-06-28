package deadkey

type PruneInput struct {
	Keys        []PruneKey `json:"keys" jsonschema:"exact namespace and key pairs to prune"`
	DryRun      *bool      `json:"dryRun,omitempty" jsonschema:"when true, preview changes without writing"`
	Apply       bool       `json:"apply,omitempty" jsonschema:"must be true to write locale files"`
	AllowUnsafe bool       `json:"allowUnsafe,omitempty" jsonschema:"allow pruning used, maybe_dynamic, ignored, or kept keys"`
}

type PruneKey struct {
	Namespace string `json:"namespace"`
	Key       string `json:"key"`
}

type PruneOutput struct {
	DryRun       bool          `json:"dryRun"`
	Pruned       int           `json:"pruned"`
	ChangedFiles []ChangedFile `json:"changedFiles"`
	Rejected     []PruneReject `json:"rejected,omitzero"`
	Warnings     []string      `json:"warnings,omitzero"`
}

type PruneReject struct {
	Namespace string `json:"namespace"`
	Key       string `json:"key"`
	Reason    string `json:"reason"`
	Status    Status `json:"status,omitempty"`
}

type ChangedFile struct {
	Path    string `json:"path"`
	Diff    string `json:"diff,omitempty"`
	Changed bool   `json:"changed"`
	Written bool   `json:"written,omitempty"`
}

func (in PruneInput) DryRunValue() bool {
	if in.DryRun != nil {
		return *in.DryRun || !in.Apply
	}
	return !in.Apply
}

package steward

type Spec struct {
	Version        string   `json:"version,omitempty"`
	Name           string   `json:"name"`
	StewardModel   string   `json:"steward_model"`
	Hooks          []string `json:"hooks"`
	Inputs         []string `json:"inputs"`
	AllowedActions []string `json:"allowed_actions"`
	ArtifactTypes  []string `json:"artifact_types,omitempty"`
	RequiredGuards []string `json:"required_guards"`
}

type Summary struct {
	Name           string
	Hooks          int
	Inputs         int
	AllowedActions int
	ArtifactTypes  int
	RequiredGuards int
}

type Proposal struct {
	Version string   `json:"version,omitempty"`
	ID      string   `json:"id"`
	Steward string   `json:"steward"`
	Hook    string   `json:"hook"`
	Outputs []Output `json:"outputs"`
}

type Output struct {
	Action       string   `json:"action"`
	Reason       string   `json:"reason,omitempty"`
	Role         string   `json:"role,omitempty"`
	Position     string   `json:"position,omitempty"`
	Text         string   `json:"text,omitempty"`
	ArtifactType string   `json:"artifact_type,omitempty"`
	ContentHash  string   `json:"content_hash,omitempty"`
	Ref          string   `json:"ref,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	Severity     string   `json:"severity,omitempty"`
	NoteHash     string   `json:"note_hash,omitempty"`
}

type ProposalSummary struct {
	ID      string
	Steward string
	Hook    string
	Outputs int
}

type DryRunResult struct {
	ProposalID     string        `json:"proposal_id"`
	Steward        string        `json:"steward"`
	Hook           string        `json:"hook"`
	AppliedActions []string      `json:"applied_actions"`
	RequestPatches []DryRunPatch `json:"request_patches,omitempty"`
	Artifacts      []DryRunRef   `json:"artifacts,omitempty"`
	Diagnostics    []DryRunRef   `json:"diagnostics,omitempty"`
	SessionTags    []string      `json:"session_tags,omitempty"`
}

type DryRunRef struct {
	Type        string `json:"type,omitempty"`
	ContentHash string `json:"content_hash,omitempty"`
	NoteHash    string `json:"note_hash,omitempty"`
	Ref         string `json:"ref"`
	Severity    string `json:"severity,omitempty"`
}

type DryRunPatch struct {
	Action       string `json:"action"`
	Target       string `json:"target"`
	InsertIndex  int    `json:"insert_index"`
	Role         string `json:"role"`
	Position     string `json:"position"`
	ContentHash  string `json:"content_hash"`
	ContentChars int    `json:"content_chars"`
}

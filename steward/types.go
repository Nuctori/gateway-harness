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
	Action           string   `json:"action"`
	Reason           string   `json:"reason,omitempty"`
	Role             string   `json:"role,omitempty"`
	Position         string   `json:"position,omitempty"`
	Text             string   `json:"text,omitempty"`
	Strategy         string   `json:"strategy,omitempty"`
	KeepLastMessages int      `json:"keep_last_messages,omitempty"`
	PreserveRoles    []string `json:"preserve_roles,omitempty"`
	ArtifactType     string   `json:"artifact_type,omitempty"`
	ContentHash      string   `json:"content_hash,omitempty"`
	Ref              string   `json:"ref,omitempty"`
	PatchHash        string   `json:"patch_hash,omitempty"`
	Description      string   `json:"description,omitempty"`
	Tags             []string `json:"tags,omitempty"`
	Severity         string   `json:"severity,omitempty"`
	NoteHash         string   `json:"note_hash,omitempty"`
}

type ProposalSummary struct {
	ID      string
	Steward string
	Hook    string
	Outputs int
}

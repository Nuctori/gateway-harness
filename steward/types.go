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

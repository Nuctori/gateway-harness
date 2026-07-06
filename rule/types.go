package rule

type Document struct {
	Version string `json:"version,omitempty"`
	Rules   []Rule `json:"rules"`
}

type Rule struct {
	Name      string    `json:"name"`
	Tags      []string  `json:"tags,omitempty"`
	Trigger   Trigger   `json:"trigger"`
	Scope     Scope     `json:"scope"`
	Operation Operation `json:"operation"`
	Audit     Audit     `json:"audit,omitempty"`
}

type Trigger struct {
	Hooks          []string `json:"hooks"`
	ContinuityDrop bool     `json:"continuity_drop,omitempty"`
}

type Scope struct {
	Models       []string `json:"models"`
	ModelMatches string   `json:"model_matches,omitempty"`
}

type Operation struct {
	Type     string `json:"type"`
	Role     string `json:"role"`
	Position string `json:"position"`
	Text     string `json:"text"`
	Reason   string `json:"reason,omitempty"`
}

type Audit struct {
	LedgerRef    string   `json:"ledger_ref,omitempty"`
	ArtifactRefs []string `json:"artifact_refs,omitempty"`
}

type Summary struct {
	Rules      int
	Hooks      []string
	Operations []string
}

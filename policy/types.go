package policy

type Policy struct {
	Version  string    `json:"version,omitempty"`
	Programs []Program `json:"programs"`
}

type Program struct {
	Name   string   `json:"name"`
	Models []string `json:"models"`
	Tags   []string `json:"tags,omitempty"`
	Steps  []Step   `json:"steps"`
}

type Step struct {
	Hook  string    `json:"hook,omitempty"`
	Hooks []string  `json:"hooks,omitempty"`
	When  Condition `json:"when,omitempty"`
	Do    []Action  `json:"do"`
}

type Condition struct {
	ModelMatches      string `json:"model_matches,omitempty"`
	EstimatedTokensGT int    `json:"estimated_tokens_gt,omitempty"`
}

type Action struct {
	Action           string   `json:"action"`
	Role             string   `json:"role,omitempty"`
	Position         string   `json:"position,omitempty"`
	Text             string   `json:"text,omitempty"`
	Strategy         string   `json:"strategy,omitempty"`
	KeepLastMessages int      `json:"keep_last_messages,omitempty"`
	PreserveRoles    []string `json:"preserve_roles,omitempty"`
	Reason           string   `json:"reason,omitempty"`
}

type Summary struct {
	Programs int
	Steps    int
	Actions  int
	Hooks    []string
}

type DryRunOptions struct {
	Hook            string
	Model           string
	EstimatedTokens int
}

type DryRunResult struct {
	Hook            string        `json:"hook"`
	Model           string        `json:"model"`
	EstimatedTokens int           `json:"estimated_tokens,omitempty"`
	MatchedPrograms []string      `json:"matched_programs,omitempty"`
	AppliedActions  []string      `json:"applied_actions,omitempty"`
	SkippedActions  []Skipped     `json:"skipped_actions,omitempty"`
	RequestPatches  []DryRunPatch `json:"request_patches,omitempty"`
}

type DryRunPatch struct {
	Program      string `json:"program"`
	Action       string `json:"action"`
	Target       string `json:"target"`
	InsertIndex  int    `json:"insert_index"`
	Role         string `json:"role"`
	Position     string `json:"position"`
	ContentHash  string `json:"content_hash"`
	ContentChars int    `json:"content_chars"`
	Reason       string `json:"reason,omitempty"`
}

type Skipped struct {
	Program string `json:"program"`
	Action  string `json:"action"`
	Reason  string `json:"reason"`
}

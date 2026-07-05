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

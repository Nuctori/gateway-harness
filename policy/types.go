package policy

import "encoding/json"

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
	ModelMatches          string `json:"model_matches,omitempty"`
	EstimatedTokensGT     int    `json:"estimated_tokens_gt,omitempty"`
	ContextContinuityDrop bool   `json:"context_continuity_drop,omitempty"`
}

type Action struct {
	Action           string   `json:"action"`
	Role             string   `json:"role,omitempty"`
	Position         string   `json:"position,omitempty"`
	Text             string   `json:"text,omitempty"`
	Source           string   `json:"source,omitempty"`
	LedgerRef        string   `json:"ledger_ref,omitempty"`
	ArtifactRefs     []string `json:"artifact_refs,omitempty"`
	Strategy         string   `json:"strategy,omitempty"`
	KeepLastMessages *int     `json:"keep_last_messages,omitempty"`
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
	Hook                  string
	Model                 string
	EstimatedTokens       int
	ContextContinuityDrop bool
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
	Program      string   `json:"program"`
	Action       string   `json:"action"`
	Target       string   `json:"target"`
	InsertIndex  int      `json:"insert_index"`
	Role         string   `json:"role"`
	Position     string   `json:"position"`
	Source       string   `json:"source,omitempty"`
	LedgerRef    string   `json:"ledger_ref,omitempty"`
	ArtifactRefs []string `json:"artifact_refs,omitempty"`
	ContentHash  string   `json:"content_hash"`
	ContentChars int      `json:"content_chars"`
	Reason       string   `json:"reason,omitempty"`
}

type Skipped struct {
	Program string `json:"program"`
	Action  string `json:"action"`
	Reason  string `json:"reason"`
}

type ApplyOptions struct {
	Hook                  string
	Model                 string
	EstimatedTokens       int
	ContextContinuityDrop bool
}

type ApplyResult struct {
	Hook            string          `json:"hook"`
	Model           string          `json:"model"`
	EstimatedTokens int             `json:"estimated_tokens,omitempty"`
	MatchedPrograms []string        `json:"matched_programs,omitempty"`
	AppliedActions  []string        `json:"applied_actions,omitempty"`
	SkippedActions  []Skipped       `json:"skipped_actions,omitempty"`
	Trace           ApplyTrace      `json:"trace"`
	Request         json.RawMessage `json:"-"`
}

type ApplyTrace struct {
	Operations []TraceOperation `json:"operations,omitempty"`
	Summary    TraceSummary     `json:"summary"`
}

type TraceOperation struct {
	Program      string   `json:"program"`
	Op           string   `json:"op"`
	Action       string   `json:"action"`
	Target       string   `json:"target"`
	InsertIndex  int      `json:"insert_index"`
	Role         string   `json:"role"`
	Source       string   `json:"source,omitempty"`
	LedgerRef    string   `json:"ledger_ref,omitempty"`
	ArtifactRefs []string `json:"artifact_refs,omitempty"`
	ContentHash  string   `json:"content_hash"`
	ContentChars int      `json:"content_chars"`
	Reason       string   `json:"reason,omitempty"`
}

type TraceSummary struct {
	Ops         int    `json:"ops"`
	ContentMode string `json:"content_mode"`
}

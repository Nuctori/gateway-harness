package plan

type Decision struct {
	TraceID       string        `json:"trace_id,omitempty"`
	EventType     string        `json:"event_type,omitempty"`
	MatchedPolicy string        `json:"matched_policy,omitempty"`
	Actions       []Action      `json:"actions,omitempty"`
	ContextPatch  *ContextPatch `json:"context_patch,omitempty"`
	SkippedReason string        `json:"skipped_reason,omitempty"`
}

type Action struct {
	Type      string `json:"type"`
	FromModel string `json:"from_model,omitempty"`
	ToModel   string `json:"to_model,omitempty"`
	Text      string `json:"text,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

type ContextPatch struct {
	Operations []PatchOperation `json:"operations"`
	Summary    PatchSummary     `json:"summary"`
}

type PatchOperation struct {
	Op                 string   `json:"op"`
	Target             string   `json:"target,omitempty"`
	Role               string   `json:"role,omitempty"`
	Position           string   `json:"position,omitempty"`
	Content            string   `json:"content,omitempty"`
	ContentHash        string   `json:"content_hash,omitempty"`
	Strategy           string   `json:"strategy,omitempty"`
	KeepLastMessages   int      `json:"keep_last_messages,omitempty"`
	PreserveRoles      []string `json:"preserve_roles,omitempty"`
	MaxEstimatedTokens int64    `json:"max_estimated_tokens,omitempty"`
	Reason             string   `json:"reason,omitempty"`
}

type PatchSummary struct {
	Ops                    int    `json:"ops"`
	AddedEstimatedTokens   int64  `json:"added_estimated_tokens"`
	RemovedEstimatedTokens int64  `json:"removed_estimated_tokens"`
	ContentMode            string `json:"content_mode"`
}

func (op PatchOperation) Redacted() PatchOperation {
	return PatchOperation{
		Op:                 op.Op,
		Target:             op.Target,
		Role:               op.Role,
		Position:           op.Position,
		ContentHash:        op.ContentHash,
		Strategy:           op.Strategy,
		KeepLastMessages:   op.KeepLastMessages,
		PreserveRoles:      op.PreserveRoles,
		MaxEstimatedTokens: op.MaxEstimatedTokens,
		Reason:             op.Reason,
	}
}

package trace

import "github.com/nuctori/gateway-harness/core/plan"

type Trace struct {
	TraceID   string           `json:"trace_id,omitempty"`
	RequestID string           `json:"request_id,omitempty"`
	Events    []EventRecord    `json:"events,omitempty"`
	Decisions []DecisionRecord `json:"decisions,omitempty"`
	Outcomes  []Outcome        `json:"outcomes,omitempty"`
}

type EventRecord struct {
	Type      string `json:"type"`
	Model     string `json:"model,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
}

type DecisionRecord struct {
	Policy         string             `json:"policy,omitempty"`
	Action         string             `json:"action,omitempty"`
	FromModel      string             `json:"from_model,omitempty"`
	ToModel        string             `json:"to_model,omitempty"`
	Reason         string             `json:"reason,omitempty"`
	MatchedProgram string             `json:"matched_program,omitempty"`
	Conditions     []ConditionRecord  `json:"conditions,omitempty"`
	PatchSummary   *plan.PatchSummary `json:"patch_summary,omitempty"`
}

type ConditionRecord struct {
	Expr   string `json:"expr"`
	Result bool   `json:"result"`
}

type Outcome struct {
	Type      string `json:"type"`
	FromModel string `json:"from_model,omitempty"`
	ToModel   string `json:"to_model,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

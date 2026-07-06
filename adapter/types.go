package adapter

import (
	"github.com/Nuctori/gateway-harness/ledger"
	"github.com/Nuctori/gateway-harness/steward"
)

type Manifest struct {
	Version       string   `json:"version,omitempty"`
	Adapter       string   `json:"adapter"`
	Hooks         []string `json:"hooks"`
	Actions       []string `json:"actions"`
	RequestShapes []string `json:"request_shapes,omitempty"`
	Guards        []string `json:"guards,omitempty"`
}

type Summary struct {
	Adapter       string
	Hooks         int
	Actions       int
	RequestShapes int
	Guards        int
}

type GoalGateRunner struct {
	Command string   `json:"command"`
	Workdir string   `json:"workdir,omitempty"`
	Args    []string `json:"args,omitempty"`
}

type GoalGateConfig struct {
	Enabled             bool           `json:"enabled"`
	Hook                string         `json:"hook,omitempty"`
	Runner              GoalGateRunner `json:"runner,omitempty"`
	AllowedInputs       []string       `json:"allowed_inputs,omitempty"`
	AllowedActions      []string       `json:"allowed_actions,omitempty"`
	MaxContinueAttempts int            `json:"max_continue_attempts,omitempty"`
	CooldownSeconds     int            `json:"cooldown_seconds,omitempty"`
}

type GoalGateRequest struct {
	Config  GoalGateConfig             `json:"config"`
	Spec    steward.Spec               `json:"spec"`
	Event   steward.Event              `json:"event"`
	Audit   steward.GoalGateAuditInput `json:"audit"`
	NowUnix int64                      `json:"now_unix,omitempty"`
}

type GoalGateFailure struct {
	Stage          string `json:"stage"`
	Code           string `json:"code"`
	Message        string `json:"message"`
	RunnerCommand  string `json:"runner_command,omitempty"`
	RunnerWorkdir  string `json:"runner_workdir,omitempty"`
	RunnerArgsHash string `json:"runner_args_hash,omitempty"`
}

type GoalGateResult struct {
	Enabled       bool                           `json:"enabled"`
	Triggered     bool                           `json:"triggered"`
	SkippedReason string                         `json:"skipped_reason,omitempty"`
	Sidecar       *steward.GoalGateSidecarResult `json:"sidecar,omitempty"`
	AppendRecord  *ledger.AppendRecord           `json:"append_record,omitempty"`
	Failure       *GoalGateFailure               `json:"failure,omitempty"`
}

type GoalGateExecutionError struct {
	Result GoalGateResult
	Err    error
}

func (e *GoalGateExecutionError) Error() string {
	if e == nil {
		return ""
	}
	if e.Result.Failure != nil && e.Result.Failure.Message != "" {
		return e.Result.Failure.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return "goal gate execution failed"
}

func (e *GoalGateExecutionError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

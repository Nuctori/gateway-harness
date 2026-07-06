package adapter

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Nuctori/gateway-harness/ledger"
	"github.com/Nuctori/gateway-harness/steward"
)

func TestValidateGoalGateConfigDisabledIsNoop(t *testing.T) {
	if err := ValidateGoalGateConfig(GoalGateConfig{}); err != nil {
		t.Fatalf("disabled goal gate should validate: %v", err)
	}
}

func TestValidateGoalGateConfigEnabledRequiresRunner(t *testing.T) {
	err := ValidateGoalGateConfig(GoalGateConfig{
		Enabled:        true,
		Hook:           steward.GoalBeforeCompleteHook,
		AllowedInputs:  []string{"goal_state"},
		AllowedActions: []string{"goal.approve_complete"},
	})
	if err == nil {
		t.Fatal("expected runner requirement")
	}
}

func TestValidateGoalGateConfigRequiresGoalStateAndTerminalActions(t *testing.T) {
	err := ValidateGoalGateConfig(GoalGateConfig{
		Enabled:        true,
		Hook:           steward.GoalBeforeCompleteHook,
		Runner:         GoalGateRunner{Command: "python"},
		AllowedInputs:  []string{"work_summary"},
		AllowedActions: []string{"goal.approve_complete", "goal.reject_complete"},
	})
	if err == nil {
		t.Fatal("expected goal_state / terminal action requirement")
	}
}

func TestBuildEffectiveGoalGateSpecRejectsConfigActionOutsideSpec(t *testing.T) {
	spec, err := steward.Decode(strings.NewReader(goalGateSpecJSON))
	if err != nil {
		t.Fatalf("decode spec: %v", err)
	}
	_, err = BuildEffectiveGoalGateSpec(spec, GoalGateConfig{
		Enabled:        true,
		Hook:           steward.GoalBeforeCompleteHook,
		Runner:         GoalGateRunner{Command: "python"},
		AllowedInputs:  []string{"goal_state", "work_summary"},
		AllowedActions: []string{"goal.approve_complete", "goal.reject_complete", "goal.request_continue", "context.inject"},
	})
	if err == nil {
		t.Fatal("expected config action outside spec error")
	}
}

func TestExecuteGoalGateDisabledSkipsWithoutRunnerCall(t *testing.T) {
	result, err := ExecuteGoalGate(context.Background(), GoalGateRequest{Config: GoalGateConfig{Enabled: false}})
	if err != nil {
		t.Fatalf("execute disabled goal gate: %v", err)
	}
	if result.Enabled || result.Triggered || result.SkippedReason != GoalGateSkipDisabled {
		t.Fatalf("unexpected disabled result: %+v", result)
	}
}

func TestExecuteGoalGateEnabledInvokesRunnerAndReturnsAppendRecord(t *testing.T) {
	proposal := strings.TrimSpace(`{
  "version": "0.1",
  "id": "proposal_goal_review_approve_001",
  "steward": "goal-completion-reviewer",
  "hook": "goal.before_complete",
  "outputs": [
    {
      "action": "goal.approve_complete",
      "reason": "all checks passed"
    }
  ]
}`)
	t.Setenv("GATEWAY_HARNESS_TEST_GOAL_PROPOSAL", proposal)
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	spec, err := steward.Decode(strings.NewReader(goalGateSpecJSON))
	if err != nil {
		t.Fatalf("decode spec: %v", err)
	}
	event, _, err := steward.DecodeEvent(strings.NewReader(goalGateEventJSON))
	if err != nil {
		t.Fatalf("decode event: %v", err)
	}
	result, err := ExecuteGoalGate(context.Background(), GoalGateRequest{
		Config: GoalGateConfig{
			Enabled:             true,
			Hook:                steward.GoalBeforeCompleteHook,
			Runner:              GoalGateRunner{Command: "go", Args: []string{"run", "../cmd/gateway-harness/testdata/goalreviewhelper"}},
			AllowedInputs:       []string{"goal_state", "work_summary", "test_results", "blockers", "recent_trace", "changed_files", "verification_summary", "user_goal"},
			AllowedActions:      []string{"goal.approve_complete", "goal.reject_complete", "goal.request_continue", "diagnosis.note.create", "ledger.artifact.create"},
			MaxContinueAttempts: 3,
			CooldownSeconds:     60,
		},
		Spec:  spec,
		Event: event,
		Audit: steward.GoalGateAuditInput{
			Project:       ledger.AppendProject{ID: "project_gateway_harness"},
			Session:       ledger.AppendSession{ID: "session_goal_gate", StartedAt: "2026-07-06T04:00:00Z"},
			EventID:       "evt_goal_gate_adapter_1",
			At:            time.Unix(1700000000, 0),
			PolicyVersion: "0.2",
			TraceHash:     "sha256:6666666666666666666666666666666666666666666666666666666666666666",
			Model:         "external-agent",
		},
		NowUnix: 1700000000,
	})
	if err != nil {
		t.Fatalf("execute enabled goal gate: %v", err)
	}
	_ = wd
	if !result.Enabled || !result.Triggered || result.Sidecar == nil || result.AppendRecord == nil {
		t.Fatalf("unexpected goal gate result: %+v", result)
	}
	if !result.Sidecar.Outcome.AllowComplete {
		t.Fatalf("expected approval outcome: %+v", result.Sidecar.Outcome)
	}
	if result.AppendRecord.Event.Action != "goal.approve_complete" {
		t.Fatalf("unexpected append record: %+v", result.AppendRecord.Event)
	}
}

func TestExecuteGoalGateRejectsEventInputOutsideRuntimeConfig(t *testing.T) {
	spec, err := steward.Decode(strings.NewReader(goalGateSpecJSON))
	if err != nil {
		t.Fatalf("decode spec: %v", err)
	}
	event, _, err := steward.DecodeEvent(strings.NewReader(goalGateEventJSON))
	if err != nil {
		t.Fatalf("decode event: %v", err)
	}
	var inputs map[string]json.RawMessage
	if err := json.Unmarshal(event.Inputs, &inputs); err != nil {
		t.Fatalf("decode event inputs: %v", err)
	}
	inputs["user_goal"] = json.RawMessage(`"extra"`)
	encodedInputs, err := json.Marshal(inputs)
	if err != nil {
		t.Fatalf("encode event inputs: %v", err)
	}
	event.Inputs = encodedInputs
	_, err = ExecuteGoalGate(context.Background(), GoalGateRequest{
		Config: GoalGateConfig{
			Enabled:             true,
			Hook:                steward.GoalBeforeCompleteHook,
			Runner:              GoalGateRunner{Command: "go", Args: []string{"run", "../cmd/gateway-harness/testdata/goalreviewhelper"}},
			AllowedInputs:       []string{"goal_state", "work_summary", "test_results", "blockers", "recent_trace", "changed_files", "verification_summary"},
			AllowedActions:      []string{"goal.approve_complete", "goal.reject_complete", "goal.request_continue", "diagnosis.note.create", "ledger.artifact.create"},
			MaxContinueAttempts: 3,
			CooldownSeconds:     60,
		},
		Spec:  spec,
		Event: event,
		Audit: steward.GoalGateAuditInput{Project: ledger.AppendProject{ID: "project_gateway_harness"}, Session: ledger.AppendSession{ID: "session_goal_gate", StartedAt: "2026-07-06T04:00:00Z"}, EventID: "evt_goal_gate_adapter_2", At: time.Unix(1700000000, 0)},
	})
	if err == nil {
		t.Fatal("expected runtime config input restriction")
	}
}

func TestExecuteGoalGateRunnerFailureReturnsAuditableErrorResult(t *testing.T) {
	spec, err := steward.Decode(strings.NewReader(goalGateSpecJSON))
	if err != nil {
		t.Fatalf("decode spec: %v", err)
	}
	event, _, err := steward.DecodeEvent(strings.NewReader(goalGateEventJSON))
	if err != nil {
		t.Fatalf("decode event: %v", err)
	}
	result, err := ExecuteGoalGate(context.Background(), GoalGateRequest{
		Config: GoalGateConfig{
			Enabled:             true,
			Hook:                steward.GoalBeforeCompleteHook,
			Runner:              GoalGateRunner{Command: "go", Args: []string{"run", "../cmd/gateway-harness/testdata/does-not-exist"}},
			AllowedInputs:       []string{"goal_state", "work_summary", "test_results", "blockers", "recent_trace", "changed_files", "verification_summary", "user_goal"},
			AllowedActions:      []string{"goal.approve_complete", "goal.reject_complete", "goal.request_continue", "diagnosis.note.create", "ledger.artifact.create"},
			MaxContinueAttempts: 3,
			CooldownSeconds:     60,
		},
		Spec:  spec,
		Event: event,
		Audit: steward.GoalGateAuditInput{
			Project:       ledger.AppendProject{ID: "project_gateway_harness"},
			Session:       ledger.AppendSession{ID: "session_goal_gate", StartedAt: "2026-07-06T04:00:00Z"},
			EventID:       "evt_goal_gate_adapter_runner_fail",
			At:            time.Unix(1700000000, 0),
			PolicyVersion: "0.2",
			TraceHash:     "sha256:7777777777777777777777777777777777777777777777777777777777777777",
			Model:         "external-agent",
		},
		NowUnix: 1700000000,
	})
	if err == nil {
		t.Fatal("expected runner failure")
	}
	var execErr *GoalGateExecutionError
	if !errors.As(err, &execErr) {
		t.Fatalf("expected GoalGateExecutionError, got %T", err)
	}
	if execErr.Result.Failure == nil {
		t.Fatalf("expected structured failure result: %+v", execErr.Result)
	}
	if execErr.Result.Failure.Code != "goal_gate_runner_failed" || execErr.Result.Failure.Stage != "runner" {
		t.Fatalf("unexpected failure details: %+v", execErr.Result.Failure)
	}
	if execErr.Result.AppendRecord == nil {
		t.Fatalf("expected append record on failure: %+v", execErr.Result)
	}
	if execErr.Result.AppendRecord.Event.Type != "error" || execErr.Result.AppendRecord.Event.ErrorCode != "goal_gate_runner_failed" {
		t.Fatalf("unexpected failure append record: %+v", execErr.Result.AppendRecord.Event)
	}
	if execErr.Result.AppendRecord.Event.Action != "goal.review.failed" {
		t.Fatalf("unexpected failure action: %+v", execErr.Result.AppendRecord.Event)
	}
	if result.Failure == nil || result.Failure.Code != "goal_gate_runner_failed" {
		t.Fatalf("expected returned result to include failure: %+v", result)
	}
}

func TestExecuteGoalGateInvalidProposalReturnsStructuredProposalFailure(t *testing.T) {
	proposal := strings.TrimSpace(`{
  "version": "0.1",
  "id": "proposal_goal_review_invalid_001",
  "steward": "goal-completion-reviewer",
  "hook": "goal.before_complete",
  "outputs": [
    {
      "action": "context.truncate",
      "reason": "not allowed here"
    }
  ]
}`)
	t.Setenv("GATEWAY_HARNESS_TEST_GOAL_PROPOSAL", proposal)
	spec, err := steward.Decode(strings.NewReader(goalGateSpecJSON))
	if err != nil {
		t.Fatalf("decode spec: %v", err)
	}
	event, _, err := steward.DecodeEvent(strings.NewReader(goalGateEventJSON))
	if err != nil {
		t.Fatalf("decode event: %v", err)
	}
	result, err := ExecuteGoalGate(context.Background(), GoalGateRequest{
		Config: GoalGateConfig{
			Enabled:             true,
			Hook:                steward.GoalBeforeCompleteHook,
			Runner:              GoalGateRunner{Command: "go", Args: []string{"run", "../cmd/gateway-harness/testdata/goalreviewhelper"}},
			AllowedInputs:       []string{"goal_state", "work_summary", "test_results", "blockers", "recent_trace", "changed_files", "verification_summary", "user_goal"},
			AllowedActions:      []string{"goal.approve_complete", "goal.reject_complete", "goal.request_continue", "diagnosis.note.create", "ledger.artifact.create"},
			MaxContinueAttempts: 3,
			CooldownSeconds:     60,
		},
		Spec:  spec,
		Event: event,
		Audit: steward.GoalGateAuditInput{
			Project:       ledger.AppendProject{ID: "project_gateway_harness"},
			Session:       ledger.AppendSession{ID: "session_goal_gate", StartedAt: "2026-07-06T04:00:00Z"},
			EventID:       "evt_goal_gate_adapter_invalid_proposal",
			At:            time.Unix(1700000000, 0),
			PolicyVersion: "0.2",
			TraceHash:     "sha256:8888888888888888888888888888888888888888888888888888888888888888",
			Model:         "external-agent",
		},
		NowUnix: 1700000000,
	})
	if err == nil {
		t.Fatal("expected invalid proposal failure")
	}
	var execErr *GoalGateExecutionError
	if !errors.As(err, &execErr) {
		t.Fatalf("expected GoalGateExecutionError, got %T", err)
	}
	if execErr.Result.Failure == nil {
		t.Fatalf("expected structured failure result: %+v", execErr.Result)
	}
	if execErr.Result.Failure.Code != "goal_gate_proposal_invalid" || execErr.Result.Failure.Stage != "proposal" {
		t.Fatalf("unexpected failure details: %+v", execErr.Result.Failure)
	}
	if result.Failure == nil || result.Failure.Code != "goal_gate_proposal_invalid" {
		t.Fatalf("expected proposal failure in returned result: %+v", result)
	}
}

const goalGateSpecJSON = `{
  "version": "0.1",
  "name": "goal-completion-reviewer",
  "steward_model": "external-agent",
  "hooks": ["goal.before_complete"],
  "inputs": [
    "goal_state",
    "work_summary",
    "test_results",
    "blockers",
    "recent_trace",
    "changed_files",
    "verification_summary",
    "user_goal"
  ],
  "allowed_actions": [
    "goal.approve_complete",
    "goal.reject_complete",
    "goal.request_continue",
    "diagnosis.note.create",
    "ledger.artifact.create"
  ],
  "artifact_types": ["trace"],
  "required_guards": [
    "explicit_invocation_only",
    "structured_output_only",
    "validate_output_actions",
    "redacted_input_only",
    "artifact_hash_required"
  ]
}`

const goalGateEventJSON = `{
  "hook": "goal.before_complete",
  "redacted": true,
  "inputs": {
    "goal_state": {
      "status": "pending_complete",
      "attempt": 1,
      "max_continue_attempts": 3,
      "cooldown_seconds": 60
    },
    "work_summary": "Focused validation for Goal Gate contract and dry-run behavior passed locally.",
    "test_results": [{"command": "go test ./steward -count=1", "status": "passed"}],
    "blockers": ["end-to-end executor interception is not wired yet"],
    "recent_trace": [{"event": "validate-steward", "status": "passed"}],
    "changed_files": ["steward/validate.go"],
    "verification_summary": "Contract validation passed.",
    "user_goal": "Finish Goal Gate safely without hidden behavior."
  }
}`

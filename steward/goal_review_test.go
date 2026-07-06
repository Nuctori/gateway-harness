package steward

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Nuctori/gateway-harness/ledger"
)

func TestEvaluateGoalProposalApprovesCompletion(t *testing.T) {
	s, err := Decode(strings.NewReader(goalGateStewardJSON))
	if err != nil {
		t.Fatalf("decode spec: %v", err)
	}
	e, _, err := DecodeEvent(strings.NewReader(goalGateEventJSON))
	if err != nil {
		t.Fatalf("decode event: %v", err)
	}
	p, err := DecodeProposal(strings.NewReader(goalGateApproveProposalJSON))
	if err != nil {
		t.Fatalf("decode proposal: %v", err)
	}
	result, err := EvaluateGoalProposal(s, e, p, time.Unix(1700000000, 0))
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if result.Decision != GoalDecisionApprove || result.ContinueAllowed {
		t.Fatalf("unexpected approval result: %+v", result)
	}
}

func TestEvaluateGoalProposalRejectsAndRequestsContinuation(t *testing.T) {
	s, err := Decode(strings.NewReader(goalGateStewardJSON))
	if err != nil {
		t.Fatalf("decode spec: %v", err)
	}
	e, _, err := DecodeEvent(strings.NewReader(goalGateEventJSON))
	if err != nil {
		t.Fatalf("decode event: %v", err)
	}
	p, err := DecodeProposal(strings.NewReader(goalGateRejectProposalJSON))
	if err != nil {
		t.Fatalf("decode proposal: %v", err)
	}
	result, err := EvaluateGoalProposal(s, e, p, time.Unix(1700000000, 0))
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if result.Decision != GoalDecisionReject || !result.ContinueAllowed || result.ContinueInstruction == "" {
		t.Fatalf("unexpected reject result: %+v", result)
	}
	if result.RejectionReasonHash == "" {
		t.Fatalf("expected rejection hash: %+v", result)
	}
}

func TestEvaluateGoalProposalExposesContinuationPatches(t *testing.T) {
	s, err := Decode(strings.NewReader(strings.Replace(goalGateStewardJSON, `"ledger.artifact.create"]`, `"ledger.artifact.create", "context.inject"]`, 1)))
	if err != nil {
		t.Fatalf("decode spec: %v", err)
	}
	e, _, err := DecodeEvent(strings.NewReader(goalGateEventJSON))
	if err != nil {
		t.Fatalf("decode event: %v", err)
	}
	raw := `{
  "version": "0.1",
  "id": "proposal_goal_review_reject_patch_001",
  "steward": "goal-completion-reviewer",
  "hook": "goal.before_complete",
  "outputs": [
    {
      "action": "goal.reject_complete",
      "reason": "deployment verification is still missing"
    },
    {
      "action": "goal.request_continue",
      "reason": "continue with deployment verification",
      "instruction": "Deploy the image and run the smoke test."
    },
    {
      "action": "context.inject",
      "role": "system",
      "position": "before_messages",
      "text": "Reviewer note: verify deployment before claiming complete."
    }
  ]
}`
	p, err := DecodeProposal(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("decode proposal: %v", err)
	}
	result, err := EvaluateGoalProposal(s, e, p, time.Unix(1700000000, 0))
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if len(result.ContinuationPatches) != 1 {
		t.Fatalf("expected continuation patch: %+v", result)
	}
	patch := result.ContinuationPatches[0]
	if patch.Action != "context.inject" || patch.Target != "continuation" || patch.Role != "system" {
		t.Fatalf("unexpected continuation patch: %+v", patch)
	}
}

func TestEvaluateGoalProposalRejectsApprovalWithContinuation(t *testing.T) {
	s, err := Decode(strings.NewReader(goalGateStewardJSON))
	if err != nil {
		t.Fatalf("decode spec: %v", err)
	}
	e, _, err := DecodeEvent(strings.NewReader(goalGateEventJSON))
	if err != nil {
		t.Fatalf("decode event: %v", err)
	}
	raw := `{
  "version": "0.1",
  "id": "proposal_goal_review_bad_001",
  "steward": "goal-completion-reviewer",
  "hook": "goal.before_complete",
  "outputs": [
    {"action": "goal.approve_complete", "reason": "done"},
    {"action": "goal.request_continue", "instruction": "keep going"}
  ]
}`
	p, err := DecodeProposal(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("decode proposal: %v", err)
	}
	if _, err := EvaluateGoalProposal(s, e, p, time.Unix(1700000000, 0)); err == nil {
		t.Fatal("expected conflicting approval/continuation error")
	}
}

func TestEvaluateGoalProposalBlocksRetryAtMaxAttempts(t *testing.T) {
	s, err := Decode(strings.NewReader(goalGateStewardJSON))
	if err != nil {
		t.Fatalf("decode spec: %v", err)
	}
	rawEvent := strings.Replace(goalGateEventJSON, `"attempt": 1`, `"attempt": 3`, 1)
	e, _, err := DecodeEvent(strings.NewReader(rawEvent))
	if err != nil {
		t.Fatalf("decode event: %v", err)
	}
	p, err := DecodeProposal(strings.NewReader(goalGateRejectProposalJSON))
	if err != nil {
		t.Fatalf("decode proposal: %v", err)
	}
	result, err := EvaluateGoalProposal(s, e, p, time.Unix(1700000000, 0))
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if result.ContinueAllowed || !strings.Contains(result.ContinueBlockedReason, "max_continue_attempts") {
		t.Fatalf("expected max-attempt block: %+v", result)
	}
}

func TestEvaluateGoalProposalBlocksDuplicateReason(t *testing.T) {
	s, err := Decode(strings.NewReader(goalGateStewardJSON))
	if err != nil {
		t.Fatalf("decode spec: %v", err)
	}
	dupHash := hashReason("Focused tests passed, but deployment verification is missing.")
	rawEvent := strings.Replace(goalGateEventJSON, `"max_continue_attempts": 3}`, fmt.Sprintf(`"max_continue_attempts": 3, "last_rejection_reason_hash": %q}`, dupHash), 1)
	e, _, err := DecodeEvent(strings.NewReader(rawEvent))
	if err != nil {
		t.Fatalf("decode event: %v", err)
	}
	p, err := DecodeProposal(strings.NewReader(goalGateRejectProposalJSON))
	if err != nil {
		t.Fatalf("decode proposal: %v", err)
	}
	result, err := EvaluateGoalProposal(s, e, p, time.Unix(1700000000, 0))
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if result.ContinueAllowed || !result.DuplicateReason {
		t.Fatalf("expected duplicate-reason block: %+v", result)
	}
}

func TestEvaluateGoalProposalBlocksCooldown(t *testing.T) {
	s, err := Decode(strings.NewReader(goalGateStewardJSON))
	if err != nil {
		t.Fatalf("decode spec: %v", err)
	}
	rawEvent := strings.Replace(goalGateEventJSON, `"max_continue_attempts": 3}`, `"max_continue_attempts": 3, "cooldown_seconds": 60, "last_rejection_at_unix": 1700000000, "last_rejection_reason_hash": "sha256:2222222222222222222222222222222222222222222222222222222222222222"}`, 1)
	e, _, err := DecodeEvent(strings.NewReader(rawEvent))
	if err != nil {
		t.Fatalf("decode event: %v", err)
	}
	p, err := DecodeProposal(strings.NewReader(goalGateRejectProposalJSON))
	if err != nil {
		t.Fatalf("decode proposal: %v", err)
	}
	result, err := EvaluateGoalProposal(s, e, p, time.Unix(1700000030, 0))
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if result.ContinueAllowed || !result.CooldownActive || result.CooldownRemainingSeconds <= 0 {
		t.Fatalf("expected cooldown block: %+v", result)
	}
}

func TestReviewGoalCompletionInvokesExternalAgent(t *testing.T) {
	t.Setenv("GATEWAY_HARNESS_TEST_GOAL_PROPOSAL", goalGateApproveProposalJSON)
	s, err := Decode(strings.NewReader(goalGateStewardJSON))
	if err != nil {
		t.Fatalf("decode spec: %v", err)
	}
	e, _, err := DecodeEvent(strings.NewReader(goalGateEventJSON))
	if err != nil {
		t.Fatalf("decode event: %v", err)
	}
	result, err := ReviewGoalCompletion(context.Background(), s, e, time.Unix(1700000000, 0), os.Args[0], "-test.run=TestGoalReviewHelperProcess", "--", "--goal-review-helper")
	if err != nil {
		t.Fatalf("review goal completion: %v", err)
	}
	if result.Decision != GoalDecisionApprove {
		t.Fatalf("unexpected runner result: %+v", result)
	}
}

func TestReviewGoalCompletionReturnsRunnerFailure(t *testing.T) {
	t.Setenv("GATEWAY_HARNESS_TEST_GOAL_PROPOSAL", "__exit_with_error__")
	s, err := Decode(strings.NewReader(goalGateStewardJSON))
	if err != nil {
		t.Fatalf("decode spec: %v", err)
	}
	e, _, err := DecodeEvent(strings.NewReader(goalGateEventJSON))
	if err != nil {
		t.Fatalf("decode event: %v", err)
	}
	if _, err := ReviewGoalCompletion(context.Background(), s, e, time.Unix(1700000000, 0), os.Args[0], "-test.run=TestGoalReviewHelperProcess", "--", "--goal-review-helper"); err == nil {
		t.Fatal("expected runner failure")
	}
}

func TestApplyGoalReviewResultApprovesCompletion(t *testing.T) {
	state := GoalState{Status: "pending_complete", Attempt: 1, MaxContinueAttempts: 3, CooldownSeconds: 60}
	result := GoalReviewResult{
		ProposalID:      "proposal_goal_review_approve_001",
		Steward:         "goal-completion-reviewer",
		Hook:            GoalBeforeCompleteHook,
		Decision:        GoalDecisionApprove,
		Reason:          "done",
		AppliedActions:  []string{"goal.approve_complete"},
		ContinueAllowed: false,
	}
	outcome := ApplyGoalReviewResult(result, state, time.Unix(1700000000, 0))
	if !outcome.AllowComplete || outcome.ContinueWork || outcome.NextGoalState.Status != "complete" {
		t.Fatalf("unexpected approve outcome: %+v", outcome)
	}
	if outcome.LedgerEventAction != "goal.approve_complete" {
		t.Fatalf("unexpected ledger action: %+v", outcome)
	}
}

func TestApplyGoalReviewResultRequestsContinuation(t *testing.T) {
	state := GoalState{Status: "pending_complete", Attempt: 1, MaxContinueAttempts: 3, CooldownSeconds: 60}
	result := GoalReviewResult{
		ProposalID:          "proposal_goal_review_001",
		Steward:             "goal-completion-reviewer",
		Hook:                GoalBeforeCompleteHook,
		Decision:            GoalDecisionReject,
		Reason:              "still missing verification",
		ContinueInstruction: "deploy and smoke test",
		ContinueAllowed:     true,
		NextAttempt:         2,
		RejectionReasonHash: hashReason("still missing verification"),
		AppliedActions:      []string{"goal.reject_complete", "goal.request_continue"},
	}
	outcome := ApplyGoalReviewResult(result, state, time.Unix(1700000000, 0))
	if outcome.AllowComplete || !outcome.ContinueWork || outcome.NextGoalState.Attempt != 2 {
		t.Fatalf("unexpected continue outcome: %+v", outcome)
	}
	if outcome.NextGoalState.LastRejectionReasonHash == "" || outcome.LedgerEventAction != "goal.request_continue" {
		t.Fatalf("unexpected side effects: %+v", outcome)
	}
}

func TestApplyGoalReviewResultBlocksCompletionWhenContinueForbidden(t *testing.T) {
	state := GoalState{Status: "pending_complete", Attempt: 3, MaxContinueAttempts: 3, CooldownSeconds: 60}
	result := GoalReviewResult{
		ProposalID:            "proposal_goal_review_001",
		Steward:               "goal-completion-reviewer",
		Hook:                  GoalBeforeCompleteHook,
		Decision:              GoalDecisionReject,
		Reason:                "still missing verification",
		ContinueAllowed:       false,
		ContinueBlockedReason: "max_continue_attempts reached (3/3)",
		RejectionReasonHash:   hashReason("still missing verification"),
		AppliedActions:        []string{"goal.reject_complete", "goal.request_continue"},
	}
	outcome := ApplyGoalReviewResult(result, state, time.Unix(1700000000, 0))
	if outcome.AllowComplete || outcome.ContinueWork {
		t.Fatalf("unexpected blocked outcome: %+v", outcome)
	}
	if outcome.LedgerEventType != "error" || outcome.LedgerEventErrorCode != "goal_gate_max_continue_attempts_reached" {
		t.Fatalf("unexpected error mapping: %+v", outcome)
	}
}

func TestBuildGoalGateAppendRecordForContinuation(t *testing.T) {
	outcome := GoalGateOutcome{
		ProposalID:          "proposal_goal_review_001",
		Decision:            GoalDecisionReject,
		AllowComplete:       false,
		ContinueWork:        true,
		ContinueInstruction: "deploy and smoke test",
		Reason:              "still missing verification",
		LedgerEventType:     "harness_action",
		LedgerEventAction:   "goal.request_continue",
		LedgerEventMetadata: map[string]string{
			"proposal_id": "proposal_goal_review_001",
			"decision":    "reject_complete",
		},
		RejectionReasonHash: hashReason("still missing verification"),
	}
	record, err := BuildGoalGateAppendRecord(GoalGateAuditInput{
		Project:       ledger.AppendProject{ID: "project_gateway_harness", Name: "Gateway Harness"},
		Session:       ledger.AppendSession{ID: "session_goal_gate", StartedAt: "2026-07-06T04:00:00Z"},
		EventID:       "evt_goal_gate_1",
		At:            time.Unix(1700000000, 0),
		PolicyVersion: "0.2",
		TraceHash:     "sha256:3333333333333333333333333333333333333333333333333333333333333333",
		Model:         "external-agent",
	}, outcome)
	if err != nil {
		t.Fatalf("build append record: %v", err)
	}
	if record.Event.Type != "harness_action" || record.Event.Action != "goal.request_continue" {
		t.Fatalf("unexpected ledger event: %+v", record.Event)
	}
	if record.Event.Metadata["continue_instruction_hash"] == "" {
		t.Fatalf("expected instruction hash metadata: %+v", record.Event.Metadata)
	}
	if record.Event.Hook != GoalBeforeCompleteHook {
		t.Fatalf("unexpected hook: %+v", record.Event)
	}
}

func TestBuildGoalGateAppendRecordForBlockedError(t *testing.T) {
	outcome := GoalGateOutcome{
		ProposalID:           "proposal_goal_review_001",
		Decision:             GoalDecisionReject,
		AllowComplete:        false,
		ContinueWork:         false,
		BlockedReason:        "max_continue_attempts reached (3/3)",
		LedgerEventType:      "error",
		LedgerEventAction:    "goal.reject_complete",
		LedgerEventErrorCode: "goal_gate_max_continue_attempts_reached",
		LedgerEventMetadata: map[string]string{
			"proposal_id": "proposal_goal_review_001",
			"decision":    "reject_complete",
		},
	}
	record, err := BuildGoalGateAppendRecord(GoalGateAuditInput{
		Project: ledger.AppendProject{ID: "project_gateway_harness"},
		Session: ledger.AppendSession{ID: "session_goal_gate", StartedAt: "2026-07-06T04:00:00Z"},
		EventID: "evt_goal_gate_2",
		At:      time.Unix(1700000000, 0),
	}, outcome)
	if err != nil {
		t.Fatalf("build append record: %v", err)
	}
	if record.Event.Type != "error" || record.Event.ErrorCode != "goal_gate_max_continue_attempts_reached" {
		t.Fatalf("unexpected blocked event: %+v", record.Event)
	}
	if record.Event.Metadata["blocked_reason"] == "" {
		t.Fatalf("expected blocked metadata: %+v", record.Event.Metadata)
	}
}

func TestExecuteGoalGateSidecarReturnsReviewOutcomeAndAppendRecord(t *testing.T) {
	t.Setenv("GATEWAY_HARNESS_TEST_GOAL_PROPOSAL", goalGateApproveProposalJSON)
	s, err := Decode(strings.NewReader(goalGateStewardJSON))
	if err != nil {
		t.Fatalf("decode spec: %v", err)
	}
	e, _, err := DecodeEvent(strings.NewReader(goalGateEventJSON))
	if err != nil {
		t.Fatalf("decode event: %v", err)
	}
	result, err := ExecuteGoalGateSidecar(
		context.Background(),
		s,
		e,
		GoalGateAuditInput{
			Project:       ledger.AppendProject{ID: "project_gateway_harness", Name: "Gateway Harness"},
			Session:       ledger.AppendSession{ID: "session_goal_gate", StartedAt: "2026-07-06T04:00:00Z"},
			EventID:       "evt_goal_gate_sidecar_1",
			At:            time.Unix(1700000000, 0),
			PolicyVersion: "0.2",
			TraceHash:     "sha256:4444444444444444444444444444444444444444444444444444444444444444",
			Model:         "external-agent",
		},
		time.Unix(1700000000, 0),
		os.Args[0], "-test.run=TestGoalReviewHelperProcess", "--", "--goal-review-helper",
	)
	if err != nil {
		t.Fatalf("execute goal gate sidecar: %v", err)
	}
	if result.Review.Decision != GoalDecisionApprove || !result.Outcome.AllowComplete {
		t.Fatalf("unexpected sidecar result: %+v", result)
	}
	if result.AppendRecord.Event.Action != "goal.approve_complete" {
		t.Fatalf("unexpected append record: %+v", result.AppendRecord)
	}
	if result.AppendRecord.Event.Metadata["proposal_id"] == "" {
		t.Fatalf("expected proposal metadata: %+v", result.AppendRecord.Event)
	}
}

func TestGoalReviewHelperProcess(t *testing.T) {
	if !hasArg("--goal-review-helper") {
		t.Skip("helper process only")
	}
	stdin, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if !strings.Contains(string(stdin), GoalBeforeCompleteHook) {
		fmt.Fprintln(os.Stderr, "goal hook missing from helper stdin")
		os.Exit(2)
	}
	proposal := os.Getenv("GATEWAY_HARNESS_TEST_GOAL_PROPOSAL")
	if proposal == "__exit_with_error__" {
		fmt.Fprintln(os.Stderr, "forced helper failure")
		os.Exit(2)
	}
	_, _ = fmt.Fprint(os.Stdout, proposal)
	os.Exit(0)
}

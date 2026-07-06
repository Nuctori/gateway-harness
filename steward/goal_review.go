package steward

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/Nuctori/gateway-harness/ledger"
)

const GoalBeforeCompleteHook = "goal.before_complete"

type GoalState struct {
	Status                  string `json:"status,omitempty"`
	Attempt                 int    `json:"attempt,omitempty"`
	MaxContinueAttempts     int    `json:"max_continue_attempts,omitempty"`
	CooldownSeconds         int    `json:"cooldown_seconds,omitempty"`
	LastRejectionReasonHash string `json:"last_rejection_reason_hash,omitempty"`
	LastRejectionAtUnix     int64  `json:"last_rejection_at_unix,omitempty"`
}

type GoalReviewDecision string

const (
	GoalDecisionApprove GoalReviewDecision = "approve_complete"
	GoalDecisionReject  GoalReviewDecision = "reject_complete"
)

type GoalReviewResult struct {
	ProposalID               string             `json:"proposal_id"`
	Steward                  string             `json:"steward"`
	Hook                     string             `json:"hook"`
	Decision                 GoalReviewDecision `json:"decision"`
	Reason                   string             `json:"reason,omitempty"`
	ContinueInstruction      string             `json:"continue_instruction,omitempty"`
	ContinueAllowed          bool               `json:"continue_allowed"`
	ContinueBlockedReason    string             `json:"continue_blocked_reason,omitempty"`
	DuplicateReason          bool               `json:"duplicate_reason,omitempty"`
	CooldownActive           bool               `json:"cooldown_active,omitempty"`
	CooldownRemainingSeconds int                `json:"cooldown_remaining_seconds,omitempty"`
	Attempt                  int                `json:"attempt,omitempty"`
	NextAttempt              int                `json:"next_attempt,omitempty"`
	MaxContinueAttempts      int                `json:"max_continue_attempts,omitempty"`
	CooldownSeconds          int                `json:"cooldown_seconds,omitempty"`
	RejectionReasonHash      string             `json:"rejection_reason_hash,omitempty"`
	AppliedActions           []string           `json:"applied_actions"`
	Artifacts                []DryRunRef        `json:"artifacts,omitempty"`
	Diagnostics              []DryRunRef        `json:"diagnostics,omitempty"`
	ContinuationPatches      []DryRunPatch      `json:"continuation_patches,omitempty"`
}

type GoalGateOutcome struct {
	ProposalID            string             `json:"proposal_id"`
	Decision              GoalReviewDecision `json:"decision"`
	AllowComplete         bool               `json:"allow_complete"`
	ContinueWork          bool               `json:"continue_work"`
	ContinueInstruction   string             `json:"continue_instruction,omitempty"`
	Reason                string             `json:"reason,omitempty"`
	BlockedReason         string             `json:"blocked_reason,omitempty"`
	NextGoalState         GoalState          `json:"next_goal_state"`
	LedgerEventType       string             `json:"ledger_event_type"`
	LedgerEventAction     string             `json:"ledger_event_action"`
	LedgerEventErrorCode  string             `json:"ledger_event_error_code,omitempty"`
	LedgerEventMetadata   map[string]string  `json:"ledger_event_metadata,omitempty"`
	RejectionReasonHash   string             `json:"rejection_reason_hash,omitempty"`
	CooldownRemainingSecs int                `json:"cooldown_remaining_seconds,omitempty"`
}

type GoalGateAuditInput struct {
	Project       ledger.AppendProject `json:"project"`
	Session       ledger.AppendSession `json:"session"`
	EventID       string               `json:"event_id"`
	At            time.Time            `json:"at"`
	PolicyVersion string               `json:"policy_version,omitempty"`
	TraceHash     string               `json:"trace_hash,omitempty"`
	Model         string               `json:"model,omitempty"`
}

type GoalGateSidecarResult struct {
	Review       GoalReviewResult   `json:"review"`
	Outcome      GoalGateOutcome    `json:"outcome"`
	AppendRecord ledger.AppendRecord `json:"append_record"`
}

func ReviewGoalCompletion(ctx context.Context, s Spec, e Event, now time.Time, command string, args ...string) (GoalReviewResult, error) {
	p, err := RunExternalAgent(ctx, s, e, nil, command, args...)
	if err != nil {
		return GoalReviewResult{}, err
	}
	return EvaluateGoalProposal(s, e, p, now)
}

func ReviewGoalCompletionInDir(ctx context.Context, s Spec, e Event, now time.Time, command string, workdir string, args ...string) (GoalReviewResult, error) {
	p, err := RunExternalAgentInDir(ctx, s, e, command, workdir, args...)
	if err != nil {
		return GoalReviewResult{}, err
	}
	return EvaluateGoalProposal(s, e, p, now)
}

func EvaluateGoalProposal(s Spec, e Event, p Proposal, now time.Time) (GoalReviewResult, error) {
	if err := ValidateEvent(s, e); err != nil {
		return GoalReviewResult{}, fmt.Errorf("event: %w", err)
	}
	if err := ValidateProposal(s, p); err != nil {
		return GoalReviewResult{}, fmt.Errorf("proposal: %w", err)
	}
	if e.Hook != GoalBeforeCompleteHook {
		return GoalReviewResult{}, fmt.Errorf("goal review requires hook %q", GoalBeforeCompleteHook)
	}
	state, err := ExtractGoalState(e.Inputs)
	if err != nil {
		return GoalReviewResult{}, err
	}

	result := GoalReviewResult{
		ProposalID:          p.ID,
		Steward:             p.Steward,
		Hook:                p.Hook,
		Attempt:             state.Attempt,
		NextAttempt:         state.Attempt + 1,
		MaxContinueAttempts: state.MaxContinueAttempts,
		CooldownSeconds:     state.CooldownSeconds,
	}

	var approve *Output
	var reject *Output
	var requestContinue *Output
	for _, output := range p.Outputs {
		result.AppliedActions = append(result.AppliedActions, output.Action)
		switch output.Action {
		case "goal.approve_complete":
			if approve != nil {
				return GoalReviewResult{}, fmt.Errorf("proposal %q has multiple goal.approve_complete outputs", p.ID)
			}
			copy := output
			approve = &copy
		case "goal.reject_complete":
			if reject != nil {
				return GoalReviewResult{}, fmt.Errorf("proposal %q has multiple goal.reject_complete outputs", p.ID)
			}
			copy := output
			reject = &copy
		case "goal.request_continue":
			if requestContinue != nil {
				return GoalReviewResult{}, fmt.Errorf("proposal %q has multiple goal.request_continue outputs", p.ID)
			}
			copy := output
			requestContinue = &copy
		case "context.inject":
			patch, err := newGoalContinuationPatch(output)
			if err != nil {
				return GoalReviewResult{}, fmt.Errorf("proposal %q %w", p.ID, err)
			}
			result.ContinuationPatches = append(result.ContinuationPatches, patch)
		case "ledger.artifact.create":
			result.Artifacts = append(result.Artifacts, DryRunRef{
				Type:        output.ArtifactType,
				ContentHash: output.ContentHash,
				Ref:         output.Ref,
			})
		case "diagnosis.note.create":
			result.Diagnostics = append(result.Diagnostics, DryRunRef{
				NoteHash: output.NoteHash,
				Ref:      output.Ref,
				Severity: output.Severity,
			})
		}
	}

	if approve != nil && reject != nil {
		return GoalReviewResult{}, fmt.Errorf("proposal %q must not approve and reject completion at the same time", p.ID)
	}
	if approve == nil && reject == nil {
		return GoalReviewResult{}, fmt.Errorf("proposal %q must include exactly one terminal goal action", p.ID)
	}
	if approve != nil {
		if requestContinue != nil {
			return GoalReviewResult{}, fmt.Errorf("proposal %q must not request continuation after approval", p.ID)
		}
		result.Decision = GoalDecisionApprove
		result.Reason = strings.TrimSpace(approve.Reason)
		result.ContinueAllowed = false
		result.NextAttempt = state.Attempt
		return result, nil
	}
	if requestContinue == nil {
		return GoalReviewResult{}, fmt.Errorf("proposal %q must pair goal.reject_complete with goal.request_continue", p.ID)
	}

	result.Decision = GoalDecisionReject
	result.Reason = strings.TrimSpace(reject.Reason)
	result.ContinueInstruction = strings.TrimSpace(requestContinue.Instruction)
	result.ContinueAllowed = true
	result.RejectionReasonHash = hashReason(result.Reason)

	if state.MaxContinueAttempts > 0 && state.Attempt >= state.MaxContinueAttempts {
		result.ContinueAllowed = false
		result.ContinueBlockedReason = fmt.Sprintf("max_continue_attempts reached (%d/%d)", state.Attempt, state.MaxContinueAttempts)
	}
	if state.LastRejectionReasonHash != "" && strings.EqualFold(strings.TrimSpace(state.LastRejectionReasonHash), result.RejectionReasonHash) {
		result.DuplicateReason = true
		result.ContinueAllowed = false
		if result.ContinueBlockedReason == "" {
			result.ContinueBlockedReason = "duplicate rejection reason"
		}
	}
	if state.CooldownSeconds > 0 && state.LastRejectionAtUnix > 0 {
		cooldownUntil := time.Unix(state.LastRejectionAtUnix, 0).Add(time.Duration(state.CooldownSeconds) * time.Second)
		if now.Before(cooldownUntil) {
			result.CooldownActive = true
			result.CooldownRemainingSeconds = int(math.Ceil(cooldownUntil.Sub(now).Seconds()))
			result.ContinueAllowed = false
			if result.ContinueBlockedReason == "" {
				result.ContinueBlockedReason = "cooldown_active"
			}
		}
	}

	return result, nil
}

func newGoalContinuationPatch(output Output) (DryRunPatch, error) {
	if strings.TrimSpace(output.Role) == "" {
		return DryRunPatch{}, fmt.Errorf("context.inject continuation patch role is required")
	}
	if strings.TrimSpace(output.Position) == "" {
		return DryRunPatch{}, fmt.Errorf("context.inject continuation patch position is required")
	}
	if strings.TrimSpace(output.Text) == "" {
		return DryRunPatch{}, fmt.Errorf("context.inject continuation patch text is required")
	}
	return newInjectPatch("continuation", 0, output), nil
}

func ApplyGoalReviewResult(result GoalReviewResult, state GoalState, at time.Time) GoalGateOutcome {
	metadata := map[string]string{
		"proposal_id": result.ProposalID,
		"decision":    string(result.Decision),
	}
	if result.Steward != "" {
		metadata["steward"] = result.Steward
	}
	if result.Hook != "" {
		metadata["hook"] = result.Hook
	}
	if result.Reason != "" {
		metadata["reason_hash"] = result.RejectionReasonHash
	}
	next := state
	outcome := GoalGateOutcome{
		ProposalID:            result.ProposalID,
		Decision:              result.Decision,
		Reason:                result.Reason,
		ContinueInstruction:   result.ContinueInstruction,
		BlockedReason:         result.ContinueBlockedReason,
		RejectionReasonHash:   result.RejectionReasonHash,
		CooldownRemainingSecs: result.CooldownRemainingSeconds,
		LedgerEventMetadata:   metadata,
		NextGoalState:         next,
	}

	switch result.Decision {
	case GoalDecisionApprove:
		next.Status = "complete"
		outcome.AllowComplete = true
		outcome.ContinueWork = false
		outcome.LedgerEventType = "harness_action"
		outcome.LedgerEventAction = "goal.approve_complete"
	case GoalDecisionReject:
		next.Status = "pending_complete"
		next.LastRejectionReasonHash = result.RejectionReasonHash
		next.LastRejectionAtUnix = at.UTC().Unix()
		if result.ContinueAllowed {
			next.Attempt = result.NextAttempt
			outcome.AllowComplete = false
			outcome.ContinueWork = true
			outcome.LedgerEventType = "harness_action"
			outcome.LedgerEventAction = "goal.request_continue"
		} else {
			outcome.AllowComplete = false
			outcome.ContinueWork = false
			outcome.LedgerEventType = "error"
			outcome.LedgerEventAction = "goal.reject_complete"
			outcome.LedgerEventErrorCode = normalizeGoalGateErrorCode(result)
		}
	default:
		outcome.AllowComplete = false
		outcome.ContinueWork = false
		outcome.LedgerEventType = "error"
		outcome.LedgerEventAction = "goal.review.invalid"
		outcome.LedgerEventErrorCode = "goal_gate_invalid_decision"
	}
	outcome.NextGoalState = next
	return outcome
}

func ExtractGoalState(raw json.RawMessage) (GoalState, error) {
	var inputs map[string]json.RawMessage
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	if err := decoder.Decode(&inputs); err != nil {
		return GoalState{}, fmt.Errorf("decode goal review inputs: %w", err)
	}
	rawGoalState, ok := inputs["goal_state"]
	if !ok {
		return GoalState{}, fmt.Errorf("goal review requires inputs.goal_state")
	}
	var state GoalState
	if err := json.Unmarshal(rawGoalState, &state); err != nil {
		return GoalState{}, fmt.Errorf("decode goal_state: %w", err)
	}
	if state.Attempt < 0 {
		return GoalState{}, fmt.Errorf("goal_state.attempt must be non-negative")
	}
	if state.MaxContinueAttempts < 0 {
		return GoalState{}, fmt.Errorf("goal_state.max_continue_attempts must be non-negative")
	}
	if state.CooldownSeconds < 0 {
		return GoalState{}, fmt.Errorf("goal_state.cooldown_seconds must be non-negative")
	}
	if state.LastRejectionAtUnix < 0 {
		return GoalState{}, fmt.Errorf("goal_state.last_rejection_at_unix must be non-negative")
	}
	return state, nil
}

func hashReason(reason string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(reason)))
	return fmt.Sprintf("sha256:%x", sum)
}

func normalizeGoalGateErrorCode(result GoalReviewResult) string {
	if result.DuplicateReason {
		return "goal_gate_duplicate_rejection_reason"
	}
	if result.CooldownActive {
		return "goal_gate_cooldown_active"
	}
	if strings.Contains(result.ContinueBlockedReason, "max_continue_attempts") {
		return "goal_gate_max_continue_attempts_reached"
	}
	if result.ContinueBlockedReason != "" {
		return "goal_gate_continue_blocked"
	}
	return "goal_gate_rejected_without_continue"
}

func BuildGoalGateAppendRecord(input GoalGateAuditInput, outcome GoalGateOutcome) (ledger.AppendRecord, error) {
	if strings.TrimSpace(input.Project.ID) == "" {
		return ledger.AppendRecord{}, fmt.Errorf("goal gate audit project id is required")
	}
	if strings.TrimSpace(input.Session.ID) == "" {
		return ledger.AppendRecord{}, fmt.Errorf("goal gate audit session id is required")
	}
	if strings.TrimSpace(input.EventID) == "" {
		return ledger.AppendRecord{}, fmt.Errorf("goal gate audit event id is required")
	}
	if input.At.IsZero() {
		return ledger.AppendRecord{}, fmt.Errorf("goal gate audit timestamp is required")
	}
	metadata := map[string]string{}
	for key, value := range outcome.LedgerEventMetadata {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		metadata[key] = value
	}
	if outcome.RejectionReasonHash != "" {
		metadata["rejection_reason_hash"] = outcome.RejectionReasonHash
	}
	if outcome.BlockedReason != "" {
		metadata["blocked_reason"] = outcome.BlockedReason
	}
	if outcome.ContinueInstruction != "" {
		sum := sha256.Sum256([]byte(outcome.ContinueInstruction))
		metadata["continue_instruction_hash"] = fmt.Sprintf("sha256:%x", sum)
	}
	event := ledger.Event{
		ID:            input.EventID,
		Type:          outcome.LedgerEventType,
		At:            input.At.UTC().Format(time.RFC3339),
		Model:         strings.TrimSpace(input.Model),
		Hook:          GoalBeforeCompleteHook,
		Action:        outcome.LedgerEventAction,
		PolicyVersion: strings.TrimSpace(input.PolicyVersion),
		TraceHash:     strings.TrimSpace(input.TraceHash),
		ErrorCode:     outcome.LedgerEventErrorCode,
		Metadata:      metadata,
	}
	return ledger.AppendRecord{
		Project: input.Project,
		Session: input.Session,
		Event:   event,
	}, nil
}

func ExecuteGoalGateSidecar(ctx context.Context, s Spec, e Event, audit GoalGateAuditInput, now time.Time, command string, args ...string) (GoalGateSidecarResult, error) {
	review, err := ReviewGoalCompletion(ctx, s, e, now, command, args...)
	if err != nil {
		return GoalGateSidecarResult{}, err
	}
	state, err := ExtractGoalState(e.Inputs)
	if err != nil {
		return GoalGateSidecarResult{}, err
	}
	outcome := ApplyGoalReviewResult(review, state, now)
	record, err := BuildGoalGateAppendRecord(audit, outcome)
	if err != nil {
		return GoalGateSidecarResult{}, err
	}
	return GoalGateSidecarResult{
		Review:       review,
		Outcome:      outcome,
		AppendRecord: record,
	}, nil
}

func ExecuteGoalGateSidecarInDir(ctx context.Context, s Spec, e Event, audit GoalGateAuditInput, now time.Time, command string, workdir string, args ...string) (GoalGateSidecarResult, error) {
	review, err := ReviewGoalCompletionInDir(ctx, s, e, now, command, workdir, args...)
	if err != nil {
		return GoalGateSidecarResult{}, err
	}
	state, err := ExtractGoalState(e.Inputs)
	if err != nil {
		return GoalGateSidecarResult{}, err
	}
	outcome := ApplyGoalReviewResult(review, state, now)
	record, err := BuildGoalGateAppendRecord(audit, outcome)
	if err != nil {
		return GoalGateSidecarResult{}, err
	}
	return GoalGateSidecarResult{
		Review:       review,
		Outcome:      outcome,
		AppendRecord: record,
	}, nil
}

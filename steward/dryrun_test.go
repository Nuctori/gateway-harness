package steward

import (
	"strings"
	"testing"
)

func TestDryRunProposalInjectsIntoResponsesInput(t *testing.T) {
	s, err := Decode(strings.NewReader(compactStewardJSON))
	if err != nil {
		t.Fatalf("decode spec: %v", err)
	}
	p, err := DecodeProposal(strings.NewReader(compactStewardProposalJSON))
	if err != nil {
		t.Fatalf("decode proposal: %v", err)
	}
	result, err := DryRunProposal(s, p, []byte(compactRequestJSON))
	if err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	if len(result.RequestPatches) != 1 {
		t.Fatalf("expected one request patch, got %+v", result.RequestPatches)
	}
	patch := result.RequestPatches[0]
	if patch.Target != "input" || patch.InsertIndex != 1 || patch.Role != "system" {
		t.Fatalf("unexpected patch: %+v", patch)
	}
	if patch.ContentHash == "" || patch.ContentChars == 0 {
		t.Fatalf("expected redacted content metadata: %+v", patch)
	}
	if len(result.Artifacts) != 1 || result.Artifacts[0].Type != "compact_summary" {
		t.Fatalf("unexpected artifacts: %+v", result.Artifacts)
	}
	if len(result.Diagnostics) != 1 || result.Diagnostics[0].NoteHash == "" {
		t.Fatalf("unexpected diagnostics: %+v", result.Diagnostics)
	}
	if len(result.SessionTags) != 2 {
		t.Fatalf("unexpected session tags: %+v", result.SessionTags)
	}
}

func TestDryRunProposalRejectsTruncate(t *testing.T) {
	s, err := Decode(strings.NewReader(strings.Replace(compactStewardJSON, `"context.inject"`, `"context.truncate"`, 1)))
	if err != nil {
		t.Fatalf("decode spec: %v", err)
	}
	if err := Validate(s); err == nil {
		t.Fatal("expected truncate rejection at steward validation")
	}
}

func TestDryRunProposalPreservesResponsesToolChainPrefix(t *testing.T) {
	s, err := Decode(strings.NewReader(compactStewardJSON))
	if err != nil {
		t.Fatalf("decode spec: %v", err)
	}
	p, err := DecodeProposal(strings.NewReader(compactStewardProposalJSON))
	if err != nil {
		t.Fatalf("decode proposal: %v", err)
	}
	result, err := DryRunProposal(s, p, []byte(statefulResponsesRequestJSON))
	if err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	if len(result.RequestPatches) != 1 {
		t.Fatalf("expected one request patch, got %+v", result.RequestPatches)
	}
	patch := result.RequestPatches[0]
	if patch.Target != "input" || patch.InsertIndex != 2 {
		t.Fatalf("tool-chain prefix insert index was not preserved: %+v", patch)
	}
}

func TestDryRunProposalReturnsGoalActions(t *testing.T) {
	s, err := Decode(strings.NewReader(goalGateStewardJSON))
	if err != nil {
		t.Fatalf("decode spec: %v", err)
	}
	p, err := DecodeProposal(strings.NewReader(goalGateRejectProposalJSON))
	if err != nil {
		t.Fatalf("decode proposal: %v", err)
	}
	result, err := DryRunProposal(s, p, []byte(goalGateRequestJSON))
	if err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	if len(result.RequestPatches) != 0 {
		t.Fatalf("goal actions must not mutate request patches: %+v", result.RequestPatches)
	}
	if len(result.GoalActions) != 2 {
		t.Fatalf("expected two goal actions, got %+v", result.GoalActions)
	}
	if result.GoalActions[0].Action != "goal.reject_complete" {
		t.Fatalf("unexpected first goal action: %+v", result.GoalActions[0])
	}
	if result.GoalActions[1].Action != "goal.request_continue" || result.GoalActions[1].Instruction == "" {
		t.Fatalf("unexpected continuation goal action: %+v", result.GoalActions[1])
	}
	if len(result.AppliedActions) != 2 {
		t.Fatalf("unexpected applied actions: %+v", result.AppliedActions)
	}
}

const compactRequestJSON = `{
  "model": "gpt-5.4-mini",
  "input": [
    {"role": "system", "content": "Existing system instruction."},
    {"role": "user", "content": "Continue the current coding task after compaction."}
  ]
}`

const statefulResponsesRequestJSON = `{
  "model": "gpt-5.4-mini",
  "previous_response_id": "resp_1",
  "input": [
    {"type": "item_reference", "id": "fc_1"},
    {"type": "function_call_output", "call_id": "call_1", "output": "{\"ok\":true}"},
    {"role": "user", "content": "continue"}
  ]
}`

const goalGateRequestJSON = `{
  "goal": {
    "id": "goal_gate_demo",
    "status": "pending_complete"
  },
  "summary": "Minimal placeholder request object for goal dry-run validation."
}`

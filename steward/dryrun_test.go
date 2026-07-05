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
	if len(result.PolicyPatches) != 1 || result.PolicyPatches[0].PatchHash == "" {
		t.Fatalf("unexpected policy patches: %+v", result.PolicyPatches)
	}
}

func TestDryRunProposalRejectsTruncate(t *testing.T) {
	s, err := Decode(strings.NewReader(strings.Replace(compactStewardJSON, `"context.inject", `, `"context.truncate", `, 1)))
	if err != nil {
		t.Fatalf("decode spec: %v", err)
	}
	raw := strings.Replace(compactStewardProposalJSON, `"context.inject"`, `"context.truncate"`, 1)
	raw = strings.Replace(raw, `"role": "system",
      "position": "after_existing_system",
      "text": "Continue preserving the user's active goal, current blockers, verified decisions, and unresolved follow-ups from the compact summary artifact."`, `"keep_last_messages": 12`, 1)
	p, err := DecodeProposal(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("decode proposal: %v", err)
	}
	if _, err := DryRunProposal(s, p, []byte(compactRequestJSON)); err == nil {
		t.Fatal("expected dry-run truncate rejection")
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

package steward

import (
	"strings"
	"testing"
)

func TestValidateAcceptsCompactSteward(t *testing.T) {
	s, err := Decode(strings.NewReader(compactStewardJSON))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if err := Validate(s); err != nil {
		t.Fatalf("validate: %v", err)
	}
	summary := Summarize(s)
	if summary.Name != "newapi-compact-context-steward" || summary.Hooks != 1 || summary.Inputs != 5 || summary.AllowedActions != 4 || summary.ArtifactTypes != 1 || summary.RequiredGuards != 5 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
}

func TestValidateRejectsWildcardHook(t *testing.T) {
	raw := strings.Replace(compactStewardJSON, `"responses.compact.before_upstream"`, `"*"`, 1)
	s, err := Decode(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if err := Validate(s); err == nil {
		t.Fatal("expected wildcard hook error")
	}
}

func TestValidateRejectsRawTranscriptInput(t *testing.T) {
	raw := strings.Replace(compactStewardJSON, `"redacted_trace"`, `"raw_transcript"`, 1)
	s, err := Decode(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if err := Validate(s); err == nil {
		t.Fatal("expected unsupported raw transcript input error")
	}
}

func TestValidateRejectsPolicyPatchAction(t *testing.T) {
	raw := strings.Replace(compactStewardJSON, `"session.tags.update"`, `"policy.patch.propose"`, 1)
	s, err := Decode(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if err := Validate(s); err == nil {
		t.Fatal("expected policy patch action error")
	}
}

func TestValidateRejectsArtifactActionWithoutArtifactTypes(t *testing.T) {
	raw := strings.Replace(compactStewardJSON, `  "artifact_types": ["compact_summary"],
`, ``, 1)
	s, err := Decode(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if err := Validate(s); err == nil {
		t.Fatal("expected missing artifact type error")
	}
}

func TestValidateProposalAcceptsCompactContextProposal(t *testing.T) {
	s, err := Decode(strings.NewReader(compactStewardJSON))
	if err != nil {
		t.Fatalf("decode spec: %v", err)
	}
	p, err := DecodeProposal(strings.NewReader(compactStewardProposalJSON))
	if err != nil {
		t.Fatalf("decode proposal: %v", err)
	}
	if err := ValidateProposal(s, p); err != nil {
		t.Fatalf("validate proposal: %v", err)
	}
	summary := SummarizeProposal(p)
	if summary.ID != "proposal_compact_context_1" || summary.Outputs != 4 {
		t.Fatalf("unexpected proposal summary: %+v", summary)
	}
}

func TestValidateProposalRejectsNotAllowedAction(t *testing.T) {
	s, err := Decode(strings.NewReader(compactStewardJSON))
	if err != nil {
		t.Fatalf("decode spec: %v", err)
	}
	raw := strings.Replace(compactStewardProposalJSON, `"context.inject"`, `"diagnosis.note.create"`, 1)
	p, err := DecodeProposal(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("decode proposal: %v", err)
	}
	if err := ValidateProposal(s, p); err == nil {
		t.Fatal("expected action not allowed error")
	}
}

func TestValidateProposalRejectsWrongHook(t *testing.T) {
	s, err := Decode(strings.NewReader(compactStewardJSON))
	if err != nil {
		t.Fatalf("decode spec: %v", err)
	}
	raw := strings.Replace(compactStewardProposalJSON, `"responses.compact.before_upstream"`, `"responses.before_upstream"`, 1)
	p, err := DecodeProposal(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("decode proposal: %v", err)
	}
	if err := ValidateProposal(s, p); err == nil {
		t.Fatal("expected hook not enabled error")
	}
}

func TestValidateProposalRejectsMissingArtifactHash(t *testing.T) {
	s, err := Decode(strings.NewReader(compactStewardJSON))
	if err != nil {
		t.Fatalf("decode spec: %v", err)
	}
	raw := strings.Replace(compactStewardProposalJSON, `"content_hash": "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",`, ``, 1)
	p, err := DecodeProposal(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("decode proposal: %v", err)
	}
	if err := ValidateProposal(s, p); err == nil {
		t.Fatal("expected missing artifact hash error")
	}
}

func TestValidateProposalRejectsIrrelevantActionField(t *testing.T) {
	s, err := Decode(strings.NewReader(compactStewardJSON))
	if err != nil {
		t.Fatalf("decode spec: %v", err)
	}
	raw := strings.Replace(compactStewardProposalJSON, `"artifact_type": "compact_summary",`, `"artifact_type": "compact_summary", "text": "raw hidden side channel",`, 1)
	p, err := DecodeProposal(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("decode proposal: %v", err)
	}
	if err := ValidateProposal(s, p); err == nil {
		t.Fatal("expected irrelevant field error")
	}
}

func TestValidateProposalRejectsIrrelevantContextInjectField(t *testing.T) {
	s, err := Decode(strings.NewReader(compactStewardJSON))
	if err != nil {
		t.Fatalf("decode spec: %v", err)
	}
	raw := strings.Replace(compactStewardProposalJSON, `"role": "system",`, `"role": "system", "tags": ["hidden-side-channel"],`, 1)
	p, err := DecodeProposal(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("decode proposal: %v", err)
	}
	if err := ValidateProposal(s, p); err == nil {
		t.Fatal("expected irrelevant context.inject field error")
	}
}

const compactStewardJSON = `{
  "version": "0.1",
  "name": "newapi-compact-context-steward",
  "steward_model": "kimi-for-coding",
  "hooks": ["responses.compact.before_upstream"],
  "inputs": ["user_goal", "session_tags", "ledger_event_metadata", "artifact_refs", "redacted_trace"],
  "allowed_actions": ["context.inject", "ledger.artifact.create", "diagnosis.note.create", "session.tags.update"],
  "artifact_types": ["compact_summary"],
  "required_guards": ["explicit_invocation_only", "structured_output_only", "validate_output_actions", "redacted_input_only", "artifact_hash_required"]
}`

const compactStewardProposalJSON = `{
  "version": "0.1",
  "id": "proposal_compact_context_1",
  "steward": "newapi-compact-context-steward",
  "hook": "responses.compact.before_upstream",
  "outputs": [
    {
      "action": "ledger.artifact.create",
      "reason": "store compact summary as an external artifact reference",
      "artifact_type": "compact_summary",
      "content_hash": "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
      "ref": "memory://summaries/session_codex_context_harness/compact_steward_1"
    },
    {
      "action": "context.inject",
      "reason": "inject compact-time working memory produced by the explicit steward",
      "role": "system",
      "position": "after_existing_system",
      "text": "Continue preserving the user's active goal, current blockers, verified decisions, and unresolved follow-ups from the compact summary artifact."
    },
    {
      "action": "diagnosis.note.create",
      "reason": "record why the compact steward injected continuity guidance",
      "note_hash": "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
      "ref": "memory://diagnostics/session_codex_context_harness/compact_steward_1",
      "severity": "info"
    },
    {
      "action": "session.tags.update",
      "reason": "mark the session as compact-stewarded for later audit",
      "tags": ["continuity:compacted", "steward:ai-in-loop"]
    }
  ]
}`

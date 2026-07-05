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
	if summary.Name != "newapi-compact-context-steward" || summary.Hooks != 1 || summary.Inputs != 5 || summary.AllowedActions != 3 || summary.ArtifactTypes != 1 || summary.RequiredGuards != 6 {
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

func TestValidateRejectsPolicyPatchWithoutHumanApproval(t *testing.T) {
	raw := strings.Replace(compactStewardJSON, `, "human_approval_for_policy_patch"`, ``, 1)
	s, err := Decode(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if err := Validate(s); err == nil {
		t.Fatal("expected policy patch guard error")
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

const compactStewardJSON = `{
  "version": "0.1",
  "name": "newapi-compact-context-steward",
  "steward_model": "kimi-for-coding",
  "hooks": ["responses.compact.before_upstream"],
  "inputs": ["user_goal", "session_tags", "ledger_event_metadata", "artifact_refs", "redacted_trace"],
  "allowed_actions": ["context.inject", "ledger.artifact.create", "policy.patch.propose"],
  "artifact_types": ["compact_summary"],
  "required_guards": ["explicit_invocation_only", "structured_output_only", "validate_output_actions", "redacted_input_only", "artifact_hash_required", "human_approval_for_policy_patch"]
}`

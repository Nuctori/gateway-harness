package rule

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

func TestCompileContinuityDropRuleToLedgerSummaryPolicy(t *testing.T) {
	doc, err := Decode(strings.NewReader(continuityDropRuleJSON))
	if err != nil {
		t.Fatalf("decode rule: %v", err)
	}

	compiled, err := Compile(doc)
	if err != nil {
		t.Fatalf("compile rule: %v", err)
	}

	if len(compiled.Programs) != 1 {
		t.Fatalf("unexpected programs: %+v", compiled.Programs)
	}
	step := compiled.Programs[0].Steps[0]
	if len(step.Hooks) != 1 || step.Hooks[0] != "context.continuity_drop.detected" {
		t.Fatalf("unexpected hooks: %+v", step.Hooks)
	}
	if !step.When.ContextContinuityDrop {
		t.Fatalf("expected continuity drop condition")
	}
	action := step.Do[0]
	if action.Action != "context.inject_ledger_summary" {
		t.Fatalf("unexpected action: %+v", action)
	}
	if action.Source != "ledger.summary" || action.LedgerRef != "ledger://project_gateway_harness/session_codex_context_harness" {
		t.Fatalf("unexpected ledger provenance: %+v", action)
	}

	encoded, err := json.Marshal(compiled)
	if err != nil {
		t.Fatalf("marshal compiled policy: %v", err)
	}
	if strings.Contains(string(encoded), "budget") || strings.Contains(string(encoded), "context.truncate") || strings.Contains(string(encoded), "ask_steward") {
		t.Fatalf("compiled rule introduced hidden behavior: %s", encoded)
	}
}

func TestCompileAuditDisabledRuleToPlainInject(t *testing.T) {
	doc, err := Decode(strings.NewReader(plainRuleJSON))
	if err != nil {
		t.Fatalf("decode rule: %v", err)
	}

	compiled, err := Compile(doc)
	if err != nil {
		t.Fatalf("compile rule: %v", err)
	}

	action := compiled.Programs[0].Steps[0].Do[0]
	if action.Action != "context.inject" {
		t.Fatalf("unexpected action: %+v", action)
	}
	if action.Source != "" || action.LedgerRef != "" || len(action.ArtifactRefs) != 0 {
		t.Fatalf("plain inject should not carry ledger provenance: %+v", action)
	}
}

func TestCompileRejectsUnknownOperation(t *testing.T) {
	doc, err := Decode(strings.NewReader(strings.Replace(plainRuleJSON, `"inject_capsule"`, `"not_a_real_operation"`, 1)))
	if err != nil {
		t.Fatalf("decode rule: %v", err)
	}

	_, err = Compile(doc)
	if err == nil || !strings.Contains(err.Error(), `unsupported operation "not_a_real_operation"`) {
		t.Fatalf("expected unsupported operation error, got %v", err)
	}
}

func TestCompileAskStewardRuleToStewardSpec(t *testing.T) {
	doc, err := Decode(strings.NewReader(askStewardRuleJSON))
	if err != nil {
		t.Fatalf("decode rule: %v", err)
	}

	specs, err := CompileStewards(doc)
	if err != nil {
		t.Fatalf("compile stewards: %v", err)
	}
	if len(specs) != 1 {
		t.Fatalf("unexpected specs: %+v", specs)
	}
	spec := specs[0]
	if spec.Name != "codex-compact-steward" || spec.StewardModel != "kimi-for-coding" {
		t.Fatalf("unexpected steward identity: %+v", spec)
	}
	if len(spec.Hooks) != 1 || spec.Hooks[0] != "responses.compact.before_upstream" {
		t.Fatalf("unexpected hooks: %+v", spec.Hooks)
	}
	if !slices.Contains(spec.RequiredGuards, "redacted_input_only") || !slices.Contains(spec.RequiredGuards, "explicit_invocation_only") {
		t.Fatalf("missing transparency guards: %+v", spec.RequiredGuards)
	}

	encoded, err := json.Marshal(spec)
	if err != nil {
		t.Fatalf("marshal steward spec: %v", err)
	}
	if strings.Contains(string(encoded), "raw_prompt") || strings.Contains(string(encoded), "context.truncate") {
		t.Fatalf("compiled steward introduced unsafe fields: %s", encoded)
	}
}

func TestCompilePolicyRejectsAskStewardRule(t *testing.T) {
	doc, err := Decode(strings.NewReader(askStewardRuleJSON))
	if err != nil {
		t.Fatalf("decode rule: %v", err)
	}

	_, err = Compile(doc)
	if err == nil || !strings.Contains(err.Error(), "cannot compile to policy") {
		t.Fatalf("expected policy compile rejection, got %v", err)
	}
}

func TestAskStewardRejectsWildcardHook(t *testing.T) {
	doc, err := Decode(strings.NewReader(strings.Replace(askStewardRuleJSON, `"responses.compact.before_upstream"`, `"*"`, 1)))
	if err != nil {
		t.Fatalf("decode rule: %v", err)
	}

	_, err = CompileStewards(doc)
	if err == nil || !strings.Contains(err.Error(), "must not use wildcard hook for AI-in-the-loop") {
		t.Fatalf("expected wildcard hook rejection, got %v", err)
	}
}

func TestAskStewardRejectsMissingRedactedInputGuard(t *testing.T) {
	doc, err := Decode(strings.NewReader(strings.Replace(askStewardRuleJSON, `"redacted_input_only",`, "", 1)))
	if err != nil {
		t.Fatalf("decode rule: %v", err)
	}

	_, err = CompileStewards(doc)
	if err == nil || !strings.Contains(err.Error(), `requires guard "redacted_input_only"`) {
		t.Fatalf("expected redacted input guard rejection, got %v", err)
	}
}

func TestAskStewardRejectsPolicyPatchApprovalWorkflow(t *testing.T) {
	raw := strings.Replace(
		askStewardRuleJSON,
		`"context.inject", "ledger.artifact.create", "diagnosis.note.create"`,
		`"context.inject", "ledger.artifact.create", "policy.patch.propose"`,
		1,
	)
	raw = strings.Replace(
		raw,
		`"artifact_hash_required"`,
		`"artifact_hash_required", "human_approval_for_policy_patch"`,
		1,
	)
	doc, err := Decode(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("decode rule: %v", err)
	}

	_, err = CompileStewards(doc)
	if err == nil || !strings.Contains(err.Error(), "policy.patch.propose") {
		t.Fatalf("expected policy patch rejection, got %v", err)
	}
}

func TestAskStewardRejectsTruncate(t *testing.T) {
	raw := strings.Replace(
		askStewardRuleJSON,
		`"context.inject", "ledger.artifact.create", "diagnosis.note.create"`,
		`"context.inject", "context.truncate", "ledger.artifact.create"`,
		1,
	)
	doc, err := Decode(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("decode rule: %v", err)
	}

	_, err = CompileStewards(doc)
	if err == nil || !strings.Contains(err.Error(), "context.truncate") {
		t.Fatalf("expected truncate rejection, got %v", err)
	}
}

const continuityDropRuleJSON = `{
  "version": "0.2",
  "rules": [
    {
      "name": "continuity-drop-ledger-sentinel",
      "tags": ["adapter:newapi", "domain:coding"],
      "trigger": {
        "hooks": ["context.continuity_drop.detected"],
        "continuity_drop": true
      },
      "scope": {
        "models": ["*"],
        "model_matches": "*"
      },
      "operation": {
        "type": "inject_capsule",
        "role": "system",
        "position": "after_existing_system",
        "reason": "project continuity capsule after detected context drop",
        "text": "Project memory sentinel: preserve explicit user goals and unresolved tasks."
      },
      "audit": {
        "ledger_ref": "ledger://project_gateway_harness/session_codex_context_harness",
        "artifact_refs": ["artifact_continuity_drop_1"]
      }
    }
  ]
}`

const plainRuleJSON = `{
  "rules": [
    {
      "name": "plain-context-capsule",
      "trigger": {
        "hooks": ["responses.before_upstream"]
      },
      "scope": {
        "models": ["gpt-5.4-mini"]
      },
      "operation": {
        "type": "inject_capsule",
        "role": "system",
        "position": "after_existing_system",
        "text": "Preserve user intent."
      }
    }
  ]
}`

const askStewardRuleJSON = `{
  "version": "0.3",
  "rules": [
    {
      "name": "compact-ai-steward",
      "trigger": {
        "hooks": ["responses.compact.before_upstream"]
      },
      "scope": {
        "models": ["*"]
      },
      "operation": {
        "type": "ask_steward",
        "steward_name": "codex-compact-steward",
        "steward_model": "kimi-for-coding",
        "inputs": ["user_goal", "session_tags", "ledger_event_metadata", "artifact_refs", "redacted_trace"],
		"allowed_actions": ["context.inject", "ledger.artifact.create", "diagnosis.note.create"],
		"artifact_types": ["compact_summary"],
		"required_guards": [
		  "explicit_invocation_only",
		  "structured_output_only",
		  "validate_output_actions",
		  "redacted_input_only",
		  "artifact_hash_required"
		]
      }
    }
  ]
}`

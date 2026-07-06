package rule

import (
	"encoding/json"
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
	doc, err := Decode(strings.NewReader(strings.Replace(plainRuleJSON, `"inject_capsule"`, `"ask_steward"`, 1)))
	if err != nil {
		t.Fatalf("decode rule: %v", err)
	}

	_, err = Compile(doc)
	if err == nil || !strings.Contains(err.Error(), `unsupported operation "ask_steward"`) {
		t.Fatalf("expected unsupported operation error, got %v", err)
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

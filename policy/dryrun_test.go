package policy

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDryRunInjectsRedactedPatchPlan(t *testing.T) {
	p, err := Decode(strings.NewReader(policyDryRunJSON))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	result, err := DryRun(p, []byte(chatRequestJSON), DryRunOptions{Hook: "chat.before_upstream"})
	if err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	if len(result.MatchedPrograms) != 1 || result.MatchedPrograms[0] != "coding" {
		t.Fatalf("unexpected matched programs: %+v", result.MatchedPrograms)
	}
	if len(result.RequestPatches) != 1 {
		t.Fatalf("expected one request patch: %+v", result.RequestPatches)
	}
	patch := result.RequestPatches[0]
	if patch.Target != "messages" || patch.InsertIndex != 1 || patch.ContentHash == "" || patch.ContentChars == 0 {
		t.Fatalf("unexpected patch: %+v", patch)
	}
}

func TestDryRunSkipsDestructiveTruncate(t *testing.T) {
	p, err := Decode(strings.NewReader(policyDryRunWithTruncateJSON))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	result, err := DryRun(p, []byte(chatRequestJSON), DryRunOptions{Hook: "chat.before_upstream", EstimatedTokens: 200})
	if err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	if len(result.SkippedActions) != 1 || result.SkippedActions[0].Action != "context.truncate" {
		t.Fatalf("expected skipped truncate: %+v", result.SkippedActions)
	}
}

func TestDryRunRequiresExplicitTokenEstimateForTokenCondition(t *testing.T) {
	p, err := Decode(strings.NewReader(policyDryRunWithTruncateJSON))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	result, err := DryRun(p, []byte(chatRequestJSON), DryRunOptions{Hook: "chat.before_upstream"})
	if err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	if len(result.SkippedActions) != 0 {
		t.Fatalf("token condition should not match without estimate: %+v", result.SkippedActions)
	}
}

func TestDryRunPreservesResponsesToolChainPrefix(t *testing.T) {
	p, err := Decode(strings.NewReader(responsesPolicyDryRunJSON))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	result, err := DryRun(p, []byte(statefulResponsesRequestJSON), DryRunOptions{Hook: "responses.before_upstream"})
	if err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	if len(result.RequestPatches) != 1 || result.RequestPatches[0].InsertIndex != 2 {
		t.Fatalf("expected injection after tool-chain prefix: %+v", result.RequestPatches)
	}
}

func TestDryRunCompactHookIsExplicitAndRedacted(t *testing.T) {
	p, err := Decode(strings.NewReader(compactPolicyDryRunJSON))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	result, err := DryRun(p, []byte(statefulResponsesRequestJSON), DryRunOptions{Hook: "responses.compact.before_upstream"})
	if err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	if len(result.MatchedPrograms) != 1 || result.MatchedPrograms[0] != "compact" {
		t.Fatalf("unexpected matched programs: %+v", result.MatchedPrograms)
	}
	if len(result.RequestPatches) != 1 {
		t.Fatalf("expected one compact patch: %+v", result.RequestPatches)
	}
	patch := result.RequestPatches[0]
	if patch.Target != "input" || patch.InsertIndex != 2 || patch.Reason != "compact reminder" {
		t.Fatalf("unexpected compact patch: %+v", patch)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal dry-run result: %v", err)
	}
	if strings.Contains(string(encoded), "Preserve goals") {
		t.Fatalf("dry-run result leaked raw content: %s", encoded)
	}
}

func TestDryRunLedgerSummaryPatchIsExplicitAndRedacted(t *testing.T) {
	p, err := Decode(strings.NewReader(ledgerSummaryPolicyDryRunJSON))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	result, err := DryRun(p, []byte(statefulResponsesRequestJSON), DryRunOptions{Hook: "responses.compact.before_upstream"})
	if err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	if len(result.RequestPatches) != 1 {
		t.Fatalf("expected one ledger summary patch: %+v", result.RequestPatches)
	}
	patch := result.RequestPatches[0]
	if patch.Action != "context.inject_ledger_summary" || patch.Source != "ledger.summary" {
		t.Fatalf("unexpected action source: %+v", patch)
	}
	if patch.LedgerRef != "ledger://project_gateway_harness/session_codex_context_harness" {
		t.Fatalf("unexpected ledger ref: %+v", patch)
	}
	if len(patch.ArtifactRefs) != 1 || patch.ArtifactRefs[0] != "artifact_compact_summary_1" {
		t.Fatalf("unexpected artifact refs: %+v", patch)
	}
	if patch.Target != "input" || patch.InsertIndex != 2 {
		t.Fatalf("responses tool-chain prefix was not preserved: %+v", patch)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal dry-run result: %v", err)
	}
	if strings.Contains(string(encoded), "Gateway Harness project memory") {
		t.Fatalf("dry-run result leaked raw ledger summary: %s", encoded)
	}
}

func TestDryRunContinuityDropHookRequiresExplicitEvent(t *testing.T) {
	p, err := Decode(strings.NewReader(continuityDropPolicyDryRunJSON))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	ordinary, err := DryRun(p, []byte(statefulResponsesRequestJSON), DryRunOptions{Hook: "responses.before_upstream"})
	if err != nil {
		t.Fatalf("ordinary dry-run: %v", err)
	}
	if len(ordinary.RequestPatches) != 0 {
		t.Fatalf("ordinary hook must not match continuity drop condition: %+v", ordinary.RequestPatches)
	}
	drop, err := DryRun(p, []byte(statefulResponsesRequestJSON), DryRunOptions{Hook: "context.continuity_drop.detected"})
	if err != nil {
		t.Fatalf("drop dry-run: %v", err)
	}
	if len(drop.RequestPatches) != 1 || drop.RequestPatches[0].Action != "context.inject_ledger_summary" {
		t.Fatalf("expected continuity drop ledger patch: %+v", drop.RequestPatches)
	}
}

func TestValidateLedgerSummaryRequiresLedgerRef(t *testing.T) {
	p, err := Decode(strings.NewReader(ledgerSummaryMissingRefPolicyJSON))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if err := Validate(p); err == nil || !strings.Contains(err.Error(), "ledger_ref is required") {
		t.Fatalf("expected ledger_ref validation error, got %v", err)
	}
}

func TestValidateRejectsLedgerFieldsOnPlainInject(t *testing.T) {
	p, err := Decode(strings.NewReader(injectWithLedgerFieldsPolicyJSON))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if err := Validate(p); err == nil || !strings.Contains(err.Error(), "require context.inject_ledger_summary") {
		t.Fatalf("expected ledger provenance validation error, got %v", err)
	}
}

func TestValidateLedgerSummaryRejectsUnexpectedSource(t *testing.T) {
	p, err := Decode(strings.NewReader(ledgerSummaryWrongSourcePolicyJSON))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if err := Validate(p); err == nil || !strings.Contains(err.Error(), "source must be ledger.summary") {
		t.Fatalf("expected ledger summary source validation error, got %v", err)
	}
}

func TestValidateRejectsTruncateFieldsOnInjectActions(t *testing.T) {
	p, err := Decode(strings.NewReader(injectWithTruncateFieldsPolicyJSON))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if err := Validate(p); err == nil || !strings.Contains(err.Error(), "truncate fields are not allowed") {
		t.Fatalf("expected inject truncate-field validation error, got %v", err)
	}

	p, err = Decode(strings.NewReader(ledgerSummaryWithTruncateFieldsPolicyJSON))
	if err != nil {
		t.Fatalf("decode ledger summary: %v", err)
	}
	if err := Validate(p); err == nil || !strings.Contains(err.Error(), "truncate fields are not allowed") {
		t.Fatalf("expected ledger summary truncate-field validation error, got %v", err)
	}
}

func TestValidateRejectsInjectFieldsOnTruncate(t *testing.T) {
	p, err := Decode(strings.NewReader(truncateWithInjectFieldsPolicyJSON))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if err := Validate(p); err == nil || !strings.Contains(err.Error(), "inject fields are not allowed") {
		t.Fatalf("expected truncate inject-field validation error, got %v", err)
	}
}

const policyDryRunJSON = `{
  "programs": [
    {
      "name": "coding",
      "models": ["gpt-*"],
      "steps": [
        {
          "hook": "chat.before_upstream",
          "when": {"model_matches": "gpt-*"},
          "do": [
            {
              "action": "context.inject",
              "role": "system",
              "position": "after_existing_system",
              "text": "Preserve the user's coding goal.",
              "reason": "coding guardrail"
            }
          ]
        }
      ]
    }
  ]
}`

const policyDryRunWithTruncateJSON = `{
  "programs": [
    {
      "name": "destructive",
      "models": ["gpt-*"],
      "steps": [
        {
          "hook": "chat.before_upstream",
          "when": {"estimated_tokens_gt": 100},
          "do": [
            {"action": "context.truncate", "keep_last_messages": 12}
          ]
        }
      ]
    }
  ]
}`

const responsesPolicyDryRunJSON = `{
  "programs": [
    {
      "name": "responses",
      "models": ["*"],
      "steps": [
        {
          "hook": "responses.before_upstream",
          "do": [
            {
              "action": "context.inject",
              "role": "system",
              "position": "after_existing_system",
              "text": "Preserve Responses state.",
              "reason": "state guardrail"
            }
          ]
        }
      ]
    }
  ]
}`

const compactPolicyDryRunJSON = `{
  "programs": [
    {
      "name": "compact",
      "models": ["*"],
      "steps": [
        {
          "hook": "responses.compact.before_upstream",
          "do": [
            {
              "action": "context.inject",
              "role": "system",
              "position": "after_existing_system",
              "text": "Preserve goals, decisions, constraints, and unresolved tasks.",
              "reason": "compact reminder"
            }
          ]
        }
      ]
    }
  ]
}`

const ledgerSummaryPolicyDryRunJSON = `{
  "programs": [
    {
      "name": "compact-ledger-sentinel",
      "models": ["*"],
      "steps": [
        {
          "hook": "responses.compact.before_upstream",
          "do": [
            {
              "action": "context.inject_ledger_summary",
              "role": "system",
              "position": "after_existing_system",
              "ledger_ref": "ledger://project_gateway_harness/session_codex_context_harness",
              "artifact_refs": ["artifact_compact_summary_1"],
              "text": "Gateway Harness project memory: preserve explicit user goals, architecture decisions, deployment constraints, and unresolved PR work.",
              "reason": "compact ledger sentinel"
            }
          ]
        }
      ]
    }
  ]
}`

const ledgerSummaryMissingRefPolicyJSON = `{
  "programs": [
    {
      "name": "compact-ledger-sentinel",
      "models": ["*"],
      "steps": [
        {
          "hook": "responses.compact.before_upstream",
          "do": [
            {
              "action": "context.inject_ledger_summary",
              "role": "system",
              "position": "after_existing_system",
              "text": "Gateway Harness project memory.",
              "reason": "compact ledger sentinel"
            }
          ]
        }
      ]
    }
  ]
}`

const injectWithLedgerFieldsPolicyJSON = `{
  "programs": [
    {
      "name": "plain-inject-with-ledger-fields",
      "models": ["*"],
      "steps": [
        {
          "hook": "responses.before_upstream",
          "do": [
            {
              "action": "context.inject",
              "role": "system",
              "position": "after_existing_system",
              "ledger_ref": "ledger://project/session",
              "text": "Plain inject should not carry ledger provenance."
            }
          ]
        }
      ]
    }
  ]
}`

const ledgerSummaryWrongSourcePolicyJSON = `{
  "programs": [
    {
      "name": "ledger-summary-wrong-source",
      "models": ["*"],
      "steps": [
        {
          "hook": "responses.compact.before_upstream",
          "do": [
            {
              "action": "context.inject_ledger_summary",
              "role": "system",
              "position": "after_existing_system",
              "source": "memory.hidden",
              "ledger_ref": "ledger://project/session",
              "text": "Gateway Harness project memory."
            }
          ]
        }
      ]
    }
  ]
}`

const injectWithTruncateFieldsPolicyJSON = `{
  "programs": [
    {
      "name": "plain-inject-with-truncate-fields",
      "models": ["*"],
      "steps": [
        {
          "hook": "responses.before_upstream",
          "do": [
            {
              "action": "context.inject",
              "role": "system",
              "position": "after_existing_system",
              "keep_last_messages": 0,
              "text": "Plain inject should not carry truncate fields."
            }
          ]
        }
      ]
    }
  ]
}`

const ledgerSummaryWithTruncateFieldsPolicyJSON = `{
  "programs": [
    {
      "name": "ledger-summary-with-truncate-fields",
      "models": ["*"],
      "steps": [
        {
          "hook": "responses.compact.before_upstream",
          "do": [
            {
              "action": "context.inject_ledger_summary",
              "role": "system",
              "position": "after_existing_system",
              "ledger_ref": "ledger://project/session",
              "preserve_roles": ["system"],
              "text": "Gateway Harness project memory."
            }
          ]
        }
      ]
    }
  ]
}`

const truncateWithInjectFieldsPolicyJSON = `{
  "programs": [
    {
      "name": "truncate-with-inject-fields",
      "models": ["*"],
      "steps": [
        {
          "hook": "responses.before_upstream",
          "do": [
            {
              "action": "context.truncate",
              "role": "system",
              "text": "Truncate should not carry inject text.",
              "ledger_ref": "ledger://project/session",
              "keep_last_messages": 8
            }
          ]
        }
      ]
    }
  ]
}`

const continuityDropPolicyDryRunJSON = `{
  "programs": [
    {
      "name": "continuity-drop",
      "models": ["*"],
      "steps": [
        {
          "hook": "context.continuity_drop.detected",
          "when": {"context_continuity_drop": true},
          "do": [
            {
              "action": "context.inject_ledger_summary",
              "role": "system",
              "position": "after_existing_system",
              "ledger_ref": "ledger://project/session/current",
              "artifact_refs": ["artifact_continuity_drop_1"],
              "reason": "project continuity capsule after detected context drop",
              "text": "Project continuity capsule: preserve explicit goals and unresolved tasks."
            }
          ]
        }
      ]
    }
  ]
}`

const chatRequestJSON = `{
  "model": "gpt-5.4-mini",
  "messages": [
    {"role": "system", "content": "Existing system."},
    {"role": "user", "content": "Continue."}
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

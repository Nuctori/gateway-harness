package policy

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestApplyInjectsRequestAndRedactsTrace(t *testing.T) {
	p, err := Decode(strings.NewReader(responsesPolicyDryRunJSON))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	result, err := Apply(p, []byte(statefulResponsesRequestJSON), ApplyOptions{Hook: "responses.before_upstream"})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(result.AppliedActions) != 1 || result.Trace.Summary.Ops != 1 {
		t.Fatalf("unexpected apply result: %+v", result)
	}
	if strings.Contains(mustMarshalApplyResult(t, result), "Preserve Responses state.") {
		t.Fatalf("apply result leaked raw injected text: %s", mustMarshalApplyResult(t, result))
	}

	var obj map[string]any
	if err := json.Unmarshal(result.Request, &obj); err != nil {
		t.Fatalf("decode applied request: %v", err)
	}
	input := obj["input"].([]any)
	if len(input) != 4 {
		t.Fatalf("expected injected input item: %+v", input)
	}
	injected := input[2].(map[string]any)
	if injected["role"] != "system" || injected["content"] != "Preserve Responses state." {
		t.Fatalf("unexpected injected item: %+v", injected)
	}
	if input[0].(map[string]any)["type"] != "item_reference" || input[1].(map[string]any)["type"] != "function_call_output" {
		t.Fatalf("responses tool-chain prefix was not preserved: %+v", input)
	}
}

func TestApplySkipsDestructiveTruncate(t *testing.T) {
	p, err := Decode(strings.NewReader(policyDryRunWithTruncateJSON))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	result, err := Apply(p, []byte(chatRequestJSON), ApplyOptions{Hook: "chat.before_upstream", EstimatedTokens: 200})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(result.SkippedActions) != 1 || result.SkippedActions[0].Action != "context.truncate" {
		t.Fatalf("expected skipped truncate: %+v", result.SkippedActions)
	}
}

func TestApplyLedgerSummaryInjectsRequestAndRedactsTrace(t *testing.T) {
	p, err := Decode(strings.NewReader(ledgerSummaryPolicyDryRunJSON))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	result, err := Apply(p, []byte(statefulResponsesRequestJSON), ApplyOptions{Hook: "responses.compact.before_upstream"})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(result.AppliedActions) != 1 || result.AppliedActions[0] != "context.inject_ledger_summary" {
		t.Fatalf("unexpected applied actions: %+v", result.AppliedActions)
	}
	if len(result.Trace.Operations) != 1 {
		t.Fatalf("expected one trace operation: %+v", result.Trace.Operations)
	}
	operation := result.Trace.Operations[0]
	if operation.Source != "ledger.summary" || operation.LedgerRef == "" {
		t.Fatalf("missing ledger trace source: %+v", operation)
	}
	if strings.Contains(mustMarshalApplyResult(t, result), "Gateway Harness project memory") {
		t.Fatalf("apply result leaked raw ledger summary: %s", mustMarshalApplyResult(t, result))
	}

	var obj map[string]any
	if err := json.Unmarshal(result.Request, &obj); err != nil {
		t.Fatalf("decode applied request: %v", err)
	}
	input := obj["input"].([]any)
	injected := input[2].(map[string]any)
	if injected["role"] != "system" || !strings.Contains(injected["content"].(string), "Gateway Harness project memory") {
		t.Fatalf("unexpected injected ledger summary: %+v", injected)
	}
	if input[0].(map[string]any)["type"] != "item_reference" || input[1].(map[string]any)["type"] != "function_call_output" {
		t.Fatalf("responses tool-chain prefix was not preserved: %+v", input)
	}
}

func TestApplyContinuityDropCanBePassedAsExplicitOption(t *testing.T) {
	p, err := Decode(strings.NewReader(continuityDropPolicyDryRunJSON))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	result, err := Apply(p, []byte(statefulResponsesRequestJSON), ApplyOptions{
		Hook:                  "responses.before_upstream",
		ContextContinuityDrop: true,
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(result.AppliedActions) != 1 || result.AppliedActions[0] != "context.inject_ledger_summary" {
		t.Fatalf("expected continuity drop apply: %+v", result.AppliedActions)
	}
	if strings.Contains(mustMarshalApplyResult(t, result), "Project continuity capsule") {
		t.Fatalf("apply result leaked raw continuity capsule: %s", mustMarshalApplyResult(t, result))
	}
}

func mustMarshalApplyResult(t *testing.T, result ApplyResult) string {
	t.Helper()
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal apply result: %v", err)
	}
	return string(encoded)
}

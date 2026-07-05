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

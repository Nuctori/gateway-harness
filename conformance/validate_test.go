package conformance

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Nuctori/gateway-harness/policy"
)

func TestValidateAcceptsResponsesToolChainFixture(t *testing.T) {
	fixture, err := Decode(strings.NewReader(responsesToolChainFixtureJSON))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if err := Validate(fixture); err != nil {
		t.Fatalf("validate: %v", err)
	}
}

func TestValidateRejectsMissingResponsesToolChainGuard(t *testing.T) {
	fixture, err := Decode(strings.NewReader(responsesToolChainFixtureJSON))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	fixture.Adapter.Guards = []string{"explicit_mutation_only", "redacted_trace"}
	if err := Validate(fixture); err == nil {
		t.Fatal("expected missing responses tool-chain guard error")
	}
}

func TestValidateRejectsUnsupportedPolicyHook(t *testing.T) {
	fixture, err := Decode(strings.NewReader(strings.Replace(responsesToolChainFixtureJSON, `"hook": "responses.before_upstream"`, `"hook": "responses.compact.before_upstream"`, 1)))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if err := Validate(fixture); err == nil {
		t.Fatal("expected unsupported policy hook error")
	}
}

func TestReplayFakeUpstreamAcceptsResponsesToolChainFixture(t *testing.T) {
	fixture, err := Decode(strings.NewReader(responsesToolChainFixtureJSON))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	result, err := ReplayFakeUpstream(context.Background(), fixture)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if result.Path != "/v1/responses" || result.StatusCode != 200 || result.RequestBody == 0 {
		t.Fatalf("unexpected replay result: %+v", result)
	}
}

func TestReplayFakeUpstreamRejectsMissingItemReference(t *testing.T) {
	fixture, err := Decode(strings.NewReader(responsesToolChainFixtureJSON))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	fixture.Request = []byte(`{
		"model": "gpt-5.4-mini",
		"previous_response_id": "resp_1",
		"input": [
			{"type": "function_call_output", "call_id": "call_1", "output": "{\"ok\":true}"}
		]
	}`)
	if _, err := ReplayFakeUpstream(context.Background(), fixture); err == nil {
		t.Fatal("expected fake upstream replay to reject missing item_reference")
	}
}

func TestReplayAppliedPolicyFakeUpstreamRedactsTrace(t *testing.T) {
	fixture, err := Decode(strings.NewReader(responsesPolicyApplyFixtureJSON))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	result, err := ReplayAppliedPolicyFakeUpstream(context.Background(), fixture, policy.ApplyOptions{Hook: "responses.before_upstream"})
	if err != nil {
		t.Fatalf("policy replay: %v", err)
	}
	if result.StatusCode != 200 || result.AppliedRequestBody == 0 || result.OriginalRequestBody == 0 {
		t.Fatalf("unexpected policy replay result: %+v", result)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal policy replay result: %v", err)
	}
	if strings.Contains(string(encoded), "Preserve Responses state.") {
		t.Fatalf("policy replay leaked raw injected content: %s", encoded)
	}
	if len(result.Trace.Operations) != 1 || result.Trace.Operations[0].InsertIndex != 2 {
		t.Fatalf("expected one trace operation after tool-chain prefix: %+v", result.Trace)
	}
}

const responsesToolChainFixtureJSON = `{
  "name": "newapi-responses-tool-chain-preserved",
  "request_shape": "responses",
  "required_guards": ["explicit_mutation_only", "preserve_responses_tool_chain", "redacted_trace"],
  "adapter": {
    "version": "0.2",
    "adapter": "newapi",
    "hooks": ["responses.before_upstream"],
    "actions": ["context.truncate"],
    "request_shapes": ["responses"],
    "guards": ["explicit_mutation_only", "preserve_responses_tool_chain", "redacted_trace"]
  },
  "policy": {
    "version": "0.2",
    "programs": [{
      "name": "preserve-responses-tool-chain",
      "models": ["*"],
      "steps": [{
        "hook": "responses.before_upstream",
        "do": [{
          "action": "context.truncate",
          "keep_last_messages": 1,
          "preserve_roles": ["system"]
        }]
      }]
    }]
  },
  "request": {
    "model": "gpt-5.4-mini",
    "previous_response_id": "resp_1",
    "input": [
      {"type": "item_reference", "id": "fc_1"},
      {"type": "function_call_output", "call_id": "call_1", "output": "{\"ok\":true}"},
      {"role": "user", "content": "continue"}
    ]
  }
}`

const responsesPolicyApplyFixtureJSON = `{
  "name": "newapi-responses-policy-apply-redacted",
  "request_shape": "responses",
  "required_guards": ["explicit_mutation_only", "preserve_responses_tool_chain", "redacted_trace"],
  "adapter": {
    "version": "0.2",
    "adapter": "newapi",
    "hooks": ["responses.before_upstream"],
    "actions": ["context.inject"],
    "request_shapes": ["responses"],
    "guards": ["explicit_mutation_only", "preserve_responses_tool_chain", "redacted_trace"]
  },
  "policy": {
    "version": "0.2",
    "programs": [{
      "name": "apply-responses-context",
      "models": ["*"],
      "steps": [{
        "hook": "responses.before_upstream",
        "do": [{
          "action": "context.inject",
          "role": "system",
          "position": "after_existing_system",
          "text": "Preserve Responses state.",
          "reason": "state guardrail"
        }]
      }]
    }]
  },
  "request": {
    "model": "gpt-5.4-mini",
    "previous_response_id": "resp_1",
    "input": [
      {"type": "item_reference", "id": "fc_1"},
      {"type": "function_call_output", "call_id": "call_1", "output": "{\"ok\":true}"},
      {"role": "user", "content": "continue"}
    ]
  }
}`

package runtime

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/nuctori/gateway-harness/core/event"
	"github.com/nuctori/gateway-harness/core/policy"
)

func TestFallbackSequenceResolvesNextModel(t *testing.T) {
	p := policy.Policy{
		Name: "coding",
		Scope: policy.Scope{
			Models: []string{"gpt-5.4", "gpt-5.5"},
			Tags:   []string{"domain:coding"},
		},
		Hooks: map[string][]policy.Rule{
			"upstream.error": {
				{
					If: policy.Condition{
						StatusIn:        []int{429},
						MessageContains: []string{"Too Many Requests"},
					},
					Action: "fallback.sequence",
					Models: []string{"gpt-5.4", "gpt-5.5"},
				},
			},
		},
	}
	ev := event.Event{
		Type:  event.TypeUpstreamError,
		Model: "gpt-5.4",
		Tags:  []string{"domain:coding"},
		Error: &event.ErrorInfo{Status: 429, Message: "exceeded retry limit, last status: 429 Too Many Requests"},
	}
	got, err := NewEngine(p).Evaluate(ev)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Decision.Actions) != 1 {
		t.Fatalf("expected one action, got %d", len(got.Decision.Actions))
	}
	action := got.Decision.Actions[0]
	if action.Type != "retry.with_model" || action.FromModel != "gpt-5.4" || action.ToModel != "gpt-5.5" {
		t.Fatalf("unexpected fallback action: %+v", action)
	}
	if got.Trace.Outcomes[0].Type != "model.switched" {
		t.Fatalf("expected model.switched outcome, got %+v", got.Trace.Outcomes[0])
	}
}

func TestPromptAppendSystemDecision(t *testing.T) {
	p := policy.Policy{
		Name:  "prompt",
		Scope: policy.Scope{Models: []string{"gpt-5.4"}},
		Hooks: map[string][]policy.Rule{
			"request.pre_prompt": {
				{Action: "prompt.append_system", Text: "coding prompt"},
			},
		},
	}
	got, err := NewEngine(p).Evaluate(event.Event{Type: event.TypeRequestPrePrompt, Model: "gpt-5.4"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Decision.Actions[0].Type != "prompt.append_system" || got.Decision.Actions[0].Text != "coding prompt" {
		t.Fatalf("unexpected prompt decision: %+v", got.Decision.Actions[0])
	}
}

func TestContextHarnessWildcardAppliesToDifferentModels(t *testing.T) {
	p := contextPolicy(16, 1200)
	models := []string{"gpt-5.4", "kimi-for-coding"}
	for _, model := range models {
		t.Run(model, func(t *testing.T) {
			got, err := NewEngine(p).Evaluate(contextEvent(model, 8200))
			if err != nil {
				t.Fatal(err)
			}
			if got.Decision.ContextPatch == nil {
				t.Fatal("expected context patch")
			}
			if got.Decision.ContextPatch.Summary.Ops != 1 {
				t.Fatalf("expected one patch op, got %+v", got.Decision.ContextPatch.Summary)
			}
			if got.Trace.Decisions[0].PatchSummary.ContentMode != "redacted" {
				t.Fatalf("expected redacted trace, got %+v", got.Trace.Decisions[0].PatchSummary)
			}
		})
	}
}

func TestContextHarnessAddsTruncateWhenOverBudget(t *testing.T) {
	got, err := NewEngine(contextPolicy(16, 1200)).Evaluate(contextEvent("gpt-5.4", 28000))
	if err != nil {
		t.Fatal(err)
	}
	if got.Decision.ContextPatch == nil || got.Decision.ContextPatch.Summary.Ops != 2 {
		t.Fatalf("expected inject and truncate patch ops, got %+v", got.Decision.ContextPatch)
	}
	if got.Decision.ContextPatch.Operations[1].Op != "truncate" {
		t.Fatalf("expected truncate op, got %+v", got.Decision.ContextPatch.Operations[1])
	}
}

func TestContextHarnessPatchOpLimit(t *testing.T) {
	_, err := NewEngine(contextPolicy(1, 1200)).Evaluate(contextEvent("gpt-5.4", 28000))
	if err == nil || !strings.Contains(err.Error(), "op limit") {
		t.Fatalf("expected op limit error, got %v", err)
	}
}

func TestContextHarnessAddedTokenBudget(t *testing.T) {
	_, err := NewEngine(contextPolicy(16, 1)).Evaluate(contextEvent("gpt-5.4", 8200))
	if err == nil || !strings.Contains(err.Error(), "token budget") {
		t.Fatalf("expected token budget error, got %v", err)
	}
}

func TestTraceDoesNotPersistRawContextContent(t *testing.T) {
	got, err := NewEngine(contextPolicy(16, 1200)).Evaluate(contextEvent("gpt-5.4", 8200))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(got.Trace)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "Preserve user intent") {
		t.Fatalf("trace leaked raw context: %s", raw)
	}
}

func contextPolicy(maxPatchOps int, maxAddedTokens int64) policy.Policy {
	return policy.Policy{
		Name: "any-model-context-harness",
		Scope: policy.Scope{
			Models: []string{"*"},
			Tags:   []string{"domain:coding"},
		},
		Hooks: map[string][]policy.Rule{
			"request.pre_context": {
				{
					Program: &policy.ContextProgram{
						Budget: policy.ProgramBudget{
							MaxPatchOps:    maxPatchOps,
							MaxAddedTokens: maxAddedTokens,
						},
						Steps: []policy.ProgramStep{
							{
								When: policy.Condition{ModelMatches: "*"},
								Do: []policy.Action{
									{
										Action:   "context.inject",
										Role:     "system",
										Position: "after_existing_system",
										Text:     "Preserve user intent and prior architecture decisions.",
									},
								},
							},
							{
								When: policy.Condition{EstimatedTokensGT: 24000},
								Do: []policy.Action{
									{
										Action:             "context.truncate",
										Strategy:           "oldest_user_assistant_pairs",
										KeepLastMessages:   12,
										PreserveRoles:      []string{"system", "developer"},
										MaxEstimatedTokens: 24000,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func contextEvent(model string, estimated int64) event.Event {
	return event.Event{
		Type:  event.TypeRequestPreContext,
		Model: model,
		Tags:  []string{"domain:coding"},
		Context: &event.Context{
			MaxTokens:       128000,
			EstimatedTokens: estimated,
			TokenEstimator:  "char_approx_v1",
			Messages: []event.Message{
				{Role: "system", Content: "You are helpful."},
				{Role: "user", Content: "Review this patch."},
			},
		},
	}
}

package newapi

import (
	"context"
	"testing"

	"github.com/nuctori/gateway-harness/core/event"
	"github.com/nuctori/gateway-harness/core/plan"
	gharness "github.com/nuctori/gateway-harness/sdk/go"
)

// Compile-time interface compliance checks.
// Any codec implementing Adapter must satisfy these interfaces.
var _ gharness.Adapter = Adapter{}
var _ gharness.ContextAdapter = Adapter{}

// ── Adapter Capability contract ─────────────────────────────────

func TestCapabilityContract(t *testing.T) {
	cap := Adapter{}.Capability()
	if len(cap.Events) == 0 {
		t.Error("Capability.Events must be non-empty")
	}
	if len(cap.Actions) == 0 {
		t.Error("Capability.Actions must be non-empty")
	}
	if len(cap.Outcomes) == 0 {
		t.Error("Capability.Outcomes must be non-empty")
	}
	for _, e := range cap.Events {
		if e == "" {
			t.Error("Capability: empty event name")
		}
	}
	for _, a := range cap.Actions {
		if a == "" {
			t.Error("Capability: empty action name")
		}
	}
	for _, o := range cap.Outcomes {
		if o == "" {
			t.Error("Capability: empty outcome name")
		}
	}
}

// ── Adapter BuildEvent contract ────────────────────────────────

func TestAdapterBuildEventContract(t *testing.T) {
	input := gharness.AdapterInput{
		RequestID: "test-req",
		TraceID:   "test-trace",
		Model:     "gpt-5.4",
		Tags:      []string{"domain:coding"},
	}
	ev, err := Adapter{}.BuildEvent(context.Background(), input)
	if err != nil {
		t.Fatalf("BuildEvent: %v", err)
	}
	// Identity fields must be preserved.
	if ev.RequestID != input.RequestID {
		t.Errorf("RequestID = %q, want %q", ev.RequestID, input.RequestID)
	}
	if ev.TraceID != input.TraceID {
		t.Errorf("TraceID = %q, want %q", ev.TraceID, input.TraceID)
	}
	if ev.Model != input.Model {
		t.Errorf("Model = %q, want %q", ev.Model, input.Model)
	}
	if len(ev.Tags) != 1 || ev.Tags[0] != "domain:coding" {
		t.Errorf("Tags = %v", ev.Tags)
	}
	// Event type must be set (not empty).
	if ev.Type == "" {
		t.Error("Event.Type must be non-empty")
	}
}

// ── Adapter ApplyDecision contract ─────────────────────────────

func TestApplyDecisionSkippedContract(t *testing.T) {
	out, err := Adapter{}.ApplyDecision(context.Background(), plan.Decision{
		SkippedReason: "no matching rule",
	})
	if err != nil {
		t.Fatalf("ApplyDecision(skipped): %v", err)
	}
	if out.Type != "decision.skipped" {
		t.Errorf("Type = %q, want %q", out.Type, "decision.skipped")
	}
	if out.Reason != "no matching rule" {
		t.Errorf("Reason = %q", out.Reason)
	}
}

func TestApplyDecisionPromptAppendContract(t *testing.T) {
	out, err := Adapter{}.ApplyDecision(context.Background(), plan.Decision{
		Actions: []plan.Action{{Type: "prompt.append_system"}},
	})
	if err != nil {
		t.Fatalf("ApplyDecision(prompt): %v", err)
	}
	if out.Type != "decision.applied" {
		t.Errorf("Type = %q, want %q", out.Type, "decision.applied")
	}
}

func TestApplyDecisionRetryWithTargetContract(t *testing.T) {
	out, err := Adapter{}.ApplyDecision(context.Background(), plan.Decision{
		Actions: []plan.Action{{
			Type: "retry.with_model", FromModel: "gpt-5.4", ToModel: "gpt-5.5",
		}},
	})
	if err != nil {
		t.Fatalf("ApplyDecision(retry): %v", err)
	}
	if out.Type != "model.switched" {
		t.Errorf("Type = %q, want %q", out.Type, "model.switched")
	}
	if out.FromModel != "gpt-5.4" || out.ToModel != "gpt-5.5" {
		t.Errorf("FromModel/ToModel = %s/%s", out.FromModel, out.ToModel)
	}
}

func TestApplyDecisionRetryExhaustedContract(t *testing.T) {
	out, err := Adapter{}.ApplyDecision(context.Background(), plan.Decision{
		Actions: []plan.Action{{
			Type: "retry.with_model", FromModel: "gpt-5.4",
		}},
	})
	if err != nil {
		t.Fatalf("ApplyDecision(exhausted): %v", err)
	}
	if out.Type != "retry.exhausted" {
		t.Errorf("Type = %q, want %q", out.Type, "retry.exhausted")
	}
}

func TestApplyDecisionUnsupportedActionContract(t *testing.T) {
	_, err := Adapter{}.ApplyDecision(context.Background(), plan.Decision{
		Actions: []plan.Action{{Type: "nonexistent"}},
	})
	if err == nil {
		t.Error("expected error for unsupported action, got nil")
	}
}

// ── ContextAdapter BuildContextEvent contract ──────────────────

func TestBuildContextEventContract(t *testing.T) {
	input := gharness.AdapterInput{
		RequestID: "ctx-test",
		TraceID:   "ctx-trace",
		Model:     "kimi",
		Tags:      []string{"domain:coding"},
		Payload:   event.Context{MaxTokens: 128000, EstimatedTokens: 5000},
	}
	ev, err := Adapter{}.BuildContextEvent(context.Background(), input)
	if err != nil {
		t.Fatalf("BuildContextEvent: %v", err)
	}
	if ev.Type != event.TypeRequestPreContext {
		t.Errorf("Type = %q, want %q", ev.Type, event.TypeRequestPreContext)
	}
	if ev.RequestID != input.RequestID {
		t.Errorf("RequestID = %q", ev.RequestID)
	}
	if ev.Context == nil {
		t.Fatal("Context must not be nil")
	}
	if ev.Context.MaxTokens != 128000 {
		t.Errorf("Context.MaxTokens = %d", ev.Context.MaxTokens)
	}
}

func TestBuildContextEventInvalidPayloadContract(t *testing.T) {
	input := gharness.AdapterInput{Payload: "not a context"}
	_, err := Adapter{}.BuildContextEvent(context.Background(), input)
	if err == nil {
		t.Error("expected error for invalid payload, got nil")
	}
}

// ── ContextAdapter ApplyContextPatch contract ──────────────────

func TestApplyContextPatchRejectedContract(t *testing.T) {
	out, err := Adapter{}.ApplyContextPatch(context.Background(), plan.ContextPatch{})
	if err != nil {
		t.Fatalf("ApplyContextPatch(empty): %v", err)
	}
	if out.Type != "context.patch_rejected" {
		t.Errorf("Type = %q, want %q", out.Type, "context.patch_rejected")
	}
}

func TestApplyContextPatchAppliedContract(t *testing.T) {
	patch := plan.ContextPatch{
		Operations: []plan.PatchOperation{{Op: "append", Target: "messages"}},
		Summary:    plan.PatchSummary{Ops: 1},
	}
	out, err := Adapter{}.ApplyContextPatch(context.Background(), patch)
	if err != nil {
		t.Fatalf("ApplyContextPatch(valid): %v", err)
	}
	if out.Type != "context.patch_applied" {
		t.Errorf("Type = %q, want %q", out.Type, "context.patch_applied")
	}
}

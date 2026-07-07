package codex

import (
	"context"
	"testing"

	"github.com/nuctori/gateway-harness/core/event"
	"github.com/nuctori/gateway-harness/core/plan"
	gharness "github.com/nuctori/gateway-harness/sdk/go"
)

var _ gharness.Adapter = Adapter{}
var _ gharness.ContextAdapter = Adapter{}

func TestBuildEventFromUserPromptSubmit(t *testing.T) {
	ev, err := Adapter{}.BuildEvent(context.Background(), gharness.AdapterInput{
		RequestID: "req-1",
		TraceID:   "trace-1",
		Model:     "gpt-5.4",
		Tags:      []string{"domain:coding"},
		Payload:   HookInput{Hook: HookUserPromptSubmit},
	})
	if err != nil {
		t.Fatalf("BuildEvent: %v", err)
	}
	if ev.Type != event.TypeRequestPrePrompt {
		t.Fatalf("Type = %q", ev.Type)
	}
	if ev.RequestID != "req-1" || ev.TraceID != "trace-1" || ev.Model != "gpt-5.4" {
		t.Fatalf("unexpected identity fields: %+v", ev)
	}
}

func TestBuildContextEventFromPreCompact(t *testing.T) {
	ev, err := Adapter{}.BuildContextEvent(context.Background(), gharness.AdapterInput{
		RequestID: "req-2",
		TraceID:   "trace-2",
		Model:     "gpt-5.4",
		Payload: HookInput{
			Hook:    HookPreCompact,
			Context: &event.Context{EstimatedTokens: 8200},
		},
	})
	if err != nil {
		t.Fatalf("BuildContextEvent: %v", err)
	}
	if ev.Type != event.TypeRequestPreContext {
		t.Fatalf("Type = %q", ev.Type)
	}
	if ev.Context == nil || ev.Context.EstimatedTokens != 8200 {
		t.Fatalf("unexpected context: %+v", ev.Context)
	}
}

func TestUnsupportedHookRejected(t *testing.T) {
	_, err := Adapter{}.BuildEvent(context.Background(), gharness.AdapterInput{Payload: HookInput{Hook: "PreToolUse"}})
	if err == nil {
		t.Fatal("expected unsupported hook error")
	}
}

func TestApplyDecision(t *testing.T) {
	out, err := Adapter{}.ApplyDecision(context.Background(), plan.Decision{Actions: []plan.Action{{Type: "prompt.append_system"}}})
	if err != nil {
		t.Fatalf("ApplyDecision: %v", err)
	}
	if out.Type != "decision.applied" {
		t.Fatalf("Type = %q", out.Type)
	}
}

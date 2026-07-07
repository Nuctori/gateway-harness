package opencode

import (
	"context"
	"testing"

	"github.com/nuctori/gateway-harness/core/event"
	"github.com/nuctori/gateway-harness/core/plan"
	gharness "github.com/nuctori/gateway-harness/sdk/go"
)

var _ gharness.Adapter = Adapter{}
var _ gharness.ContextAdapter = Adapter{}

func TestBuildEventFromPromptAppend(t *testing.T) {
	ev, err := Adapter{}.BuildEvent(context.Background(), gharness.AdapterInput{
		RequestID: "req-1",
		TraceID:   "trace-1",
		Model:     "gpt-5.4",
		Payload:   HookInput{Hook: HookPromptAppend},
	})
	if err != nil {
		t.Fatalf("BuildEvent: %v", err)
	}
	if ev.Type != event.TypeRequestPrePrompt {
		t.Fatalf("Type = %q", ev.Type)
	}
}

func TestBuildContextEventFromSessionCompacting(t *testing.T) {
	ev, err := Adapter{}.BuildContextEvent(context.Background(), gharness.AdapterInput{
		RequestID: "req-2",
		TraceID:   "trace-2",
		Model:     "gpt-5.4",
		Payload: HookInput{
			Hook:    HookSessionCompacting,
			Context: &event.Context{EstimatedTokens: 8200},
		},
	})
	if err != nil {
		t.Fatalf("BuildContextEvent: %v", err)
	}
	if ev.Type != event.TypeRequestPreContext {
		t.Fatalf("Type = %q", ev.Type)
	}
}

func TestUnsupportedHookRejected(t *testing.T) {
	_, err := Adapter{}.BuildEvent(context.Background(), gharness.AdapterInput{Payload: HookInput{Hook: "session.created"}})
	if err == nil {
		t.Fatal("expected unsupported hook error")
	}
}

func TestApplyContextPatch(t *testing.T) {
	out, err := Adapter{}.ApplyContextPatch(context.Background(), plan.ContextPatch{Operations: []plan.PatchOperation{{Op: "append", Target: "messages"}}})
	if err != nil {
		t.Fatalf("ApplyContextPatch: %v", err)
	}
	if out.Type != "context.patch_applied" {
		t.Fatalf("Type = %q", out.Type)
	}
}

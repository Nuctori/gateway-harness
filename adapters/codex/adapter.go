package codex

import (
	"context"
	"fmt"

	"github.com/nuctori/gateway-harness/core/event"
	"github.com/nuctori/gateway-harness/core/plan"
	"github.com/nuctori/gateway-harness/core/trace"
	gharness "github.com/nuctori/gateway-harness/sdk/go"
)

const (
	HookUserPromptSubmit = "UserPromptSubmit"
	HookPreCompact       = "PreCompact"
)

type HookInput struct {
	Hook     string            `json:"hook,omitempty" yaml:"hook,omitempty"`
	Context  *event.Context    `json:"context,omitempty" yaml:"context,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

type Adapter struct{}

func (Adapter) Capability() gharness.Capability {
	return gharness.Capability{
		SourceHooks: []string{HookUserPromptSubmit, HookPreCompact},
		Events:      []string{string(event.TypeRequestPrePrompt), string(event.TypeRequestPreContext)},
		Actions:     []string{"prompt.append_system", "context.patch"},
		Outcomes:    []string{"decision.applied", "context.patch_applied", "context.patch_rejected"},
	}
}

func (Adapter) BuildEvent(_ context.Context, input gharness.AdapterInput) (event.Event, error) {
	return buildEvent(input, false)
}

func (Adapter) ApplyDecision(_ context.Context, decision plan.Decision) (trace.Outcome, error) {
	if len(decision.Actions) == 0 && decision.ContextPatch == nil {
		return trace.Outcome{Type: "decision.skipped", Reason: decision.SkippedReason}, nil
	}
	if decision.ContextPatch != nil {
		return trace.Outcome{Type: "context.patch_applied"}, nil
	}
	action := decision.Actions[0]
	switch action.Type {
	case "prompt.append_system":
		return trace.Outcome{Type: "decision.applied"}, nil
	default:
		return trace.Outcome{}, fmt.Errorf("unsupported decision action %q", action.Type)
	}
}

func (Adapter) BuildContextEvent(_ context.Context, input gharness.AdapterInput) (event.Event, error) {
	return buildEvent(input, true)
}

func (Adapter) ApplyContextPatch(_ context.Context, patch plan.ContextPatch) (trace.Outcome, error) {
	if len(patch.Operations) == 0 {
		return trace.Outcome{Type: "context.patch_rejected", Reason: "empty patch"}, nil
	}
	return trace.Outcome{Type: "context.patch_applied"}, nil
}

func buildEvent(input gharness.AdapterInput, contextOnly bool) (event.Event, error) {
	payload, err := hookInput(input.Payload)
	if err != nil {
		return event.Event{}, err
	}
	if payload.Hook == HookUserPromptSubmit {
		if contextOnly {
			return event.Event{}, fmt.Errorf("hook %q is not a context event", payload.Hook)
		}
		return event.Event{
			Type:      event.TypeRequestPrePrompt,
			RequestID: input.RequestID,
			TraceID:   input.TraceID,
			Model:     input.Model,
			Tags:      input.Tags,
			Metadata:  payload.Metadata,
		}, nil
	}
	if payload.Hook == HookPreCompact {
		if payload.Context == nil {
			return event.Event{}, fmt.Errorf("hook %q requires a context payload", payload.Hook)
		}
		return event.Event{
			Type:      event.TypeRequestPreContext,
			RequestID: input.RequestID,
			TraceID:   input.TraceID,
			Model:     input.Model,
			Tags:      input.Tags,
			Context:   payload.Context,
			Metadata:  payload.Metadata,
		}, nil
	}
	return event.Event{}, fmt.Errorf("unsupported codex hook %q", payload.Hook)
}

func hookInput(payload any) (HookInput, error) {
	switch v := payload.(type) {
	case HookInput:
		return v, nil
	case *HookInput:
		if v == nil {
			return HookInput{}, fmt.Errorf("payload is nil *codex.HookInput")
		}
		return *v, nil
	default:
		return HookInput{}, fmt.Errorf("payload is not codex.HookInput (got %T)", payload)
	}
}

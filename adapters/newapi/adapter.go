package newapi

import (
	"context"
	"fmt"

	"github.com/nuctori/gateway-harness/core/event"
	"github.com/nuctori/gateway-harness/core/plan"
	"github.com/nuctori/gateway-harness/core/trace"
	gharness "github.com/nuctori/gateway-harness/sdk/go"
)

type Adapter struct{}

func (Adapter) Capability() gharness.Capability {
	return gharness.Capability{
		Events: []string{
			"request.pre_prompt",
			"upstream.error",
			"request.pre_context",
		},
		Actions: []string{
			"prompt.append_system",
			"retry.with_model",
			"fallback.sequence",
			"context.patch",
		},
		Outcomes: []string{
			"decision.applied",
			"model.switched",
			"retry.exhausted",
			"context.patch_applied",
			"context.patch_rejected",
		},
	}
}

func (Adapter) BuildEvent(_ context.Context, input gharness.AdapterInput) (event.Event, error) {
	return event.Event{
		Type:      event.TypeRequestPrePrompt,
		RequestID: input.RequestID,
		TraceID:   input.TraceID,
		Model:     input.Model,
		Tags:      input.Tags,
	}, nil
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
	case "retry.with_model":
		if action.ToModel == "" {
			return trace.Outcome{Type: "retry.exhausted", FromModel: action.FromModel}, nil
		}
		return trace.Outcome{Type: "model.switched", FromModel: action.FromModel, ToModel: action.ToModel}, nil
	default:
		return trace.Outcome{}, fmt.Errorf("unsupported decision action %q", action.Type)
	}
}

func (Adapter) BuildContextEvent(_ context.Context, input gharness.AdapterInput) (event.Event, error) {
	ctx, ok := input.Payload.(event.Context)
	if !ok {
		return event.Event{}, fmt.Errorf("BuildContextEvent: payload is not event.Context (got %T)", input.Payload)
	}
	return event.Event{
		Type:      event.TypeRequestPreContext,
		RequestID: input.RequestID,
		TraceID:   input.TraceID,
		Model:     input.Model,
		Tags:      input.Tags,
		Context:   &ctx,
	}, nil
}

func (Adapter) ApplyContextPatch(_ context.Context, patch plan.ContextPatch) (trace.Outcome, error) {
	if len(patch.Operations) == 0 {
		return trace.Outcome{Type: "context.patch_rejected", Reason: "empty patch"}, nil
	}
	return trace.Outcome{Type: "context.patch_applied"}, nil
}

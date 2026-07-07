package gharness

import (
	"context"

	"github.com/nuctori/gateway-harness/core/event"
	"github.com/nuctori/gateway-harness/core/plan"
	"github.com/nuctori/gateway-harness/core/trace"
)

type Capability struct {
	Events   []string
	Actions  []string
	Outcomes []string
}

type AdapterInput struct {
	RequestID string
	TraceID   string
	Model     string
	Tags      []string
	Payload   any
}

type Adapter interface {
	Capability() Capability
	BuildEvent(ctx context.Context, input AdapterInput) (event.Event, error)
	ApplyDecision(ctx context.Context, decision plan.Decision) (trace.Outcome, error)
}

type ContextAdapter interface {
	Adapter
	BuildContextEvent(ctx context.Context, input AdapterInput) (event.Event, error)
	ApplyContextPatch(ctx context.Context, patch plan.ContextPatch) (trace.Outcome, error)
}

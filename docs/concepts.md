# Gateway Harness Concepts

Gateway Harness is a programmable context layer for LLM gateways.

## Hook

A hook is a lifecycle phase where a policy can run.

Examples:

- `request.before_model_mapping`
- `request.before_upstream`
- `responses.compact.before_upstream`

Hooks should be real adapter phases. Do not expose a hook in UI or schema unless an adapter can
execute it.

## Action

An action is a context transformation primitive.

Current actions:

- `context.inject`: insert or append a context fragment.
- `context.inject_ledger_summary`: insert an explicit project/session ledger summary supplied by
  the adapter or operator.
- `context.truncate`: keep a recent tail of context and optionally preserve roles.

Actions are not product features. They are context programming operations that adapters execute
against host-specific request objects.

## Condition

Conditions decide whether a step runs after the hook is reached.

Current conditions:

- `model_matches`
- `estimated_tokens_gt`

## Explicit Guards

Gateway Harness does not define implicit program-level budgets. If an adapter needs to limit or
guard a mutation, that behavior should be represented by an explicit action or adapter guard. A
harness must not invent a hidden model context window or reject a request just because its own
estimate is smaller than the upstream model capacity.

Token estimates are inputs to explicit conditions, not hidden limits. A condition such as
`estimated_tokens_gt` should only match when the adapter or operator passed an explicit estimate for
the current dry-run or request trace.

## Trace

Adapters should emit redacted traces that include:

- matched program
- hook
- model
- action count
- added estimated tokens
- content hashes, not raw injected content

## Policy Dry-Run

Policy dry-run is a preflight check for ordinary policies. It reports matched programs, applied
actions, skipped actions, and redacted patch plans without mutating the request.

Dry-run must not print raw injected text or the full rewritten request. Destructive actions such as
`context.truncate` are reported as skipped. For Responses requests, insertion planning must preserve
the stateful protocol prefix (`item_reference`, `function_call`, `function_call_output`, and
`reasoning`) before inserting context messages.

Compact hooks are ordinary explicit hooks. A compaction-aware adapter may run
`responses.compact.before_upstream`, emit a ledger `compact` event, or call a configured steward, but
Gateway Harness core does not silently summarize, truncate, or recover context that the client did
not send.

`context.inject_ledger_summary` is the transparent version of a compaction sentinel. The policy must
name a `ledger_ref` and provide the summary text it intends to inject. Gateway Harness records only
the source, ledger reference, artifact references, content hash, and content length in dry-run and
trace output. It does not read hidden local state, call an AI, or invent a summary at runtime.

## Policy Apply Replay

Adapters need a stronger check than dry-run: the applied request copy must still satisfy the host
protocol shape. Policy apply replay applies a policy in memory, posts the applied request to a local
fake upstream, and reports only request sizes plus redacted trace metadata.

This is intentionally different from a command that prints the rewritten request. The applied
request can contain raw injected text because adapters need to send it upstream, but CLI and logs
should expose only trace operations, hashes, lengths, hooks, and status codes.

## Ledger

A ledger records project and session history for audit and later review.

Ledger entries should include event metadata, content hashes, and references to external artifacts.
They should not embed raw user prompts, raw model responses, or hidden summaries.
Metadata is for labels and IDs; obvious raw-content keys such as `prompt`, `response`, and
`messages` are rejected by the ledger validator.

Events can reference session artifacts through `artifact_refs`. For example, a `compact` event can
point at the exact `compact_summary` artifact that was produced or consumed for that compaction. The
ledger validator rejects references that do not exist in the same session, so audit trails stay
linked instead of relying on naming conventions.

Typical event types:

- `request`
- `response`
- `tool_call`
- `compact`
- `failover`
- `harness_action`
- `error`

Summarizers, stores, and indexes should live in sidecars or adapters. Gateway Harness core only
defines the transparent contract they can validate against.

## Steward

A steward is an explicit AI-in-the-loop sidecar that can propose context-management changes at
configured hooks.

Stewards are for cases such as:

- compact-time summaries
- stuck-session diagnosis
- failover context repair
- policy patch proposals

Stewards must not be implicit gateway behavior. A valid steward spec must declare explicit hooks, a
steward model, redacted inputs, allowed output actions, artifact types, and required guards.

Required safety boundaries:

- no wildcard hooks
- no raw transcript inputs
- structured outputs only
- output actions must be validated before application
- policy patch proposals require human approval
- artifacts must be referenced by hash

## Steward Proposal

A steward proposal is the structured output returned by a steward.

It must be validated against the steward spec that enabled it. This cross-check prevents an AI from
using a disabled hook, emitting an action that was not allowed, creating an artifact without a hash,
or turning a policy suggestion into an implicit runtime change.

Proposal outputs are action-shaped records, not free-form transcripts. For example:

- `context.inject` must include role, position, and text.
- `ledger.artifact.create` must include artifact type, content hash, and reference.
- `policy.patch.propose` must include patch hash, reference, and description.

Dry-run prints a redacted patch plan for non-destructive outputs. It must not print the full
rewritten request, contact an upstream, write persistent state, call an AI, or perform destructive
context edits such as truncate.

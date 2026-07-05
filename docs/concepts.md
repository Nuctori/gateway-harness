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

## Trace

Adapters should emit redacted traces that include:

- matched program
- hook
- model
- action count
- added estimated tokens
- content hashes, not raw injected content

## Ledger

A ledger records project and session history for audit and later review.

Ledger entries should include event metadata, content hashes, and references to external artifacts.
They should not embed raw user prompts, raw model responses, or hidden summaries.
Metadata is for labels and IDs; obvious raw-content keys such as `prompt`, `response`, and
`messages` are rejected by the ledger validator.

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

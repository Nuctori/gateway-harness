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

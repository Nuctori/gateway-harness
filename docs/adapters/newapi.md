# NewAPI Adapter Contract

NewAPI is an adapter and example host for Gateway Harness.

The adapter should publish `examples/newapi/adapter.capability.json` as the explicit contract used
by CLIs, WebUIs, and conformance tests.

Suggested hook mapping:

| Gateway Harness hook                    | NewAPI phase                                |
| --------------------------------------- | ------------------------------------------- |
| `chat.before_model_mapping`             | before `ModelMappedHelper` in chat relay    |
| `chat.before_upstream`                  | after model mapping, before upstream body   |
| `responses.before_model_mapping`        | before `ModelMappedHelper` in responses     |
| `responses.before_upstream`             | after model mapping, before upstream body   |
| `responses.compact.before_model_mapping` | compact endpoint before model mapping       |
| `responses.compact.before_upstream`     | compact endpoint before upstream body       |
| `context.continuity_drop.detected`      | virtual event during pre-upstream handling after a transparent context-drop preflight |
| `request.before_model_mapping`          | adapter alias for all request pre-map hooks |
| `request.before_upstream`               | adapter alias for all pre-upstream hooks    |

Adapter invariants:

- Unknown hooks and actions must fail validation before execution.
- Each step must declare `hook` or `hooks`; adapters must not invent a default execution phase.
- `context.inject` should be redacted in logs.
- Adapter-local guards must be explicit and must not behave like hidden model context limits.
- The adapter should disable pass-through body mode when a policy may mutate context.
- The continuity-drop hook must be driven by redacted token-count metadata only. It must not log raw
  prompts, call an AI implicitly, or mutate a request unless an operator-configured policy action
  matches.

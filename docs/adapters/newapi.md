# NewAPI Adapter Contract

NewAPI is an adapter and example host for Gateway Harness.

Suggested hook mapping:

| Gateway Harness hook                    | NewAPI phase                                |
| --------------------------------------- | ------------------------------------------- |
| `chat.before_model_mapping`             | before `ModelMappedHelper` in chat relay    |
| `chat.before_upstream`                  | after model mapping, before upstream body   |
| `responses.before_model_mapping`        | before `ModelMappedHelper` in responses     |
| `responses.before_upstream`             | after model mapping, before upstream body   |
| `responses.compact.before_model_mapping` | compact endpoint before model mapping       |
| `responses.compact.before_upstream`     | compact endpoint before upstream body       |
| `request.before_model_mapping`          | adapter alias for all request pre-map hooks |
| `request.before_upstream`               | adapter alias for all pre-upstream hooks    |

Adapter invariants:

- Unknown hooks and actions must fail validation before execution.
- Legacy steps without `hook` should default to `request.before_upstream`.
- `context.inject` should be redacted in logs.
- `max_context_tokens` is a deprecated compatibility field and should not be a gateway hard reject.
- The adapter should disable pass-through body mode when a policy may mutate context.


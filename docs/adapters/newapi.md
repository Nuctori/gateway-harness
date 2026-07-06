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

## Goal Gate integration

If NewAPI or an adjacent executor wants AI-in-the-loop completion review, keep it in an explicit
sidecar or host interceptor instead of hardcoding workflow into relay handlers.

Recommended flow:

1. The executor decides, by explicit config, that a completion attempt should trigger Goal Gate.
2. The host constructs a redacted `goal.before_complete` event using only allowed inputs such as
   `goal_state`, `work_summary`, `test_results`, `blockers`, `recent_trace`, `changed_files`,
   `verification_summary`, and `user_goal`.
3. The host calls `gateway-harness execute-goal-gate-sidecar <steward.json> <event.json> <audit.json> -- <runner>`
   or uses the in-process helpers `ReviewGoalCompletion`, `EvaluateGoalProposal`,
   `ApplyGoalReviewResult`, and `BuildGoalGateAppendRecord`.
4. The host consumes the returned `allow_complete`, `continue_work`, `continue_instruction`, and
   `next_goal_state` fields.
5. The host appends the returned redacted ledger record explicitly. No raw prompt content should be
   stored unless the operator separately opted into an artifact path outside the default ledger.

Suggested host-visible config shape:

```json
{
  "enabled": false,
  "hook": "goal.before_complete",
  "runner": {
    "command": "python",
    "args": ["examples/smolagents/goal_reviewer.py"]
  },
  "allowed_inputs": [
    "goal_state",
    "work_summary",
    "test_results",
    "blockers",
    "recent_trace",
    "changed_files",
    "verification_summary",
    "user_goal"
  ],
  "allowed_actions": [
    "goal.approve_complete",
    "goal.reject_complete",
    "goal.request_continue",
    "diagnosis.note.create",
    "ledger.artifact.create"
  ],
  "max_continue_attempts": 3,
  "cooldown_seconds": 60
}
```

This contract is intentionally explicit and can be surfaced directly in a WebUI using
`gateway-harness goal-gate-config-schema`.

The schema includes titles, descriptions, examples, and GUI-friendly enumerations for each allowed
input and action. A host console should prefer rendering those schema hints directly instead of
copying the catalog into separate hidden frontend constants.

For hosts that want a thinner path to a real settings page, `gateway-harness goal-gate-form-model`
returns a default section layout, control hints, and transparency copy on top of the same schema.
That model is optional and should stay a host-facing presentation helper, not a second execution
contract.

On the output side, `gateway-harness goal-gate-result-schema` publishes the explicit execution
result contract. Hosts can use it to wire approval banners, continue-work instructions, structured
failure handling, ledger append previews, and optional continuation-patch previews without
reverse-engineering runtime structs.

If a host prefers a single initialization fetch, `gateway-harness goal-gate-host-bundle` returns the
config schema, form schema, form model, result schema, and sample results in one payload. This is a
host convenience layer only; the underlying execution contract still comes from the validated config,
event, proposal, and result objects.

At runtime, this host config is not just documentation. It narrows the effective steward surface:
the hook must match, event inputs must stay inside `allowed_inputs`, and proposal actions must stay
inside `allowed_actions` even if the underlying steward spec declared a broader set.

`runner.workdir` is also explicit. Hosts should resolve relative values against the config file
location before invoking the runner, so execution does not depend on the caller's current shell
directory.

If execution fails, hosts should not silently pass completion. `execute-goal-gate` now returns a
structured failure result with `failure.stage`, `failure.code`, and `failure.message`; when audit
input is present, the result also includes an `append_record` for a redacted ledger `error` event.
Hosts can print or persist that result, then exit non-zero without inventing a second opaque error
path.

The default NewAPI request path must remain transparent: if Goal Gate is not configured, normal API
traffic must not trigger runner calls, prompt injection, completion interception, or ledger writes.

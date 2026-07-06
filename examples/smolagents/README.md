# smolagents steward runner

This example uses Hugging Face `smolagents` as the external AI agent. Gateway Harness does not embed
the agent runtime; it invokes the runner explicitly and validates the returned steward proposal.

Install:

```sh
python -m pip install "smolagents[litellm]"
```

Run:

```sh
export GATEWAY_HARNESS_STEWARD_MODEL="openai/kimi-for-coding"
export OPENAI_API_BASE="http://192.168.31.65:3000/v1"
export OPENAI_API_KEY="..."

gateway-harness run-steward \
  fixtures/newapi/compact-context.steward.json \
  fixtures/newapi/compact-context.steward-event.json \
  -- python examples/smolagents/compact_steward.py \
  > /tmp/compact-context.steward-proposal.json

gateway-harness dry-run-steward-proposal \
  fixtures/newapi/compact-context.steward.json \
  /tmp/compact-context.steward-proposal.json \
  fixtures/newapi/compact-context.request.json

gateway-harness run-steward \
  fixtures/goal-gate/goal.before_complete.steward.json \
  fixtures/goal-gate/goal.before_complete.steward-event.json \
  -- python examples/smolagents/goal_reviewer.py \
  > /tmp/goal.before_complete.steward-proposal.json

gateway-harness dry-run-steward-proposal \
  fixtures/goal-gate/goal.before_complete.steward.json \
  /tmp/goal.before_complete.steward-proposal.json \
  fixtures/goal-gate/goal.before_complete.request.json
```

The event must be redacted (`"redacted": true`) and pass `validate-steward-event`; its `inputs` may
only use input names declared by the steward spec and cannot include raw-content keys such as
`prompt`, `messages`, `content`, `input`, or `output`. The runner must print only a steward proposal
JSON object to stdout. Harness rejects unsupported actions such as `context.truncate` and
`policy.patch.propose`.

`goal_reviewer.py` is the reference runner for `goal.before_complete`. It keeps Gateway Harness on
the event/contract side: the runner may approve completion, reject it, or request explicit
continuation, but Harness still validates the hook, the allowed inputs, and every emitted action
before an adapter or sidecar can act on the proposal.

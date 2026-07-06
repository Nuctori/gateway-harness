# Goal Gate Host Reference

This example shows the thinnest host-side completion interceptor for Gateway Harness.

It performs one explicit Goal Gate cycle:

1. load Goal Gate host config
2. load steward spec, event, and audit input
3. call the configured external runner through `adapter.ExecuteGoalGate`
4. consume `allow_complete`, `continue_work`, `continue_instruction`, and `next_goal_state`
5. append the returned redacted record into a ledger file

Run from the repository root:

```bash
GATEWAY_HARNESS_TEST_GOAL_PROPOSAL='{"version":"0.1","id":"proposal_goal_review_approve_host","steward":"goal-completion-reviewer","hook":"goal.before_complete","outputs":[{"action":"goal.approve_complete","reason":"reference host example approved"}]}' \
go run ./examples/goal-gate-host
```

The example writes a ledger file to `outputs/goal-gate-host.ledger.json` by default.

The bundled `goal-gate.demo.config.json` is intentionally enabled so the example demonstrates the
full completion-interception path. By contrast, `examples/newapi/goal-gate.config.json` stays
default-off to model production-safe host configuration.

## Minimal UI demo

This directory also contains the chat-first UI surface for the host bundle contract.

1. Refresh the bundled JSON payload:

```powershell
powershell -ExecutionPolicy Bypass -File ./examples/goal-gate-host/refresh-host-bundle.ps1
```

2. Start the local HTTP host from the repository root:

```bash
go run ./examples/goal-gate-host-http
```

3. Open [ui-demo.html](http://127.0.0.1:4070/ui-demo.html).

The demo reads the live bundle when the HTTP host is available, then falls back to the bundled JSON.
Its main interaction is intentionally a chat box, not a raw JSON editor:

- chat-first configuration assistant that turns natural-language requests into explicit config drafts
- manual settings drawer rendered from `form_model`
- live generated config JSON preview
- explicit apply step for assistant-generated drafts
- live execute preview when the HTTP host is present

The default enabled draft uses the minimal review scope: `goal_state`, `work_summary`, `test_results`,
`blockers`, and `recent_trace`. Broader redacted inputs such as `changed_files`,
`verification_summary`, and `user_goal` should only be added by explicit user choice.

The assistant path is intentionally proposal-only: it never auto-enables Goal Gate, auto-applies a
draft, mutates requests, or applies continuation patches behind the user's back. Its job is to show
that the current CLI contracts are enough to drive both a manual settings page and a chat-oriented
host UX.

To exercise a rejection / continue path, point `-event` at a different event fixture and provide a
proposal that returns `goal.reject_complete` plus `goal.request_continue`.

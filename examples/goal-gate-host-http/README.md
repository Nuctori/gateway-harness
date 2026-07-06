# Goal Gate Host HTTP Demo

This package exposes the thinnest live HTTP surface for the Goal Gate host demo.

Run it from the repository root:

```bash
go run ./examples/goal-gate-host-http
```

Then open [ui-demo.html](http://127.0.0.1:4070/ui-demo.html).

## What it proves

- Goal Gate stays default-off unless an explicit config is provided.
- A host can expose both manual settings and a chat-first configuration assistant without hidden behavior.
- The assistant only returns explicit config drafts; it does not auto-enable Goal Gate, auto-save config, or auto-apply continuation patches.
- The host can call the real `execute-goal-gate` path and render structured approve / reject / failure results.
- Default review scope stays minimal: broader redacted inputs must be explicitly authorized.

## Endpoints

- `GET /api/healthz`
  - Basic liveness probe.
- `GET /api/goal-gate/bundle`
  - Returns the one-shot host bundle: config schema, form schema, form model, result schema, interaction model, and examples.
- `GET /api/goal-gate/example-request`
  - Returns the demo config, steward event, and audit payload used by the UI preview.
- `POST /api/goal-gate/config-assistant`
  - Accepts a natural-language request plus the current explicit config.
  - Returns a proposal-only config draft with field-level changes and `requires_confirmation: true`.
- `POST /api/goal-gate/execute`
  - Accepts explicit config + steward event + audit input.
  - Runs the configured external runner through `adapter.ExecuteGoalGate`.
  - Returns a structured `GoalGateResult` on success or a structured failure result on auditable errors.

## Transparency contract

This demo intentionally keeps the AI-in-loop surface thin:

- no embedded model runtime
- no hidden prompt injection
- no hidden goal interception
- no hidden ledger writes
- no implicit draft application

Any real behavior still has to come from explicit config, explicit allowed inputs/actions, and a validated structured proposal.

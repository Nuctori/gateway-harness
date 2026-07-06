# Gateway Harness

[中文 README](README.zh-CN.md)

Gateway Harness is a small, host-agnostic policy layer for programmable LLM gateway context.

It is intended to be released and versioned as its own project. Gateway integrations such as NewAPI
should consume the policy/schema/CLI contract as adapters, not own the core concept.

It defines:

- **Hook**: when a policy runs, for example `request.before_upstream`.
- **Action**: what context transformation runs, for example `context.inject`.
- **Rule**: the normalized operator-facing contract: `Trigger + Scope + Operation + Audit`.
- **Condition**: whether a step applies to a model, token estimate, tag, or request shape.
- **Trace**: redacted audit metadata for debugging without leaking prompt content.
- **Adapter**: host-specific glue for a gateway such as NewAPI.
- **Adapter Capability**: an explicit manifest for supported hooks, actions, request shapes, and guards.
- **Ledger**: project/session/event audit records that reference hashes and external artifacts instead of raw prompt content.
- **Steward**: an explicit AI-in-the-loop sidecar contract for proposing validated context patches and audit artifacts.
- **Ledger Sentinel**: an explicit `context.inject_ledger_summary` action for reinjecting a
  project/session memory summary after compaction without hidden truncation, hidden budgets, or
  implicit AI calls.

The default normalized rule layer exposes two operations today:

- `inject_capsule`: inject one explicit context capsule.
- `ask_steward`: compile an explicit AI-in-the-loop steward spec. Compilation does not call AI or
  mutate requests.

Ledger provenance is audit metadata for injection, not a separate user-facing mental model. The
normalized AI steward path is intentionally non-destructive: no `context.truncate`, no
`policy.patch.propose`, and no fake gateway-side human-approval workflow.

Runtime AI-in-the-loop stays explicit and thin. Gateway Harness does not embed an agent framework or
provider; `run-steward` invokes an external open-source agent runner, passes a validated redacted
event JSON to stdin, and validates the proposal JSON returned on stdout. The event must use only
inputs declared by the steward spec and must not contain reserved raw-content keys such as
`prompt`, `messages`, `content`, `input`, or `output`. See `examples/smolagents/` for the recommended
lightweight runner pattern.

NewAPI is treated as an adapter example, not as the owner of the Gateway Harness concept.

## CLI

Validate a policy:

```bash
gateway-harness validate examples/newapi/context-harness.policy.json
```

Explain a policy:

```bash
gateway-harness explain examples/newapi/context-harness.policy.json
```

Print the JSON Schema:

```bash
gateway-harness schema
```

Validate a normalized rule:

```bash
gateway-harness validate-rule fixtures/newapi/context-rule.continuity-drop.json
```

Compile a normalized rule to the lower-level policy contract:

```bash
gateway-harness compile-rule fixtures/newapi/context-rule.continuity-drop.json
```

Compile a normalized AI-in-the-loop rule to steward specs:

```bash
gateway-harness compile-rule-stewards fixtures/newapi/context-rule.ask-steward.json
```

Invoke an external steward runner:

```bash
gateway-harness run-steward \
  fixtures/newapi/compact-context.steward.json \
  fixtures/newapi/compact-context.steward-event.json \
  -- python examples/smolagents/compact_steward.py
```

Validate a `goal.before_complete` steward event:

```bash
gateway-harness validate-steward-event fixtures/goal-gate/goal.before_complete.steward.json fixtures/goal-gate/goal.before_complete.steward-event.json
```

Validate goal-completion proposals:

```bash
gateway-harness validate-steward-proposal fixtures/goal-gate/goal.before_complete.steward.json fixtures/goal-gate/goal.before_complete.steward-proposal.approve.json

gateway-harness validate-steward-proposal fixtures/goal-gate/goal.before_complete.steward.json fixtures/goal-gate/goal.before_complete.steward-proposal.reject.json
```

Dry-run a rejected goal-completion proposal:

```bash
gateway-harness dry-run-steward-proposal fixtures/goal-gate/goal.before_complete.steward.json fixtures/goal-gate/goal.before_complete.steward-proposal.reject.json fixtures/goal-gate/goal.before_complete.request.json
```

Evaluate a goal-completion proposal into a terminal decision:

```bash
gateway-harness evaluate-goal-proposal fixtures/goal-gate/goal.before_complete.steward.json fixtures/goal-gate/goal.before_complete.steward-event.json fixtures/goal-gate/goal.before_complete.steward-proposal.reject.json
```

Apply that decision into the next explicit goal state for an adapter or sidecar:

```bash
gateway-harness apply-goal-review-result fixtures/goal-gate/goal.before_complete.steward.json fixtures/goal-gate/goal.before_complete.steward-event.json fixtures/goal-gate/goal.before_complete.steward-proposal.reject.json
```

Execute the host-side Goal Gate flow end to end using explicit config, spec, event, and audit input:

```bash
gateway-harness execute-goal-gate examples/newapi/goal-gate.config.json fixtures/goal-gate/goal.before_complete.steward.json fixtures/goal-gate/goal.before_complete.steward-event.json fixtures/goal-gate/goal.before_complete.audit.json
```

Print the rule schema:

```bash
gateway-harness rule-schema
```

Dry-run a policy against a request copy:

```bash
gateway-harness dry-run-policy examples/newapi/context-harness.policy.json responses.compact.before_upstream fixtures/newapi/policy-dry-run.request.json
```

Validate an adapter capability manifest:

```bash
gateway-harness validate-adapter examples/newapi/adapter.capability.json
```

Print the Goal Gate host config schema:

```bash
gateway-harness goal-gate-config-schema
```

Print the Goal Gate host form schema and the default UI-facing form model:

```bash
gateway-harness goal-gate-result-schema
gateway-harness goal-gate-form-schema
gateway-harness goal-gate-form-model
gateway-harness goal-gate-host-bundle
```

The schema is intended to be rendered directly by a WebUI or host console. It includes titles,
descriptions, examples, and GUI-friendly enumerations for `hook`, `allowed_inputs`, and
`allowed_actions`, so hosts do not need to hardcode their own hidden field catalog.

The form model is the next thin layer up: it keeps the same explicit contract, but adds section
grouping, suggested control types, field order, and a default transparency note so a host can build
an obvious configuration page without inventing its own hidden layout rules. The same bundle can
also power a chat-first assistant surface, as long as the assistant only proposes explicit config
drafts and the host still requires an explicit apply/save step.

The host bundle now also carries a localized `hook_catalog` with `zh-CN` and `en-US` labels,
summaries, and examples. That lets a WebUI render hover help or a hook gallery without inventing a
second hidden field dictionary. The current Goal Gate host still exposes only
`goal.before_complete` as a runnable interception point; `goal.before_resume` and
`goal.after_complete` are available in the lower-level contract and catalog, but are not yet
surfaced as runnable Goal Gate toggles in this host UI.

`goal-gate-result-schema` is the matching output-side contract for hosts and WebUIs. It documents
approve, reject/continue, structured failure, ledger append, and optional `continuation_patches`
without requiring consumers to reverse-engineer the Go structs.

`goal-gate-host-bundle` is the one-shot integration bundle: it returns the config schema, form
schema, form model, result schema, and sample approve/reject/failure results in one JSON payload so
hosts do not need to orchestrate multiple CLI calls before rendering a page or wiring a client.

For a concrete host-side rendering example, see [examples/goal-gate-host/README.md](/C:/Users/Nuctori/Documents/Codex/2026-07-03/c-new-api-build/work/gateway-harness/examples/goal-gate-host/README.md), [ui-demo.html](/C:/Users/Nuctori/Documents/Codex/2026-07-03/c-new-api-build/work/gateway-harness/examples/goal-gate-host/ui-demo.html), and the live demo host in [examples/goal-gate-host-http/main.go](/C:/Users/Nuctori/Documents/Codex/2026-07-03/c-new-api-build/work/gateway-harness/examples/goal-gate-host-http/main.go).

For an acceptance-oriented checklist of the current Goal Gate behavior, see [docs/goal-gate-acceptance.md](/C:/Users/Nuctori/Documents/Codex/2026-07-03/c-new-api-build/work/gateway-harness/docs/goal-gate-acceptance.md).

Validate a Goal Gate host config:

```bash
gateway-harness validate-goal-gate-config examples/newapi/goal-gate.config.json
```

Relative `runner.workdir` values are resolved against the config file location before execution.
When `execute-goal-gate` fails, it still prints a structured `GoalGateResult` with `failure` and,
when audit input is sufficient, an `append_record` for an explicit ledger `error` event before
exiting non-zero.

Validate a conformance fixture:

```bash
gateway-harness validate-conformance fixtures/newapi/responses-tool-chain.conformance.json
```

Replay a conformance fixture against a local fake upstream:

```bash
gateway-harness replay-conformance fixtures/newapi/responses-tool-chain.conformance.json
```

Apply a fixture policy, replay the applied request against a local fake upstream, and print only a
redacted trace:

```bash
gateway-harness replay-policy-conformance fixtures/newapi/responses-policy-apply.conformance.json responses.before_upstream
```

Print the conformance fixture schema:

```bash
gateway-harness conformance-schema
```

Validate a project/session audit ledger:

```bash
gateway-harness validate-ledger fixtures/newapi/project-session.ledger.json
```

Explain a ledger:

```bash
gateway-harness explain-ledger fixtures/newapi/project-session.ledger.json
```

Query persisted project/session ledger entries by project, session, tag, or event type:

```bash
gateway-harness query-ledger fixtures/newapi/project-session.ledger.json -tag adapter:newapi -tag domain:coding -event-type compact
```

Append an explicit redacted event record into a project/session ledger, creating the ledger,
project, or session if needed:

```bash
gateway-harness append-ledger-record runtime/project-session.ledger.json fixtures/newapi/continuity-drop.append-record.json
```

Print the ledger schema:

```bash
gateway-harness ledger-schema
```

Print the ledger append-record schema:

```bash
gateway-harness ledger-record-schema
```

Validate an AI steward spec:

```bash
gateway-harness validate-steward fixtures/newapi/compact-context.steward.json
```

Explain a steward spec:

```bash
gateway-harness explain-steward fixtures/newapi/compact-context.steward.json
```

Print the steward schema:

```bash
gateway-harness steward-schema
```

Validate an AI steward proposal against its spec:

```bash
gateway-harness validate-steward-proposal fixtures/newapi/compact-context.steward.json fixtures/newapi/compact-context.steward-proposal.json
```

Explain a steward proposal:

```bash
gateway-harness explain-steward-proposal fixtures/newapi/compact-context.steward.json fixtures/newapi/compact-context.steward-proposal.json
```

Print the steward proposal schema:

```bash
gateway-harness steward-proposal-schema
```

Dry-run a steward proposal against a request copy:

```bash
gateway-harness dry-run-steward-proposal fixtures/newapi/compact-context.steward.json fixtures/newapi/compact-context.steward-proposal.json fixtures/newapi/compact-context.request.json
```

Goal Gate uses the same steward contract surface. In dry-run, goal actions such as
`goal.approve_complete`, `goal.reject_complete`, and `goal.request_continue` are surfaced as
structured `goal_actions`; they do not mutate the request copy. Real completion interception,
continue limits, cooldown, and dedupe still belong to an explicit adapter or sidecar integration.

For adapters and sidecars, `evaluate-goal-proposal` and `apply-goal-review-result` are the thin
execution layer on top of the steward contract. The first command decides whether the goal may
complete; the second command computes the next explicit `goal_state`, a continuation/no-continuation
decision, and redacted audit metadata that can be persisted through the ledger.

In-process integrations can also call the steward helpers directly: `ReviewGoalCompletion`,
`EvaluateGoalProposal`, `ApplyGoalReviewResult`, and `BuildGoalGateAppendRecord`. That lets a
sidecar keep Goal Gate explicit end to end: invoke a configured runner, validate its structured
proposal, compute the next persisted goal state, then append a redacted `harness_action` or `error`
event into the session ledger without inventing hidden workflow state.

Conformance fixtures validate Gateway Harness contracts, adapter capabilities, and realistic request
shapes. `replay-conformance` posts the fixture request to a local fake upstream to exercise the HTTP
path without network access or model calls. It does not replace live upstream tests.

`replay-policy-conformance` goes one step further: it applies the fixture policy to a request copy,
posts the applied request to the fake upstream, and prints only request sizes plus a redacted trace.
It is the CI-safe path for proving an adapter-style mutation preserves protocol shape without
leaking raw prompts or raw injected text.

Ledger files validate the audit boundary for project/session history. They intentionally store event
metadata, content hashes, and artifact references, not raw prompts or raw model outputs. Metadata is
for labels and IDs; obvious raw-content keys such as `prompt`, `response`, and `messages` are rejected.
`query-ledger` makes persisted sessions searchable by project, session, tag, or event type while
returning only session metadata, event counts, matched event IDs, and artifact IDs.
`append-ledger-record` is the minimal persistence primitive for adapters and sidecars: it atomically
adds one explicit event plus optional hashed artifacts to a ledger file. It can create a project or
session boundary, but it still rejects raw prompt/response metadata and validates the full ledger
after the append.

Steward specs validate AI-in-the-loop context management. A steward can be configured for compact,
failover, or diagnostic hooks, but it must use explicit hooks, redacted inputs, structured outputs,
validated actions, and artifact hashes. Gateway Harness core does not call a steward unless an adapter
implements that explicit sidecar behavior.

Steward proposals validate what an AI steward actually returned. A proposal is checked against the
spec that enabled it, so an AI cannot use a disabled hook, emit an unlisted action, write an artifact
without a hash, or silently apply a policy patch.

`dry-run-steward-proposal` prints a redacted patch plan for non-destructive proposal outputs. It does
not print the full rewritten request, call an AI, write a database, contact an upstream, or perform
destructive `context.truncate` edits.

`dry-run-policy` applies the same transparency rule to ordinary policies. It reports matched
programs, applied actions, skipped destructive actions, and redacted request patch metadata such as
target, insert index, role, reason, content hash, and content length. It does not print the rewritten
request or raw injected text. Token-gated conditions only match when the caller provides an explicit
`estimated_tokens` argument.

For compaction-aware adapters, `context.inject_ledger_summary` is the recommended sentinel action.
The adapter or operator supplies the summary text explicitly and names the `ledger_ref` plus any
`artifact_refs`; Gateway Harness only applies the explicit patch and records redacted provenance.

Adapters that can transparently detect a sharp context drop may expose the explicit
`context.continuity_drop.detected` hook plus `when.context_continuity_drop=true`. Gateway Harness
validates and dry-runs that contract, but does not store raw prompts, call an AI, or recover context
the client did not send.

## Project Layout

```text
cmd/gateway-harness/      CLI entrypoint
adapter/                  Adapter capability manifest structs and validation
conformance/              Protocol fixture validation
ledger/                   Project/session audit ledger validation
steward/                  AI-in-the-loop steward contract validation
policy/                   Policy structs, validation, summaries
schema/                   JSON Schema for editors and WebUI
docs/                     Concepts and adapter contracts
examples/newapi/          NewAPI adapter example policy
fixtures/newapi/          NewAPI conformance fixtures
```

## Release Shape

The main project should publish:

- `gateway-harness` CLI binaries.
- `gateway-harness.policy.schema.json`.
- `gateway-harness.adapter.schema.json`.
- `gateway-harness.conformance.schema.json`.
- `gateway-harness.ledger.schema.json`.
- `gateway-harness.ledger-record.schema.json`.
- `gateway-harness.steward.schema.json`.
- `gateway-harness.steward-proposal.schema.json`.
- Checksums.
- `gateway-harness-examples.tar.gz` with examples, fixtures, docs, and README files.
- `gateway-harness-demo-stack-<platform>.tar.gz` with the runnable NewAPI + Goal Gate demo bundle for
  Linux amd64 and Linux arm64.

Gateway-specific builds, patches, and images belong in adapter repositories such as
`newapi-gateway-harness-example`.

See [RELEASE.md](RELEASE.md) for the v0.2 release boundary.

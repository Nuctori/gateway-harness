# Gateway Harness

[中文 README](README.zh-CN.md)

Gateway Harness is a small, host-agnostic policy layer for programmable LLM gateway context.

It is intended to be released and versioned as its own project. Gateway integrations such as NewAPI
should consume the policy/schema/CLI contract as adapters, not own the core concept.

It defines:

- **Hook**: when a policy runs, for example `request.before_upstream`.
- **Action**: what context transformation runs, for example `context.inject`.
- **Condition**: whether a step applies to a model, token estimate, tag, or request shape.
- **Trace**: redacted audit metadata for debugging without leaking prompt content.
- **Adapter**: host-specific glue for a gateway such as NewAPI.
- **Adapter Capability**: an explicit manifest for supported hooks, actions, request shapes, and guards.
- **Ledger**: project/session/event audit records that reference hashes and external artifacts instead of raw prompt content.
- **Steward**: an explicit AI-in-the-loop sidecar contract for proposing validated context patches and audit artifacts.
- **Ledger Sentinel**: an explicit `context.inject_ledger_summary` action for reinjecting a
  project/session memory summary after compaction without hidden truncation, hidden budgets, or
  implicit AI calls.

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

Dry-run a policy against a request copy:

```bash
gateway-harness dry-run-policy examples/newapi/context-harness.policy.json responses.compact.before_upstream fixtures/newapi/policy-dry-run.request.json
```

Validate an adapter capability manifest:

```bash
gateway-harness validate-adapter examples/newapi/adapter.capability.json
```

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

Gateway-specific builds, patches, and images belong in adapter repositories such as
`newapi-gateway-harness-example`.

See [RELEASE.md](RELEASE.md) for the v0.2 release boundary.

# Release Plan

Gateway Harness releases are independent from NewAPI releases.

## v0.2.0 Scope

v0.2.0 promotes Gateway Harness from a static policy/schema package into a CI-verifiable harness
contract for realistic adapter flows.

- Everything in v0.1.x.
- `context.inject_ledger_summary` for explicit compaction/project-memory sentinels.
- Policy dry-run with redacted patch plans for ordinary requests and compact hooks.
- Conformance replay against a local fake upstream.
- Policy-apply conformance replay, proving that a policy mutation preserves the request shape before
  it reaches an adapter upstream.
- Project/session ledger query by project, session, tag, and event type via `query-ledger`.
- Project/session ledger append primitive via `append-ledger-record`, with atomic file replacement,
  project/session creation, full validation, and raw prompt/response metadata rejection.
- AI steward proposal validation and dry-run against request copies.
- CI coverage for NewAPI example policy, adapter capability, conformance replay, ledger query,
  steward specs, and steward proposals.

## v0.2.1 Scope

- Adds the NewAPI online acceptance script packaged with examples.
- The script validates a deployed NewAPI host without taking over port 80: live policy export,
  Gateway Harness policy validation, ledger-summary hook coverage, hidden budget/truncate rejection,
  failover option checks, HTTP health, and Docker port checks.

## v0.2.2 Scope

- Extends the NewAPI online acceptance script with optional live smoke tests.
- When `NEWAPI_API_KEY` or `NEWAPI_API_KEY_FILE` is explicitly provided, the script sends a minimal
  `/v1/responses` request and verifies redacted `context_harness` trace metadata in Docker logs.
- `COMPACT_SMOKE=1` explicitly enables `/v1/responses/compact` smoke and matching compact trace
  validation. Compact smoke is opt-in because it may consume upstream quota.
- The script does not discover tokens from the NewAPI database and does not print token values.

## v0.2.3 Scope

- Keeps live-smoke failure response bodies suppressed by default.
- Adds `PRINT_ERROR_BODY=1` as an explicit debugging opt-in for upstream error details.

## v0.2.4 Scope

- Adds a mock CI acceptance test for `examples/newapi/online-acceptance.sh`.
- The mock test covers the no-token path, opt-in live `/v1/responses` smoke, opt-in compact smoke,
  Docker port checks, redacted trace checks, failover option validation, and default failure-body
  suppression without requiring a live NewAPI host or real API token.

## v0.2.5 Scope

- Fixes the mock CI acceptance runner to invoke `online-acceptance.sh` through `sh`, matching the
  documented usage and avoiding executable-bit assumptions on fresh GitHub Actions checkouts.

## v0.2.6 Scope

- Adds the explicit `context.continuity_drop.detected` adapter event hook and
  `when.context_continuity_drop=true` condition.
- Keeps continuity-drop handling transparent: adapters provide the event signal, policies decide
  whether to inject, and Gateway Harness does not store raw prompts, call an AI, truncate context,
  or recover transcript content the client did not send.
- Updates policy, adapter, and steward schemas so WebUIs and CLIs share the same contract.
- Extends the NewAPI example policy, adapter capability manifest, CI dry-run path, and docs for
  the continuity-drop ledger sentinel.

## v0.2.7 Scope

- Adds `append-ledger-record` for appending explicit, redacted project/session ledger records.
- Adds `ledger-record-schema`.
- Ensures nested ledger parent directories are created during atomic ledger writes.

## v0.2.8 Scope

- Adds the normalized Rule contract: `Rule = Trigger + Scope + Operation + Audit`.
- Keeps the default rule operation intentionally narrow: `inject_capsule` only.
- Compiles `inject_capsule` without audit to `context.inject`, and with `audit.ledger_ref` to
  `context.inject_ledger_summary` as a lower-level provenance encoding.
- Keeps destructive `context.truncate`, hidden budgets, and implicit AI calls out of the normalized
  rule layer.
- Adds `validate-rule`, `explain-rule`, `compile-rule`, and `rule-schema` CLI commands.
- Adds the NewAPI continuity-drop normalized rule fixture and CI coverage that validates the
  compiled policy does not contain `context.truncate`, `budget`, or `ask_steward`.

## v0.2.9 Scope

- Adds normalized `ask_steward` rules that compile to explicit steward specs.
- Keeps steward compilation non-executing: Gateway Harness does not call AI, mutate requests, or
  read raw prompts.
- Rejects `context.truncate`, `policy.patch.propose`, and `human_approval_for_policy_patch` in the
  normalized steward path. The gateway does not pretend to own a human approval workflow.
- Adds `compile-rule-stewards` and a NewAPI compact steward rule fixture.

## v0.1.x Scope

- Policy structs and validation.
- Adapter capability structs and validation.
- Conformance fixture validation.
- Project/session ledger validation.
- AI-in-the-loop steward spec validation.
- AI-in-the-loop steward proposal validation.
- JSON Schema for editor and WebUI integration.
- CLI commands: `validate`, `explain`, `schema`, `validate-adapter`, `explain-adapter`, `adapter-schema`,
  `validate-conformance`, `explain-conformance`, `replay-conformance`, `conformance-schema`,
  `validate-ledger`, `explain-ledger`, `query-ledger`, `append-ledger-record`, `ledger-schema`,
  `ledger-record-schema`, `validate-steward`, `explain-steward`,
  `steward-schema`, `validate-steward-proposal`, `explain-steward-proposal`, and
  `steward-proposal-schema`, `dry-run-policy`, `replay-policy-conformance`, and
  `dry-run-steward-proposal`.
- NewAPI example policy and adapter contract documentation.
- Release archive with examples, fixtures, docs, README files, and license.
- Cross-compiled CLI artifacts for Linux amd64, Linux arm64, Linux armv7, and Windows amd64.

## Out Of Scope For v0.2.0

- Shipping a patched NewAPI binary or Docker image from this repository.
- Storing or replaying model conversation state.
- Storing raw prompt or response content in the ledger contract.
- Hidden gateway context-window enforcement.
- Executing arbitrary scripts from policies.
- Calling AI stewards implicitly without an explicit steward spec and adapter implementation.
- Applying AI steward proposals without validating them against their steward spec.
- Performing destructive proposal edits during dry-run.
- Performing destructive policy edits during dry-run.

## Tagging

Use semantic version tags:

```bash
git tag v0.2.0
git push origin v0.2.0
```

The release workflow uploads CLI binaries, schemas, and checksums.

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
- AI steward proposal validation and dry-run against request copies.
- CI coverage for NewAPI example policy, adapter capability, conformance replay, ledger query,
  steward specs, and steward proposals.

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
  `validate-ledger`, `explain-ledger`, `query-ledger`, `ledger-schema`, `validate-steward`, `explain-steward`,
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

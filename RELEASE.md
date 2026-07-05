# Release Plan

Gateway Harness releases are independent from NewAPI releases.

## v0.1.x Scope

- Policy structs and validation.
- Adapter capability structs and validation.
- Conformance fixture validation.
- Project/session ledger validation.
- AI-in-the-loop steward spec validation.
- JSON Schema for editor and WebUI integration.
- CLI commands: `validate`, `explain`, `schema`, `validate-adapter`, `explain-adapter`, `adapter-schema`,
  `validate-conformance`, `explain-conformance`, `replay-conformance`, `conformance-schema`,
  `validate-ledger`, `explain-ledger`, `ledger-schema`, `validate-steward`, `explain-steward`,
  and `steward-schema`.
- NewAPI example policy and adapter contract documentation.
- Cross-compiled CLI artifacts for Linux amd64, Linux arm64, Linux armv7, and Windows amd64.

## Out Of Scope For v0.1.x

- Shipping a patched NewAPI binary or Docker image from this repository.
- Storing or replaying model conversation state.
- Storing raw prompt or response content in the ledger contract.
- Hidden gateway context-window enforcement.
- Executing arbitrary scripts from policies.
- Calling AI stewards implicitly without an explicit steward spec and adapter implementation.

## Tagging

Use semantic version tags:

```bash
git tag v0.1.0
git push origin v0.1.0
```

The release workflow uploads CLI binaries, schemas, and checksums.

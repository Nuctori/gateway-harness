# Release Plan

Gateway Harness releases are independent from NewAPI releases.

## v0.1.x Scope

- Policy structs and validation.
- Adapter capability structs and validation.
- Conformance fixture validation.
- JSON Schema for editor and WebUI integration.
- CLI commands: `validate`, `explain`, `schema`, `validate-adapter`, `explain-adapter`, `adapter-schema`,
  `validate-conformance`, `explain-conformance`, and `conformance-schema`.
- NewAPI example policy and adapter contract documentation.
- Cross-compiled CLI artifacts for Linux amd64, Linux arm64, Linux armv7, and Windows amd64.

## Out Of Scope For v0.1.x

- Shipping a patched NewAPI binary or Docker image from this repository.
- Storing or replaying model conversation state.
- Hidden gateway context-window enforcement.
- Executing arbitrary scripts from policies.

## Tagging

Use semantic version tags:

```bash
git tag v0.1.0
git push origin v0.1.0
```

The release workflow uploads CLI binaries, the policy schema, and checksums.

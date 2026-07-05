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

Validate an adapter capability manifest:

```bash
gateway-harness validate-adapter examples/newapi/adapter.capability.json
```

## Project Layout

```text
cmd/gateway-harness/      CLI entrypoint
adapter/                  Adapter capability manifest structs and validation
policy/                   Policy structs, validation, summaries
schema/                   JSON Schema for editors and WebUI
docs/                     Concepts and adapter contracts
examples/newapi/          NewAPI adapter example policy
```

## Release Shape

The main project should publish:

- `gateway-harness` CLI binaries.
- `gateway-harness.policy.schema.json`.
- `gateway-harness.adapter.schema.json`.
- Checksums.
- Example policies.

Gateway-specific builds, patches, and images belong in adapter repositories such as
`newapi-gateway-harness-example`.

See [RELEASE.md](RELEASE.md) for the v0.1 release boundary.

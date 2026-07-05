# NewAPI Adapter Example

This directory is an adapter example. It shows how NewAPI can host Gateway Harness policies, but
NewAPI does not own the Gateway Harness concept.

Use the CLI to validate the example policy:

```bash
gateway-harness validate examples/newapi/context-harness.policy.json
```

NewAPI adapter responsibilities:

- Map NewAPI relay phases to Gateway Harness hooks.
- Convert Chat / Responses requests into mutable context objects.
- Execute Gateway Harness actions against those objects.
- Write redacted trace metadata into NewAPI logs.
- Keep Gateway Harness budgets transparent and scoped to harness-owned mutations.


# NewAPI Adapter Example

This directory is an adapter example. It shows how NewAPI can host Gateway Harness policies, but
NewAPI does not own the Gateway Harness concept.

Use the CLI to validate the example policy:

```bash
gateway-harness validate examples/newapi/context-harness.policy.json
```

Validate the adapter capability manifest:

```bash
gateway-harness validate-adapter examples/newapi/adapter.capability.json
```

Run online acceptance on a deployed NewAPI host after installing the `gateway-harness` CLI:

```bash
sh examples/newapi/online-acceptance.sh
```

The script validates the live `context_harness.policy`, checks that ledger-summary injection is
explicit, rejects hidden `budget` / `context.truncate` policy fields, verifies model failover options,
checks NewAPI HTTP health, and ensures the NewAPI container is not publishing port 80.

NewAPI adapter responsibilities:

- Map NewAPI relay phases to Gateway Harness hooks.
- Publish an explicit adapter capability manifest.
- Convert Chat / Responses requests into mutable context objects.
- Execute Gateway Harness actions against those objects.
- Write redacted trace metadata into NewAPI logs.
- Keep adapter-local guards explicit and avoid hidden model context limits.

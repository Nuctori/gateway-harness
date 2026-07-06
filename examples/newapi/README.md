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

To additionally prove the live relay path emits redacted harness trace metadata, provide a NewAPI
token explicitly:

```bash
NEWAPI_API_KEY=sk-... sh examples/newapi/online-acceptance.sh
```

Compact smoke is opt-in because it may consume upstream quota:

```bash
NEWAPI_API_KEY=sk-... COMPACT_SMOKE=1 sh examples/newapi/online-acceptance.sh
```

The script never discovers tokens from the database and never prints the token. Use
`NEWAPI_API_KEY_FILE=/path/to/token` if you prefer not to put the token in shell history.
Failed live-smoke response bodies are suppressed by default; set `PRINT_ERROR_BODY=1` only when you
explicitly want upstream error details for debugging.

CI can exercise the acceptance script without a live NewAPI host by running:

```bash
sh examples/newapi/online-acceptance.test.sh
```

The mock test creates a temporary SQLite options database and fake `curl`, `docker`, and
`gateway-harness` commands. It covers the default no-token path, live `/v1/responses` smoke,
compact smoke, Docker port checks, redacted trace checks, failover option validation, and the
default suppression of failed upstream response bodies.

NewAPI adapter responsibilities:

- Map NewAPI relay phases to Gateway Harness hooks.
- Publish an explicit adapter capability manifest.
- Convert Chat / Responses requests into mutable context objects.
- Execute Gateway Harness actions against those objects.
- Write redacted trace metadata into NewAPI logs.
- Keep adapter-local guards explicit and avoid hidden model context limits.

# Gateway Harness v0.2

`v0.2` adds a bounded programmable context harness.

Acceptance phrase:

> I can program Gateway Harness to control the context of any model, on this 玩客云, without turning the gateway into a heavyweight agent platform.

Required capabilities:

- `models: ["*"]` wildcard selector.
- Normalized `request.pre_context` event.
- Declarative context program.
- Deterministic `ContextPatch`.
- Patch budget limits.
- Redacted trace metadata.
- ARMv7 build.

Non-goals:

- Visual programming UI.
- Arbitrary JavaScript/Python/Lua.
- Vector database.
- External AI summarization requirement.
- Full raw prompt persistence by default.

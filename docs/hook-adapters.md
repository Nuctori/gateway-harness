# Hook Adapters

Gateway Harness keeps the core normalized and pushes client-specific hook handling into ingress adapters.

## Rule

- Core policy and runtime only understand normalized events.
- Each CLI or agent surface gets a thin adapter that translates its hook names into those events.
- Conversation audit/injection stays in the adapter layer, not in policy logic.

## Current Mapping

| Upstream surface | Hook event | Normalized event | Current support |
| --- | --- | --- | --- |
| Codex | `UserPromptSubmit` | `request.pre_prompt` | supported |
| Codex | `PreCompact` | `request.pre_context` | supported |
| Codex | `PreToolUse` | tool-hook boundary | not yet mapped in core |
| OpenCode | `tui.prompt.append` | `request.pre_prompt` | supported |
| OpenCode | `experimental.session.compacting` | `request.pre_context` | supported |
| OpenCode | `tool.execute.before` | tool-hook boundary | not yet mapped in core |

## Why This Shape

Codex and OpenCode do not expose the same hook names or payload conventions. The cleanest boundary is:

`upstream hook -> adapter -> normalized event -> policy engine -> decision -> adapter`

This keeps the decision model stable while letting each client evolve independently.

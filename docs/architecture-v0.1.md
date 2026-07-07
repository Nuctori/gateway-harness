# Gateway Harness v0.1

`v0.1` proves a deterministic core loop:

```text
Event -> Match Policy -> Decision/Plan -> Adapter Execute -> Outcome -> Trace
```

Required capabilities:

- Policy validation.
- Prompt injection decision.
- Static `fallback.sequence` resolving to `retry.with_model`.
- Trace replay.
- Go SDK adapter contract.

Non-goals:

- WebUI.
- XML runtime.
- Arbitrary script execution.
- Memory/indexing/summarization.
- Provider-specific logic in core.

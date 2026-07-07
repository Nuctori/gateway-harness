# Gateway Harness

Gateway Harness is a small programmable and auditable AI gateway harness.

The current implementation targets the `v0.1` and `v0.2` design docs:

- `v0.1`: deterministic policy decisions, prompt injection, static model fallback, trace replay.
- `v0.2`: bounded context harness for any normalized model request via `ContextPatch`.

It is intentionally CLI/SDK first, provider-agnostic, and ARMv7-friendly.

## Quick Start

```bash
go test ./...
go run ./cmd/harness policy validate examples/policies/coding.yaml
go run ./cmd/harness run --event examples/fixtures/upstream-error-429.json --policy examples/policies/coding.yaml
go run ./cmd/harness context dry-run --event examples/fixtures/pre-context-any-model.json --policy examples/policies/context-harness.yaml
go run ./cmd/harness trace replay examples/fixtures/context-patch-trace.json
```

## ARMv7 Build

```bash
GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 go build -o dist/harness-linux-armv7 ./cmd/harness
```

PowerShell:

```powershell
$env:GOOS='linux'; $env:GOARCH='arm'; $env:GOARM='7'; $env:CGO_ENABLED='0'; go build -o dist/harness-linux-armv7 ./cmd/harness
```

## Design Rule

Core produces deterministic decisions and patches. Adapters apply them.

The core does not call model providers, does not read gateway databases, does not execute arbitrary scripts, and does not persist raw prompts by default.

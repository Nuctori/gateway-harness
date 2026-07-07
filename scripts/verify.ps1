$ErrorActionPreference = "Stop"

go test ./...
go run ./cmd/harness policy validate examples/policies/coding.yaml
go run ./cmd/harness run --event examples/fixtures/upstream-error-429.json --policy examples/policies/coding.yaml
go run ./cmd/harness context dry-run --event examples/fixtures/pre-context-any-model.json --policy examples/policies/context-harness.yaml
go run ./cmd/harness context dry-run --event examples/fixtures/pre-context-kimi.json --policy examples/policies/context-harness.yaml
go run ./cmd/harness trace replay examples/fixtures/context-patch-trace.json
./scripts/build-armv7.ps1

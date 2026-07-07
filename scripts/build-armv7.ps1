$ErrorActionPreference = "Stop"

New-Item -ItemType Directory -Force -Path "dist" | Out-Null

$env:GOOS = "linux"
$env:GOARCH = "arm"
$env:GOARM = "7"
$env:CGO_ENABLED = "0"

go build -o "dist/harness-linux-armv7" ./cmd/harness

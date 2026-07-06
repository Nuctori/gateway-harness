param(
  [string]$OutputPath = "goal-gate-host.bundle.example.json"
)

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$repoRoot = Resolve-Path (Join-Path $scriptDir "..\..")
$targetPath = Join-Path $scriptDir $OutputPath

Push-Location $repoRoot
try {
  $json = go run ./cmd/gateway-harness goal-gate-host-bundle
  Set-Content -LiteralPath $targetPath -Value $json -Encoding UTF8
  Write-Output "wrote $targetPath"
}
finally {
  Pop-Location
}

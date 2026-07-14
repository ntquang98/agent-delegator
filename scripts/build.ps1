[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'
$repoRoot = Split-Path -Parent $PSScriptRoot
$bridgeRoot = Join-Path $repoRoot 'bridge'
$output = Join-Path $repoRoot 'plugins\delegate-hub\bin\delegate-hub.exe'

New-Item -ItemType Directory -Force -Path (Split-Path -Parent $output) | Out-Null
Push-Location $bridgeRoot
try {
    go build -trimpath -ldflags '-s -w' -o $output .\cmd\delegate-hub
} finally {
    Pop-Location
}

Write-Host "Built $output"

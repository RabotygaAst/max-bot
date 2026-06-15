param(
    [string]$MockAddr = ":1080",
    [string]$MockConfig = "mock-onec-config.json"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
Set-Location $repoRoot

if (-not (Test-Path ".\cmd\bot\devmock\main.go")) {
    throw "Missing .\cmd\bot\devmock\main.go. Update the repository and retry."
}

if (-not (Test-Path ".\.env.local")) {
    Copy-Item ".\.env.local.example" ".\.env.local"
    Write-Host "Created .env.local from .env.local.example"
}

Get-Content ".\.env.local" | ForEach-Object {
    $line = $_.Trim()
    if ($line -eq "" -or $line.StartsWith("#")) {
        return
    }

    $parts = $line -split "=", 2
    if ($parts.Count -ne 2) {
        return
    }

    [Environment]::SetEnvironmentVariable($parts[0].Trim(), $parts[1].Trim(), "Process")
}

Write-Host "Starting local 1C/MAX mock on $MockAddr..."
$mockProcess = Start-Process -FilePath "go" -ArgumentList @("run", "./cmd/bot/devmock", "-addr", $MockAddr, "-config", $MockConfig) -PassThru -NoNewWindow

try {
    Start-Sleep -Seconds 2
    Write-Host "Starting webhook/debug bot on $env:HTTP_ADDR with in-memory storage..."
    go run ./cmd/bot
}
finally {
    if ($mockProcess -and -not $mockProcess.HasExited) {
        Stop-Process -Id $mockProcess.Id -Force
    }
}

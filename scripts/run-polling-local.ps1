param(
    [Parameter(Mandatory = $true)]
    [string]$MaxToken,
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
if (-not (Test-Path ".\cmd\bot-polling\main.go")) {
    throw "Missing .\cmd\bot-polling\main.go. Update the repository and retry."
}

[Environment]::SetEnvironmentVariable("MAX_BASE_URL", "https://platform-api.max.ru", "Process")
[Environment]::SetEnvironmentVariable("MAX_TOKEN", $MaxToken, "Process")
[Environment]::SetEnvironmentVariable("ONEC_BASE_URL", "http://localhost:1080", "Process")
[Environment]::SetEnvironmentVariable("ONEC_TOKEN", "MOCK_ONEC_TOKEN", "Process")
[Environment]::SetEnvironmentVariable("INTERNAL_API_TOKEN", "CHANGE_ME_INTERNAL_TOKEN", "Process")
[Environment]::SetEnvironmentVariable("REQUEST_TIMEOUT_SECONDS", "10", "Process")
[Environment]::SetEnvironmentVariable("POLLING_LIMIT", "100", "Process")
[Environment]::SetEnvironmentVariable("POLLING_TIMEOUT_SECONDS", "30", "Process")
[Environment]::SetEnvironmentVariable("POLLING_RETRY_SECONDS", "5", "Process")
[Environment]::SetEnvironmentVariable("POLLING_TYPES", "message_created,message_callback,bot_started", "Process")
[Environment]::SetEnvironmentVariable("DATABASE_URL", $null, "Process")

Write-Host "Starting local 1C mock on $MockAddr..."
$mockProcess = Start-Process -FilePath "go" -ArgumentList @("run", "./cmd/bot/devmock", "-addr", $MockAddr, "-config", $MockConfig) -PassThru -NoNewWindow

try {
    Start-Sleep -Seconds 2
    Write-Host "Starting MAX polling bot. Press Ctrl+C to stop."
    go run ./cmd/bot-polling
}
finally {
    if ($mockProcess -and -not $mockProcess.HasExited) {
        Stop-Process -Id $mockProcess.Id -Force
    }
}

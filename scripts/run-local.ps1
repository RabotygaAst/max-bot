param(
    [string]$MockAddr = ":1080",
    [string]$MockConfig = "mock-onec-config.json"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
Set-Location $repoRoot

if (-not (Test-Path ".\cmd\devmock\main.go")) {
    throw "Не найден .\cmd\devmock\main.go. Обновите локальную копию репозитория (например: git pull) и повторите запуск."
}

if (-not (Test-Path ".\.env.local")) {
    Copy-Item ".\.env.local.example" ".\.env.local"
    Write-Host "Создан .env.local из .env.local.example"
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

Write-Host "Запускаю локальный mock 1C/MAX на $MockAddr..."
$mockProcess = Start-Process -FilePath "go" -ArgumentList @("run", "./cmd/devmock", "-addr", $MockAddr, "-config", $MockConfig) -PassThru -NoNewWindow

try {
    Start-Sleep -Seconds 2
    Write-Host "Запускаю бота на $env:HTTP_ADDR с in-memory хранилищем..."
    go run ./cmd/bot
}
finally {
    if ($mockProcess -and -not $mockProcess.HasExited) {
        Stop-Process -Id $mockProcess.Id -Force
    }
}
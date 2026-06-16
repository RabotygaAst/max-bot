param(
    [string]$EnvFile = ".env.local",
    [switch]$BuildExe,
    [switch]$OpenFirewall
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
Set-Location $repoRoot

if (-not (Test-Path ".\cmd\bot\main.go")) {
    throw "Не найден .\cmd\bot\main.go. Запускайте скрипт из актуального репозитория max-bot."
}

if (-not (Test-Path $EnvFile)) {
    if (Test-Path ".\.env.local.example") {
        Copy-Item ".\.env.local.example" $EnvFile
        Write-Host "Создан $EnvFile из .env.local.example. Заполните токены и ONEC_BASE_URL, затем запустите скрипт еще раз." -ForegroundColor Yellow
        exit 1
    }
    throw "Не найден $EnvFile и нет .env.local.example."
}

Get-Content $EnvFile | ForEach-Object {
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

$requiredVars = @("MAX_TOKEN", "ONEC_BASE_URL", "ONEC_TOKEN", "INTERNAL_API_TOKEN")
foreach ($name in $requiredVars) {
    $value = [Environment]::GetEnvironmentVariable($name, "Process")
    if ([string]::IsNullOrWhiteSpace($value) -or $value.StartsWith("CHANGE_ME")) {
        throw "Заполните $name в $EnvFile."
    }
}

if ([string]::IsNullOrWhiteSpace($env:MAX_POLLING_ENABLED)) {
    $env:MAX_POLLING_ENABLED = "true"
}
if ([string]::IsNullOrWhiteSpace($env:HTTP_ADDR)) {
    $env:HTTP_ADDR = ":8080"
}

if ($OpenFirewall) {
    $port = "8080"
    if ($env:HTTP_ADDR -match ":(?<port>\d+)$") {
        $port = $Matches["port"]
    }
    $ruleName = "max-bot local HTTP $port"
    $existingRule = Get-NetFirewallRule -DisplayName $ruleName -ErrorAction SilentlyContinue
    if (-not $existingRule) {
        New-NetFirewallRule -DisplayName $ruleName -Direction Inbound -Action Allow -Protocol TCP -LocalPort $port | Out-Null
        Write-Host "Открыт входящий TCP-порт $port в Windows Firewall: $ruleName" -ForegroundColor Green
    }
}

$healthPort = "8080"
if ($env:HTTP_ADDR -match ":(?<port>\d+)$") {
    $healthPort = $Matches["port"]
}
Write-Host "Проверка локального health endpoint будет доступна по http://localhost:$healthPort/healthz" -ForegroundColor Cyan
Write-Host "Long polling: MAX_POLLING_ENABLED=$env:MAX_POLLING_ENABLED" -ForegroundColor Cyan
Write-Host "1C/Apache: ONEC_BASE_URL=$env:ONEC_BASE_URL" -ForegroundColor Cyan

if ($BuildExe) {
    New-Item -ItemType Directory -Force -Path ".\bin" | Out-Null
    go build -o ".\bin\max-bot.exe" .\cmd\bot
    Write-Host "Собран .\bin\max-bot.exe" -ForegroundColor Green
    & ".\bin\max-bot.exe"
} else {
    go run .\cmd\bot
}

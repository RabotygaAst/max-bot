param(
    [string]$DatabaseUrl = "",
    [int64]$MaxUserId = 123456789,
    [string]$AccountId = "ACC-000123456",
    [string]$AccountNumber = "000123456"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
Set-Location $repoRoot

function Find-Psql {
    $cmd = Get-Command "psql" -ErrorAction SilentlyContinue
    if ($cmd) { return $cmd.Source }
    $candidates = @(Get-ChildItem "C:\Program Files\PostgreSQL\*\bin\psql.exe" -ErrorAction SilentlyContinue | Sort-Object FullName -Descending)
    if ($candidates.Count -gt 0) { return $candidates[0].FullName }
    throw "psql.exe was not found. Add PostgreSQL bin directory to PATH or install PostgreSQL command line tools."
}

function Read-EnvValue([string]$Name) {
    if (-not (Test-Path ".env.local")) { return "" }
    foreach ($line in Get-Content ".env.local") {
        $trimmed = $line.Trim()
        if ($trimmed -eq "" -or $trimmed.StartsWith("#")) { continue }
        $parts = $trimmed -split "=", 2
        if ($parts.Count -eq 2 -and $parts[0].Trim() -eq $Name) { return $parts[1].Trim() }
    }
    return ""
}

function Escape-SqlLiteral([string]$Value) {
    return $Value.Replace("'", "''")
}

if ($DatabaseUrl -eq "") {
    $DatabaseUrl = Read-EnvValue "DATABASE_URL"
}
if ($DatabaseUrl -eq "") {
    $DatabaseUrl = "postgres://maxbot:maxbot_local_2026@localhost:5432/maxbot?sslmode=disable"
}

$psql = Find-Psql
$tempFile = New-TemporaryFile
try {
    $accountJson = '{"account_number":"' + (Escape-SqlLiteral $AccountNumber) + '"}'
    $sql = @"
INSERT INTO dialog_sessions (max_user_id, step, active_account_id, temp, updated_at)
VALUES ($MaxUserId, '', '$(Escape-SqlLiteral $AccountId)', '$accountJson'::jsonb, NOW())
ON CONFLICT (max_user_id) DO UPDATE
SET step = EXCLUDED.step,
    active_account_id = EXCLUDED.active_account_id,
    temp = EXCLUDED.temp,
    updated_at = NOW();

INSERT INTO max_events (event_id, status, operation_id, error_text, received_at, processed_at)
VALUES ('seed-local-$MaxUserId', 'processed', 'seed-local-op', '', NOW(), NOW())
ON CONFLICT (event_id) DO UPDATE
SET status = EXCLUDED.status,
    operation_id = EXCLUDED.operation_id,
    error_text = EXCLUDED.error_text,
    processed_at = NOW();

INSERT INTO event_logs (event_id, max_user_id, action, details, created_at)
VALUES ('seed-local-$MaxUserId', $MaxUserId, 'seed_test_user', '{"account_id":"$(Escape-SqlLiteral $AccountId)","account_number":"$(Escape-SqlLiteral $AccountNumber)"}'::jsonb, NOW());

SELECT max_user_id, step, active_account_id, temp, updated_at
FROM dialog_sessions
WHERE max_user_id = $MaxUserId;

SELECT event_id, status, operation_id, error_text, processed_at
FROM max_events
WHERE event_id = 'seed-local-$MaxUserId';
"@
    Set-Content $tempFile $sql -Encoding UTF8
    Write-Host "Using psql: $psql"
    Write-Host "Seeding local bot DB through DATABASE_URL: $DatabaseUrl"
    & $psql $DatabaseUrl -v ON_ERROR_STOP=1 -f $tempFile
    if ($LASTEXITCODE -ne 0) { throw "psql failed with exit code $LASTEXITCODE" }
}
finally {
    Remove-Item $tempFile -Force -ErrorAction SilentlyContinue
}

param([string]$DatabaseUrl = "")
Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
$repoRoot = Split-Path -Parent $PSScriptRoot
Set-Location $repoRoot
function Find-Psql { $cmd = Get-Command "psql" -ErrorAction SilentlyContinue; if ($cmd) { return $cmd.Source }; $candidates = @(Get-ChildItem "C:\Program Files\PostgreSQL\*\bin\psql.exe" -ErrorAction SilentlyContinue | Sort-Object FullName -Descending); if ($candidates.Count -gt 0) { return $candidates[0].FullName }; throw "psql.exe was not found." }
function Read-EnvValue([string]$Name) { if (-not (Test-Path ".env.local")) { return "" }; foreach ($line in Get-Content ".env.local") { $p=$line.Trim() -split "=",2; if ($p.Count -eq 2 -and $p[0].Trim() -eq $Name) { return $p[1].Trim() } }; return "" }
if ($DatabaseUrl -eq "") { $DatabaseUrl = Read-EnvValue "DATABASE_URL" }
if ($DatabaseUrl -eq "") { $DatabaseUrl = "postgres://maxbot:maxbot_local_2026@localhost:5432/maxbot?sslmode=disable" }
$psql = Find-Psql
& $psql $DatabaseUrl -v ON_ERROR_STOP=1 -f "scripts/seed-postgres-local.sql"
if ($LASTEXITCODE -ne 0) { throw "psql failed with exit code $LASTEXITCODE" }

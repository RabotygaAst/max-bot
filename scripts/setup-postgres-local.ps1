param(
    [string]$HostName = "localhost",
    [int]$Port = 5432,
    [string]$AdminUser = "postgres",
    [string]$Database = "maxbot",
    [string]$AppUser = "maxbot",
    [string]$AppPassword = "maxbot_local_2026",
    [string]$SchemaFile = "init-db.sql",
    [switch]$WriteEnvLocal
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
Set-Location $repoRoot

function Find-Psql {
    $cmd = Get-Command "psql" -ErrorAction SilentlyContinue
    if ($cmd) {
        return $cmd.Source
    }

    $candidates = Get-ChildItem "C:\Program Files\PostgreSQL\*\bin\psql.exe" -ErrorAction SilentlyContinue |
        Sort-Object FullName -Descending
    if ($candidates.Count -gt 0) {
        return $candidates[0].FullName
    }

    throw "psql.exe was not found. Add PostgreSQL bin directory to PATH or install PostgreSQL command line tools."
}

function Escape-SqlLiteral([string]$Value) {
    return $Value.Replace("'", "''")
}

function Invoke-Psql([string[]]$Arguments, [string]$Password) {
    $oldPassword = [Environment]::GetEnvironmentVariable("PGPASSWORD", "Process")
    try {
        if ($Password) {
            [Environment]::SetEnvironmentVariable("PGPASSWORD", $Password, "Process")
        }
        & $script:PsqlPath @Arguments
        if ($LASTEXITCODE -ne 0) {
            throw "psql failed with exit code $LASTEXITCODE"
        }
    }
    finally {
        [Environment]::SetEnvironmentVariable("PGPASSWORD", $oldPassword, "Process")
    }
}

if (-not (Test-Path $SchemaFile)) {
    throw "Schema file '$SchemaFile' was not found. Run this script from the repository root or keep the default path."
}

$script:PsqlPath = Find-Psql
Write-Host "Using psql: $script:PsqlPath"

$adminPassword = Read-Host "Enter password for PostgreSQL admin user '$AdminUser' (empty if passwordless)" -AsSecureString
$adminPasswordPlain = [Runtime.InteropServices.Marshal]::PtrToStringAuto([Runtime.InteropServices.Marshal]::SecureStringToBSTR($adminPassword))

[Environment]::SetEnvironmentVariable("PGPASSWORD", $adminPasswordPlain, "Process")
$roleExists = & $script:PsqlPath -h $HostName -p $Port -U $AdminUser -d postgres -tAc "SELECT 1 FROM pg_roles WHERE rolname = '$((Escape-SqlLiteral $AppUser))'"
if ($LASTEXITCODE -ne 0) {
    throw "Cannot connect to PostgreSQL as '$AdminUser'. Check password, host and port."
}

if (-not ($roleExists -match "1")) {
    Write-Host "Creating role $AppUser..."
    Invoke-Psql @("-h", $HostName, "-p", "$Port", "-U", $AdminUser, "-d", "postgres", "-v", "ON_ERROR_STOP=1", "-c", "CREATE USER `"$AppUser`" WITH PASSWORD '$((Escape-SqlLiteral $AppPassword))';") $adminPasswordPlain
} else {
    Write-Host "Role $AppUser already exists. Updating password..."
    Invoke-Psql @("-h", $HostName, "-p", "$Port", "-U", $AdminUser, "-d", "postgres", "-v", "ON_ERROR_STOP=1", "-c", "ALTER USER `"$AppUser`" WITH PASSWORD '$((Escape-SqlLiteral $AppPassword))';") $adminPasswordPlain
}

$dbExists = & $script:PsqlPath -h $HostName -p $Port -U $AdminUser -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname = '$((Escape-SqlLiteral $Database))'"
if ($LASTEXITCODE -ne 0) { throw "Cannot check database existence." }

if (-not ($dbExists -match "1")) {
    Write-Host "Creating database $Database..."
    Invoke-Psql @("-h", $HostName, "-p", "$Port", "-U", $AdminUser, "-d", "postgres", "-v", "ON_ERROR_STOP=1", "-c", "CREATE DATABASE `"$Database`" OWNER `"$AppUser`";") $adminPasswordPlain
} else {
    Write-Host "Database $Database already exists."
}

Write-Host "Applying schema from $SchemaFile..."
Invoke-Psql @("-h", $HostName, "-p", "$Port", "-U", $AppUser, "-d", $Database, "-v", "ON_ERROR_STOP=1", "-f", $SchemaFile) $AppPassword

$databaseUrl = "postgres://${AppUser}:${AppPassword}@${HostName}:${Port}/${Database}?sslmode=disable"
Write-Host ""
Write-Host "Local PostgreSQL is ready. DATABASE_URL:"
Write-Host $databaseUrl

if ($WriteEnvLocal) {
    if (-not (Test-Path ".env.local")) {
        Copy-Item ".env.local.example" ".env.local"
    }
    $content = Get-Content ".env.local" -Raw
    if ($content -match "(?m)^DATABASE_URL=") {
        $content = $content -replace "(?m)^DATABASE_URL=.*$", "DATABASE_URL=$databaseUrl"
    } else {
        $content = $content.TrimEnd() + "`r`nDATABASE_URL=$databaseUrl`r`n"
    }
    Set-Content ".env.local" $content -Encoding ASCII
    Write-Host "Updated .env.local with local DATABASE_URL."
}

@echo off
setlocal
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0seed-postgres-local.ps1" %*
exit /b %ERRORLEVEL%

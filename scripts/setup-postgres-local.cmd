@echo off
setlocal
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0setup-postgres-local.ps1" %*
exit /b %ERRORLEVEL%

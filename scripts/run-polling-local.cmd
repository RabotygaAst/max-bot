@echo off
setlocal
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0run-polling-local.ps1" %*
exit /b %ERRORLEVEL%

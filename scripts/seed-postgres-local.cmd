@echo off
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0seed-postgres-local.ps1" %*

@echo off
powershell -ExecutionPolicy Bypass -File "%~dp0seed-postgres-local.ps1" %*

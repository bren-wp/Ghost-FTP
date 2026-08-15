@echo off
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0BUILD-WINDOWS.ps1"
if errorlevel 1 pause

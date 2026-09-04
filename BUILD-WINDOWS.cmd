@echo off
setlocal

set "SCRIPT_DIR=%~dp0"
set "BUILD_SCRIPT=%SCRIPT_DIR%BUILD-WINDOWS.ps1"
set "POWERSHELL=%SystemRoot%\System32\WindowsPowerShell\v1.0\powershell.exe"

if not exist "%BUILD_SCRIPT%" (
    echo [ERROR] BUILD-WINDOWS.ps1 nije pronaden:
    echo "%BUILD_SCRIPT%"
    echo.
    pause
    exit /b 1
)

if not exist "%POWERSHELL%" (
    echo [ERROR] Windows PowerShell nije dostupan.
    echo.
    pause
    exit /b 1
)

"%POWERSHELL%" ^
    -NoLogo ^
    -NoProfile ^
    -NonInteractive ^
    -ExecutionPolicy Bypass ^
    -File "%BUILD_SCRIPT%"

set "EXIT_CODE=%ERRORLEVEL%"

if not "%EXIT_CODE%"=="0" (
    echo.
    echo [ERROR] GhostFTP build nije uspio. Exit code: %EXIT_CODE%
    echo.
    pause
    exit /b %EXIT_CODE%
)

echo.
echo [OK] GhostFTP Windows build uspjesno je dovrsen.

exit /b 0

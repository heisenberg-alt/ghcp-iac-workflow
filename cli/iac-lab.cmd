@echo off
REM IaC Lab Launcher for Windows
REM This script launches the appropriate version based on your shell

echo.
echo     IaC Lab - Infrastructure as Code Learning Experience
echo     =====================================================
echo.

REM Check if running in PowerShell-compatible environment
where pwsh >nul 2>&1
if %ERRORLEVEL% EQU 0 (
    echo Launching with PowerShell Core...
    pwsh -ExecutionPolicy Bypass -File "%~dp0iac-lab.ps1"
    goto :end
)

where powershell >nul 2>&1
if %ERRORLEVEL% EQU 0 (
    echo Launching with Windows PowerShell...
    powershell -ExecutionPolicy Bypass -File "%~dp0iac-lab.ps1"
    goto :end
)

REM Try bash (Git Bash or WSL)
where bash >nul 2>&1
if %ERRORLEVEL% EQU 0 (
    echo Launching with Bash...
    bash "%~dp0iac-lab.sh"
    goto :end
)

echo ERROR: No compatible shell found!
echo Please install one of the following:
echo   - PowerShell Core (pwsh)
echo   - Git Bash
echo   - WSL (Windows Subsystem for Linux)
pause

:end

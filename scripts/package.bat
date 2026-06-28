@echo off
chcp 65001 > nul
echo [Windows Batch] Starting project cleanup and packaging...

:: Auto navigate to repository root directory
cd /d "%~dp0\.."

set /p NAME="Enter your name for the archive file (e.g. ZhangSan, default: submission): "
if "%NAME%"=="" set NAME=submission

echo.
echo 1/3 Cleaning up temporary build artifacts and node_modules...
if exist "frontend\node_modules" rd /s /q "frontend\node_modules"
if exist "frontend\dist" rd /s /q "frontend\dist"
if exist "engine\data" rd /s /q "engine\data"
if exist "engine\test_data" rd /s /q "engine\test_data"
if exist "bin" rd /s /q "bin"
del /f /q "*.exe" "engine\*.exe" "*.log" "*.pid" 2>nul

echo 2/3 Executing PowerShell compression script...
powershell -ExecutionPolicy Bypass -File "%~dp0package.ps1" -Name "%NAME%"

pause

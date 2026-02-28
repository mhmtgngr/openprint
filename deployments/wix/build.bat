@echo off
REM Build script for OpenPrint Windows Agent
REM This script builds the Go executable and creates an MSI installer

setlocal enabledelayedexpansion

echo ========================================
echo OpenPrint Agent Build Script
echo ========================================

REM Configuration
set AGENT_VERSION=1.0.0
set PRODUCT_GUID={YOUR-GUID-HERE-0000-0000-000000000001}
set AGENT_EXE_GUID={YOUR-GUID-HERE-0000-0000-000000000002}
set CONFIG_GUID={YOUR-GUID-HERE-0000-0000-000000000003}
set CONFIG_DIR_GUID={YOUR-GUID-HERE-0000-0000-000000000004}
set SHORTCUT_GUID={YOUR-GUID-HERE-0000-0000-000000000005}
set FIREWALL_GUID={YOUR-GUID-HERE-0000-0000-000000000006}

REM Set build directories
set SCRIPT_DIR=%~dp0
set PROJECT_ROOT=%SCRIPT_DIR%..\..
set AGENT_DIR=%PROJECT_ROOT%\cmd\agent
set BUILD_DIR=%PROJECT_ROOT%\build
set OUTPUT_DIR=%BUILD_DIR%\installer

echo.
echo [1/5] Cleaning build directories...
if exist "%BUILD_DIR%" rmdir /s /q "%BUILD_DIR%"
mkdir "%BUILD_DIR%"
mkdir "%OUTPUT_DIR%"

echo.
echo [2/5] Building Go executable...
cd /d "%AGENT_DIR%"

REM Set Go build flags for Windows
set CGO_ENABLED=1
set GOOS=windows
set GOARCH=amd64

REM Build with Windows GUI subsystem (no console)
go build -ldflags "-s -w -H windowsgui" -o "%BUILD_DIR%\agent.exe" .

if errorlevel 1 (
    echo ERROR: Failed to build agent.exe
    exit /b 1
)

echo [3/5] Copying configuration template...
copy nul "%OUTPUT_DIR%\config.json"

echo [4/5] Copying WiX source...
copy "%SCRIPT_DIR%\OpenPrintAgent.wxs" "%BUILD_DIR%\OpenPrintAgent.wxs"

echo.
echo [5/5] Generating MSI installer...
cd /d "%BUILD_DIR%"

REM Generate GUIDs dynamically
for /f %%i in ('powershell -Command "[Guid]::NewGuid()"') do set UPGRADE_GUID=%%i
for /f %%i in ('powershell -Command "[Guid]::NewGuid()"') do set AGENT_EXE_GUID=%%i
for /f %%i in ('powershell -Command "[Guid]::NewGuid()"') do set CONFIG_GUID=%%i
for /f %%i in ('powershell -Command "[Guid]::NewGuid()"') do set CONFIG_DIR_GUID=%%i
for /f %%i in ('powershell -Command "[Guid]::NewGuid()"') do set SHORTCUT_GUID=%%i
for /f %%i in ('powershell -Command "[Guid]::NewGuid()"') do set FIREWALL_GUID=%%i

REM Replace GUID placeholders in WiX file
powershell -Command "(Get-Content OpenPrintAgent.wxs) -replace 'GUID-PLACEHOLDER-UPGRADE-CODE', '%UPGRADE_GUID%' | Set-Content OpenPrintAgent_Parsed.wxs"
powershell -Command "(Get-Content OpenPrintAgent_Parsed.wxs) -replace 'GUID-PLACEHOLDER-AGENT-EXE', '%AGENT_EXE_GUID%' | Set-Content OpenPrintAgent.wxs"
powershell -Command "(Get-Content OpenPrintAgent.wxs) -replace 'GUID-PLACEHOLDER-CONFIG', '%CONFIG_GUID%' | Set-Content OpenPrintAgent_Temp.wxs"
powershell -Command "(Get-Content OpenPrintAgent_Temp.wxs) -replace 'GUID-PLACEHOLDER-CONFIG-DIR', '%CONFIG_DIR_GUID%' | Set-Content OpenPrintAgent.wxs"
powershell -Command "(Get-Content OpenPrintAgent.wxs) -replace 'GUID-PLACEHOLDER-SHORTCUT', '%SHORTCUT_GUID%' | Set-Content OpenPrintAgent_Temp.wxs"
powershell -Command "(Get-Content OpenPrintAgent_Temp.wxs) -replace 'GUID-PLACEHOLDER-FIREWALL', '%FIREWALL_GUID%' | Set-Content OpenPrintAgent.wxs"

REM Check if candle.exe (WiX compiler) is available
where candle.exe >nul 2>&1
if errorlevel 1 (
    echo WARNING: WiX Toolset not found in PATH
    echo Please install WiX Toolset from https://wixtoolset.org/
    echo.
    echo Agent executable is available at: %BUILD_DIR%\agent.exe
    echo You can manually install the agent as a service using:
    echo   sc create OpenPrintAgent binPath= "%BUILD_DIR%\agent.exe" start= auto
    echo   sc start OpenPrintAgent
    exit /b 0
)

REM Compile WiX source
candle.exe OpenPrintAgent.wxs -ext WixUtilExtension
if errorlevel 1 (
    echo ERROR: WiX compilation failed
    exit /b 1
)

REM Link to create MSI
light.exe OpenPrintAgent.wixobj -out "%OUTPUT_DIR%\OpenPrintAgent-%AGENT_VERSION%.msi" -ext WixUtilExtension
if errorlevel 1 (
    echo ERROR: MSI linking failed
    exit /b 1
)

echo.
echo ========================================
echo Build Complete!
echo ========================================
echo.
echo Installer: %OUTPUT_DIR%\OpenPrintAgent-%AGENT_VERSION%.msi
echo Executable: %BUILD_DIR%\agent.exe
echo.
echo To install the agent:
echo   1. Run the MSI installer, or
echo   2. Install manually: sc create OpenPrintAgent binPath= "%BUILD_DIR%\agent.exe" start= auto
echo.

cd /d "%SCRIPT_DIR%"
endlocal

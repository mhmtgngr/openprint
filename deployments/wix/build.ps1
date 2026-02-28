# Build script for OpenPrint Windows Agent (PowerShell version)
# This script builds the Go executable and creates an MSI installer

param(
    [string]$Version = "1.0.0",
    [switch]$SkipInstaller = $false
)

$ErrorActionPreference = "Stop"

$AgentVersion = $Version
$ProjectRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$AgentDir = Join-Path $ProjectRoot "cmd\agent"
$BuildDir = Join-Path $ProjectRoot "build"
$OutputDir = Join-Path $BuildDir "installer"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "OpenPrint Agent Build Script" -ForegroundColor Cyan
Write-Host "Version: $AgentVersion" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Step 1: Clean build directories
Write-Host "[1/5] Cleaning build directories..." -ForegroundColor Yellow
if (Test-Path $BuildDir) {
    Remove-Item -Path $BuildDir -Recurse -Force
}
New-Item -ItemType Directory -Path $BuildDir -Force | Out-Null
New-Item -ItemType Directory -Path $OutputDir -Force | Out-Null

# Step 2: Build Go executable
Write-Host "[2/5] Building Go executable..." -ForegroundColor Yellow
$env:CGO_ENABLED = "1"
$env:GOOS = "windows"
$env:GOARCH = "amd64"

$AgentExePath = Join-Path $BuildDir "agent.exe"
$BuildArgs = @(
    "build"
    "-ldflags", "-s -w -H windowsgui"
    "-o", "`"$AgentExePath`""
    "`"$AgentDir`""
)

& go @BuildArgs
if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Failed to build agent.exe" -ForegroundColor Red
    exit 1
}

# Step 3: Copy configuration template
Write-Host "[3/5] Copying configuration template..." -ForegroundColor Yellow
$ConfigContent = @{
    server_url = "http://localhost:8003"
    agent_name = $env:COMPUTERNAME
    enrollment_token = ""
    organization_id = ""
    heartbeat_interval_seconds = 30
    job_poll_interval_seconds = 10
    max_retry_count = 3
    log_level = "info"
} | ConvertTo-Json

$ConfigPath = Join-Path $OutputDir "config.json"
$ConfigContent | Out-File -FilePath $ConfigPath -Encoding UTF8

# Step 4: Prepare WiX source
Write-Host "[4/5] Preparing WiX source..." -ForegroundColor Yellow
$WixSourcePath = Join-Path $PSScriptRoot "OpenPrintAgent.wxs"
$WixBuildPath = Join-Path $BuildDir "OpenPrintAgent.wxs"

# Generate new GUIDs for the installer
$UpgradeCode = [Guid]::NewGuid()
$AgentExeGuid = [Guid]::NewGuid()
$ConfigGuid = [Guid]::NewGuid()
$ConfigDirGuid = [Guid]::NewGuid()
$$ShortcutGuid = [Guid]::NewGuid()
$FirewallGuid = [Guid]::NewGuid()

# Read and replace placeholders
$WixContent = Get-Content $WixSourcePath -Raw
$WixContent = $WixContent -replace 'GUID-PLACEHOLDER-UPGRADE-CODE', $UpgradeCode
$WixContent = $WixContent -replace 'GUID-PLACEHOLDER-AGENT-EXE', $AgentExeGuid
$WixContent = $WixContent -replace 'GUID-PLACEHOLDER-CONFIG', $ConfigGuid
$WixContent = $WixContent -replace 'GUID-PLACEHOLDER-CONFIG-DIR', $ConfigDirGuid
$WixContent = $WixContent -replace 'GUID-PLACEHOLDER-SHORTCUT', $ShortcutGuid
$WixContent = $WixContent -replace 'GUID-PLACEHOLDER-FIREWALL', $FirewallGuid
$WixContent = $WixContent -replace '1\.0\.0\.0', "$AgentVersion.0"

$WixContent | Out-File -FilePath $WixBuildPath -Encoding UTF8

# Step 5: Build MSI installer
if (-not $SkipInstaller) {
    Write-Host "[5/5] Building MSI installer..." -ForegroundColor Yellow

    # Check if WiX is available
    $CandlePath = Get-Command "candle.exe" -ErrorAction SilentlyContinue
    $LightPath = Get-Command "light.exe" -ErrorAction SilentlyContinue

    if ($null -eq $CandlePath -or $null -eq $LightPath) {
        Write-Host "WARNING: WiX Toolset not found in PATH" -ForegroundColor Yellow
        Write-Host "Please install WiX Toolset from https://wixtoolset.org/" -ForegroundColor Yellow
        Write-Host ""
        Write-Host "Agent executable is available at: $AgentExePath" -ForegroundColor Green
        Write-Host ""
        Write-Host "To install the agent as a service manually:" -ForegroundColor Cyan
        Write-Host "  New-Service -Name `"OpenPrintAgent`" -BinaryPathName `"$AgentExePath`" -StartupType Automatic" -ForegroundColor White
        Write-Host "  Start-Service -Name `"OpenPrintAgent`"" -ForegroundColor White
    } else {
        # Compile WiX source
        $CandleArgs = @(
            "`"$WixBuildPath`""
            "-ext", "WixUtilExtension"
            "-out", "`"$BuildDir\OpenPrintAgent.wixobj`""
        )
        & candle.exe @CandleArgs

        if ($LASTEXITCODE -ne 0) {
            Write-Host "ERROR: WiX compilation failed" -ForegroundColor Red
            exit 1
        }

        # Link to create MSI
        $MsiPath = Join-Path $OutputDir "OpenPrintAgent-$AgentVersion.msi"
        $LightArgs = @(
            "`"$BuildDir\OpenPrintAgent.wixobj`""
            "-out", "`"$MsiPath`""
            "-ext", "WixUtilExtension"
        )
        & light.exe @LightArgs

        if ($LASTEXITCODE -ne 0) {
            Write-Host "ERROR: MSI linking failed" -ForegroundColor Red
            exit 1
        }

        Write-Host ""
        Write-Host "========================================" -ForegroundColor Green
        Write-Host "Build Complete!" -ForegroundColor Green
        Write-Host "========================================" -ForegroundColor Green
        Write-Host ""
        Write-Host "Installer: $MsiPath" -ForegroundColor Cyan
        Write-Host "Executable: $AgentExePath" -ForegroundColor Cyan
        Write-Host ""
        Write-Host "To install the agent:" -ForegroundColor Yellow
        Write-Host "  1. Run the MSI installer, or" -ForegroundColor White
        Write-Host "  2. Install as service: New-Service -Name `"OpenPrintAgent`" -BinaryPathName `"$AgentExePath`"" -ForegroundColor White
    }
} else {
    Write-Host "[5/5] Skipping MSI installer (SkipInstaller specified)" -ForegroundColor Yellow
}

Write-Host ""

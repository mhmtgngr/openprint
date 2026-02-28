# OpenPrint Windows Agent

The OpenPrint Windows Agent is a Windows service that connects to the OpenPrint Cloud platform to manage print jobs on Windows machines.

## Features

- **Windows Service Integration**: Runs as a native Windows service with automatic startup
- **Printer Discovery**: Automatically discovers all printers configured on the Windows machine
- **Secure Enrollment**: Uses enrollment tokens for secure agent registration
- **Job Polling**: Continuously polls for new print jobs from the cloud platform
- **Automatic Updates**: Supports remote configuration updates and commands
- **Event Logging**: Integrates with Windows Event Log for troubleshooting
- **Firewall Configuration**: Automatically configures Windows Firewall exceptions

## Building the Agent

### Prerequisites

1. **Go 1.24+**: Install from https://golang.org/dl/
2. **WiX Toolset** (for MSI installer): Install from https://wixtoolset.org/
3. **Windows SDK**: Required for Windows service compilation

### Build Steps

#### Option 1: Using PowerShell (Recommended)

```powershell
cd deployments\wix
.\build.ps1
```

#### Option 2: Using Batch Script

```cmd
cd deployments\wix
build.bat
```

#### Option 3: Build Executable Only

```cmd
cd cmd\agent
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=1
go build -ldflags "-s -w -H windowsgui" -o agent.exe .
```

## Installation

### Method 1: MSI Installer (Recommended)

1. Double-click `OpenPrintAgent-1.0.0.msi`
2. Follow the installation wizard
3. Configure the agent by editing `C:\ProgramData\OpenPrint\agent\config.json`
4. Start the service from Services.msc or it will start automatically

### Method 2: Manual Installation

1. Copy `agent.exe` to `C:\Program Files\OpenPrint\Agent\`
2. Create configuration file at `C:\ProgramData\OpenPrint\agent\config.json`:
   ```json
   {
     "server_url": "https://your-openprint-server.com",
     "agent_name": "MyComputer",
     "enrollment_token": "your-enrollment-token",
     "organization_id": "your-org-id"
   }
   ```
3. Install the service:
   ```cmd
   sc create OpenPrintAgent binPath= "C:\Program Files\OpenPrint\Agent\agent.exe" start= auto
   sc start OpenPrintAgent
   ```

### Method 3: PowerShell Installation

```powershell
New-Service -Name "OpenPrintAgent" `
    -BinaryPathName "C:\Program Files\OpenPrint\Agent\agent.exe" `
    -StartupType Automatic
Start-Service -Name "OpenPrintAgent"
```

## Configuration

The agent configuration is stored in `C:\ProgramData\OpenPrint\agent\config.json`:

```json
{
  "server_url": "https://print.yourcompany.com",
  "agent_id": "auto-generated-after-registration",
  "agent_name": "ComputerName",
  "enrollment_token": "optional-token-for-initial-registration",
  "organization_id": "your-organization-id",
  "heartbeat_interval_seconds": 30,
  "job_poll_interval_seconds": 10,
  "max_retry_count": 3,
  "log_level": "info"
}
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `server_url` | string | required | URL of the OpenPrint server |
| `agent_id` | string | auto-generated | Unique agent ID (auto-assigned) |
| `agent_name` | string | hostname | Friendly name for the agent |
| `enrollment_token` | string | - | Token for initial registration |
| `organization_id` | string | - | Organization to join |
| `heartbeat_interval_seconds` | int | 30 | How often to send heartbeat |
| `job_poll_interval_seconds` | int | 10 | How often to check for new jobs |
| `max_retry_count` | int | 3 | Maximum retry attempts for failed jobs |
| `log_level` | string | info | Logging level (debug, info, warn, error) |

## Service Management

### Start/Stop the Service

**Services MMC:**
1. Press Win+R, type `services.msc`
2. Find "OpenPrint Cloud Print Agent"
3. Right-click and select Start/Stop

**PowerShell:**
```powershell
Stop-Service -Name "OpenPrintAgent"
Start-Service -Name "OpenPrintAgent"
Restart-Service -Name "OpenPrintAgent"
```

**Command Prompt:**
```cmd
net stop OpenPrintAgent
net start OpenPrintAgent
```

### View Service Status

```powershell
Get-Service -Name "OpenPrintAgent"
```

### Uninstall the Service

```cmd
sc stop OpenPrintAgent
sc delete OpenPrintAgent
```

Or use the Programs and Features in Control Panel to uninstall the MSI.

## Troubleshooting

### View Event Logs

1. Open Event Viewer (`eventvwr.msc`)
2. Navigate to **Windows Logs** > **Application**
3. Filter by Source "OpenPrintAgent"

### Service Won't Start

1. Check if the configuration file exists at `C:\ProgramData\OpenPrint\agent\config.json`
2. Verify the `server_url` is correct
3. Check Windows Event Log for error messages
4. Ensure network connectivity to the server

### Printers Not Discovered

1. Ensure printers are installed in Windows
2. Run `Get-Printer` in PowerShell to verify printers are available
3. Check Event Log for discovery errors

### Jobs Not Printing

1. Verify the printer is online and has paper
2. Check that the printer driver is correctly installed
3. Test print directly from Windows first
4. Review job status in the OpenPrint web dashboard

## Security Considerations

- The agent runs as LocalSystem by default for printer access
- HTTPS should be used for `server_url` in production
- Enrollment tokens should be treated as sensitive credentials
- Firewall rules are automatically created during installation
- Consider creating a dedicated service account with minimum required permissions

## Firewall Configuration

The installer automatically creates Windows Firewall rules. Manual configuration:

```powershell
New-NetFirewallRule -DisplayName "OpenPrint Agent" `
    -Direction Outbound `
    -Program "C:\Program Files\OpenPrint\Agent\agent.exe" `
    -Action Allow
```

## Silent Installation

For deployment via Group Policy or SCCM:

```cmd
msiexec /i OpenPrintAgent-1.0.0.msi /qn /norestart SERVER_URL="https://print.company.com" ENROLLMENT_TOKEN="your-token"
```

## Support

For issues and support:
- Documentation: https://docs.openprint.com
- Issues: https://github.com/openprint/openprint/issues

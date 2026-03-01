# Guacamole Printer Configuration Guide

## Important: Guacamole Printing Limitations

Apache Guacamole does **NOT** support native RAW printer passthrough. When you set
`enable-printing=true` on a Guacamole RDP connection, it creates a **virtual PDF printer**
inside the RDP session. Print output is converted to PDF and offered as a browser download.

There is no mechanism in Guacamole to redirect a client's physical printer into the RDP
session for direct RAW printing.

For native printing through OpenPrint, use the **Agent-based architecture** described below.

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                     WINDOWS SERVER (RDP Host)                   │
│                                                                 │
│  ┌──────────────┐    ┌──────────────────┐    ┌───────────────┐  │
│  │  Windows App  │───▶│  Windows Print   │───▶│  OpenPrint    │  │
│  │  (.exe)       │    │  Spooler         │    │  Agent        │  │
│  └──────────────┘    └──────────────────┘    └──────┬────────┘  │
│                                                      │          │
│  ┌──────────────────────────────────────────┐        │          │
│  │  Guacamole (guacd) - browser RDP access  │        │          │
│  │  (optional, for remote desktop only)     │        │          │
│  └──────────────────────────────────────────┘        │          │
└──────────────────────────────────────────────────────┼──────────┘
                                                       │
                                              HTTPS / REST API
                                                       │
                                                       ▼
                                          ┌────────────────────┐
                                          │  OpenPrint Cloud   │
                                          │  (Job Routing)     │
                                          └────────┬───────────┘
                                                   │
                                          ┌────────┴───────────┐
                                          ▼                    ▼
                                  ┌──────────────┐    ┌──────────────┐
                                  │ Network      │    │ Client PC    │
                                  │ Printer (IP) │    │ + Agent      │
                                  │              │    │ + USB Printer│
                                  └──────────────┘    └──────────────┘
```

---

## Windows Server Components (RDP Host)

The Windows Server is where users run applications via RDP (optionally through Guacamole).

### Required Components

| # | Component | Version | How to Install | Purpose |
|---|-----------|---------|----------------|---------|
| 1 | Windows Server | 2016+ | OS install | Base operating system |
| 2 | Remote Desktop Services | Built-in | Server Manager → Add Roles | Allows RDP connections |
| 3 | Print and Document Services | Built-in | Server Manager → Add Roles | Windows print subsystem |
| 4 | PowerShell | 5.1+ (built-in) | Pre-installed | `Get-Printer` cmdlet for discovery |
| 5 | .NET Framework | 4.7.2+ (built-in) | Pre-installed | `System.Drawing.Printing` for print execution |
| 6 | **OpenPrint Agent** | Latest | MSI installer / manual | Printer discovery, job polling, print execution |
| 7 | Printer Drivers | Per-printer | Vendor installer or Windows Update | Drivers for all target printers |

### Installation Steps

#### 1. Enable Print and Document Services Role

```powershell
# PowerShell (Run as Administrator)
Install-WindowsFeature Print-Services -IncludeManagementTools
Install-WindowsFeature Print-Server
```

#### 2. Enable Remote Desktop Services

```powershell
Install-WindowsFeature RDS-RD-Server
# Allow RDP connections
Set-ItemProperty -Path 'HKLM:\System\CurrentControlSet\Control\Terminal Server' -Name "fDenyTSConnections" -Value 0
Enable-NetFirewallRule -DisplayGroup "Remote Desktop"
```

#### 3. Configure Printer Redirection Policy (for RDP clients)

```powershell
# Allow client printer redirection (for non-Guacamole RDP clients)
Set-ItemProperty -Path 'HKLM:\SOFTWARE\Policies\Microsoft\Windows NT\Terminal Services' `
    -Name "fDisableCpm" -Value 0 -Type DWord

# Auto-install printer drivers for redirected printers
Set-ItemProperty -Path 'HKLM:\SOFTWARE\Policies\Microsoft\Windows NT\Terminal Services' `
    -Name "fForceClientLptDef" -Value 1 -Type DWord
```

#### 4. Install Printer Drivers

```powershell
# List installed printer drivers
Get-PrinterDriver | Select-Object Name, Manufacturer

# Add a printer driver (example: HP Universal)
Add-PrinterDriver -Name "HP Universal Printing PCL 6"

# Add a network printer
Add-Printer -ConnectionName "\\printserver\SharedPrinter"

# Add an IP printer
Add-PrinterPort -Name "IP_192.168.1.100" -PrinterHostAddress "192.168.1.100"
Add-Printer -Name "Office-HP-LaserJet" -DriverName "HP Universal Printing PCL 6" -PortName "IP_192.168.1.100"
```

#### 5. Install OpenPrint Agent

```powershell
# Create config directory
New-Item -ItemType Directory -Force -Path "C:\ProgramData\OpenPrint"

# Create agent configuration
@{
    server_url = "https://your-openprint-cloud.example.com"
    agent_name = "PrintServer-01"
    enrollment_token = "your-enrollment-token"
    organization_id = "your-org-id"
    heartbeat_interval_seconds = 30
    job_poll_interval_seconds = 10
    log_level = "info"
} | ConvertTo-Json | Set-Content "C:\ProgramData\OpenPrint\config.json"

# Install as Windows service
sc.exe create OpenPrintAgent binPath= "C:\Program Files\OpenPrint\openprint-agent.exe" start= auto
sc.exe description OpenPrintAgent "OpenPrint Cloud Print Management Agent"
sc.exe start OpenPrintAgent
```

#### 6. Verify Agent Printer Discovery

```powershell
# Check what the agent will discover
Get-Printer | Select-Object Name, DriverName, PortName, PrinterStatus, Shared | Format-Table

# Expected output:
# Name                  DriverName                    PortName         PrinterStatus Shared
# ----                  ----------                    --------         ------------- ------
# HP LaserJet 4200      HP Universal Printing PCL 6   IP_192.168.1.10  Normal        True
# Canon iR-ADV C5535    Canon Generic Plus UFR II      IP_192.168.1.20  Normal        False
```

### Optional: Guacamole (for browser-based RDP access only)

If you need browser-based remote desktop access (not for printing):

```xml
<!-- Guacamole connection config (user-mapping.xml or database) -->
<connection name="PrintServer">
    <protocol>rdp</protocol>
    <param name="hostname">192.168.x.x</param>
    <param name="port">3389</param>
    <param name="username">user</param>
    <param name="password">password</param>
    <param name="security">nla</param>

    <!-- PDF virtual printer only - NOT native printing -->
    <param name="enable-printing">true</param>
    <param name="printer-name">Guacamole-PDF</param>
</connection>
```

> **Note**: The `enable-printing` parameter only creates a virtual PDF printer.
> Users will see "Guacamole-PDF" in the print dialog. Output is PDF download only.
> For native printing, use the OpenPrint Agent (installed above).

---

## Windows Client Components (Print Endpoint)

The Windows Client is where the physical printer is connected. There are three scenarios:

### Scenario A: Network Printers (most common)

No client-side software needed. The OpenPrint Agent on the server sends print jobs
directly to the network printer via IP.

| # | Component | Purpose |
|---|-----------|---------|
| 1 | Network printer with static IP | Direct IP printing |
| 2 | Printer driver on **server** | Server renders and sends data |
| 3 | Network connectivity (server ↔ printer) | TCP/IP communication (port 9100/IPP) |

### Scenario B: Local/USB Printers on Client PCs

When printers are physically connected to user workstations (USB, parallel, etc.),
install a second OpenPrint Agent on the client to receive routed jobs.

| # | Component | Version | How to Install | Purpose |
|---|-----------|---------|----------------|---------|
| 1 | Windows | 10/11 or Server | OS install | Client OS |
| 2 | **OpenPrint Agent** | Latest | MSI installer | Receives jobs, manages local printers |
| 3 | Printer Drivers | Per-printer | Vendor installer | Drivers for USB/local printers |
| 4 | PowerShell | 5.1+ (built-in) | Pre-installed | Printer discovery |
| 5 | .NET Framework | 4.7.2+ (built-in) | Pre-installed | Print execution |

```powershell
# Install OpenPrint Agent on client
@{
    server_url = "https://your-openprint-cloud.example.com"
    agent_name = "Client-Workstation-01"
    enrollment_token = "your-enrollment-token"
    organization_id = "your-org-id"
    heartbeat_interval_seconds = 30
    job_poll_interval_seconds = 10
    log_level = "info"
} | ConvertTo-Json | Set-Content "C:\ProgramData\OpenPrint\config.json"

sc.exe create OpenPrintAgent binPath= "C:\Program Files\OpenPrint\openprint-agent.exe" start= auto
sc.exe start OpenPrintAgent
```

### Scenario C: Thin Clients / Chromebooks (Guacamole PDF fallback)

When users access via browser through Guacamole and have no agent installed:

| # | Component | Purpose |
|---|-----------|---------|
| 1 | Web browser (Chrome, Firefox, Edge) | Access Guacamole web UI |
| 2 | PDF viewer | Open downloaded PDF |
| 3 | Local printer + driver | Manual "print PDF" step |

> **Limitation**: This requires the user to manually print the downloaded PDF.
> It is not automated native printing.

---

## Print Flow Comparison

### OpenPrint Agent (Recommended - Native RAW)

```
User clicks Print in App
        │
        ▼
Windows Print Spooler on Server
        │
        ▼
OpenPrint Agent detects job (polls every 10s)
        │
        ▼
Agent sends job status to OpenPrint Cloud
        │
        ▼
Cloud routes job to target agent/printer
        │
        ▼
Target Agent executes print via PowerShell
        │
        ▼
Physical printer outputs document ✓
```

### Guacamole Built-in (PDF Only)

```
User clicks Print in App
        │
        ▼
Selects "Guacamole-PDF" virtual printer
        │
        ▼
Guacamole converts to PDF
        │
        ▼
PDF downloaded in browser
        │
        ▼
User manually opens PDF and prints locally ✗ (not automated)
```

### True RDP Redirect (Requires native RDP client, NOT Guacamole)

```
User connects via mstsc.exe / FreeRDP (with printer redirect enabled)
        │
        ▼
Local printers appear as "PrinterName (Redirected N)"
        │
        ▼
User prints to redirected printer
        │
        ▼
RDP virtual channel sends data to client
        │
        ▼
Client's local spooler prints to physical printer ✓
```

---

## Guacamole Parameters Reference

Valid Guacamole RDP printing parameters (for PDF virtual printer only):

| Parameter | Type | Description |
|-----------|------|-------------|
| `enable-printing` | boolean | Enable/disable virtual PDF printer |
| `printer-name` | string | Name of virtual printer shown in print dialog |

**Invalid / Non-existent parameters:**

| Parameter | Status |
|-----------|--------|
| `disable-printing` | Does NOT exist |
| `printer-driver` | Does NOT exist for RDP connections |

---

## Troubleshooting

### Agent not discovering printers

```powershell
# Verify Print Spooler service is running
Get-Service Spooler | Select-Object Status, StartType

# Verify PowerShell can see printers
Get-Printer | Format-Table Name, DriverName, PortName, PrinterStatus

# Check agent logs
Get-Content "C:\ProgramData\OpenPrint\agent.log" -Tail 50
```

### Printer shows as offline

```powershell
# Check printer port connectivity (network printers)
Test-NetConnection -ComputerName 192.168.1.100 -Port 9100

# Check printer status
Get-Printer -Name "PrinterName" | Select-Object PrinterStatus

# Restart Print Spooler
Restart-Service Spooler
```

### Guacamole PDF printer not appearing

```bash
# Check guacd is running
systemctl status guacd

# Verify connection config includes enable-printing
grep -r "enable-printing" /etc/guacamole/

# Check guacd logs
journalctl -u guacd -f
```

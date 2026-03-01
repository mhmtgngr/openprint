# Guacamole + OpenPrint Native Printer Configuration Guide

## Important: Guacamole Printing Limitations

Apache Guacamole does **NOT** support native RAW printer passthrough. When you set
`enable-printing=true` on a Guacamole RDP connection, it creates a **virtual PDF printer**
inside the RDP session. Print output is converted to PDF and offered as a browser download.

OpenPrint solves this with an **agent-based architecture** that captures print jobs on the
RDP session host and routes them to the user's local printer via the cloud.

---

## Architecture: How It Works

```
┌──────────────────────────────────────────────────────────────────────────┐
│                 WINDOWS SERVER (RDP Session Host)                        │
│                                                                          │
│  ┌──────────────┐    ┌──────────────┐    ┌────────────────────────────┐  │
│  │ Windows App  │    │ Windows      │    │ OpenPrint Agent            │  │
│  │ (.exe)       │───▶│ Print        │───▶│ (role: server)             │  │
│  └──────────────┘    │ Spooler      │    │                            │  │
│                      └──────────────┘    │ 1. Creates virtual printer │  │
│  ┌──────────────────────────┐            │    "OpenPrint"             │  │
│  │ Guacamole (guacd)        │            │ 2. TCP listener :9100     │  │
│  │ Browser-based RDP access │            │ 3. Captures spool data    │  │
│  └──────────────────────────┘            │ 4. Identifies RDP user    │  │
│                                          │ 5. Uploads to cloud       │  │
│                                          └─────────┬──────────────────┘  │
└────────────────────────────────────────────────────┼─────────────────────┘
                                                     │
                                            HTTPS / REST API
                                                     │
                                                     ▼
                                        ┌──────────────────────┐
                                        │  OpenPrint Cloud     │
                                        │                      │
                                        │  - storage-service   │
                                        │    (stores document) │
                                        │  - job-service       │
                                        │    (routes by user)  │
                                        │  - registry-service  │
                                        │    (user→printer     │
                                        │     mappings)        │
                                        └──────────┬───────────┘
                                                   │
                                          Job routed via
                                          user_printer_mappings
                                                   │
                                                   ▼
                                        ┌──────────────────────┐
                                        │ CLIENT WORKSTATION   │
                                        │                      │
                                        │ OpenPrint Agent      │
                                        │ (role: client)       │
                                        │                      │
                                        │ 1. Polls for jobs    │
                                        │ 2. Downloads document│
                                        │ 3. Prints to local   │
                                        │    USB/IP printer    │
                                        └──────────────────────┘
```

### Step-by-step flow:

1. User opens browser, connects to Windows Server via **Guacamole** (RDP)
2. User runs Windows application, clicks **Print**
3. In the print dialog, user selects **"OpenPrint"** virtual printer
4. Windows Spooler sends print data to the OpenPrint Agent's TCP listener (localhost:9100)
5. **Server Agent** captures the raw data, identifies the user (via spooler metadata + RDP session)
6. Server Agent uploads the document to **storage-service** and creates a job in **job-service**
7. Job is tagged with `printer_id: __user_default__` and the user's email
8. **Client Agent** (on user's workstation) polls job-service for new jobs
9. Job-service matches the job to the client agent via **user_printer_mappings** table
10. Client Agent downloads the document and prints to the user's local physical printer

---

## Setup Guide

### Step 1: Deploy OpenPrint Cloud Services

Ensure all OpenPrint cloud services are running (see main README):

```bash
cd deployments/docker
docker-compose up -d
```

Run the database migration for user-printer mappings:

```bash
migrate -path migrations -database "postgres://openprint:openprint@localhost:5432/openprint?sslmode=disable" up
```

### Step 2: Install Server Agent (RDP Session Host)

On the Windows Server where users connect via Guacamole RDP:

```powershell
# Create config directory
New-Item -ItemType Directory -Force -Path "C:\ProgramData\OpenPrint\agent"

# Create server agent configuration
@{
    server_url = "https://your-openprint-cloud.example.com"
    agent_name = "RDP-Server-01"
    agent_role = "server"
    virtual_printer_name = "OpenPrint"
    print_listen_port = 9100
    storage_service_url = "https://your-openprint-cloud.example.com"
    enrollment_token = "your-enrollment-token"
    organization_id = "your-org-id"
    heartbeat_interval_seconds = 30
    job_poll_interval_seconds = 10
    log_level = "info"
} | ConvertTo-Json | Set-Content "C:\ProgramData\OpenPrint\agent\config.json"

# Install as Windows service
sc.exe create OpenPrintAgent binPath= "C:\Program Files\OpenPrint\openprint-agent.exe" start= auto
sc.exe description OpenPrintAgent "OpenPrint Cloud Print Agent (Server Mode)"
sc.exe start OpenPrintAgent
```

The server agent will automatically:
- Create a virtual printer called "OpenPrint" using the "Generic / Text Only" driver
- Create a TCP/IP port pointing to localhost:9100
- Start listening for print data on port 9100
- Discover existing printers on the server

#### Required Windows Components (Server)

| Component | How to Install |
|-----------|----------------|
| Print and Document Services | `Install-WindowsFeature Print-Services -IncludeManagementTools` |
| Remote Desktop Services | `Install-WindowsFeature RDS-RD-Server` |
| PowerShell 5.1+ | Pre-installed |
| .NET Framework 4.7.2+ | Pre-installed |
| "Generic / Text Only" printer driver | Pre-installed (Windows built-in) |

### Step 3: Install Client Agent (User Workstation)

On each user's Windows workstation where the physical printer is connected:

```powershell
# Create config directory
New-Item -ItemType Directory -Force -Path "C:\ProgramData\OpenPrint\agent"

# Create client agent configuration
@{
    server_url = "https://your-openprint-cloud.example.com"
    agent_name = "Workstation-User01"
    agent_role = "client"
    enrollment_token = "your-enrollment-token"
    organization_id = "your-org-id"
    heartbeat_interval_seconds = 30
    job_poll_interval_seconds = 10
    log_level = "info"
} | ConvertTo-Json | Set-Content "C:\ProgramData\OpenPrint\agent\config.json"

# Install as Windows service
sc.exe create OpenPrintAgent binPath= "C:\Program Files\OpenPrint\openprint-agent.exe" start= auto
sc.exe description OpenPrintAgent "OpenPrint Cloud Print Agent (Client Mode)"
sc.exe start OpenPrintAgent
```

The client agent will automatically:
- Discover local printers (USB, network, shared)
- Register them with OpenPrint Cloud
- Poll for print jobs targeted at this agent
- Execute print jobs on local printers

#### Required Windows Components (Client)

| Component | How to Install |
|-----------|----------------|
| PowerShell 5.1+ | Pre-installed |
| .NET Framework 4.7.2+ | Pre-installed |
| Printer drivers | Vendor installer for each connected printer |

### Step 4: Create User-Printer Mapping

Map each user to their local printer/agent. This tells the system where to route
captured print jobs.

```bash
# Create mapping via API
curl -X POST https://your-openprint-cloud.example.com/user-printer-mappings \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "user_email": "john@example.com",
    "user_name": "DOMAIN\\john",
    "client_agent_id": "<client-agent-uuid>",
    "target_printer_name": "HP LaserJet 4200",
    "server_agent_id": "<server-agent-uuid>",
    "organization_id": "<org-uuid>",
    "is_default": true
  }'
```

You can also manage mappings via the dashboard or use the API to list agents and printers:

```bash
# List agents to find agent IDs
curl https://your-openprint-cloud.example.com/agents \
  -H "Authorization: Bearer $TOKEN"

# List discovered printers for a client agent
curl https://your-openprint-cloud.example.com/agents/<client-agent-id>/printers \
  -H "Authorization: Bearer $TOKEN"
```

### Step 5: Configure Guacamole (RDP Access)

Configure your Guacamole connection to the RDP server:

```xml
<connection name="PrintServer">
    <protocol>rdp</protocol>
    <param name="hostname">192.168.x.x</param>
    <param name="port">3389</param>
    <param name="username">user</param>
    <param name="password">password</param>
    <param name="security">nla</param>
    <!-- Guacamole PDF printing is optional; OpenPrint handles native printing -->
</connection>
```

### Step 6: Test the Flow

1. Open browser, connect to RDP server via Guacamole
2. Open any Windows application (Word, Notepad, etc.)
3. Click **File > Print**
4. Select **"OpenPrint"** from the printer list
5. Click **Print**
6. The document should print on the user's local physical printer within seconds

---

## Agent Configuration Reference

### Server Mode (`agent_role: "server"`)

| Config Field | Default | Description |
|-------------|---------|-------------|
| `agent_role` | `"standard"` | Set to `"server"` for RDP session host |
| `virtual_printer_name` | `"OpenPrint"` | Name shown in Windows print dialog |
| `print_listen_port` | `9100` | TCP port for capturing print data |
| `storage_service_url` | Same as `server_url` | URL of storage-service for uploads |

### Client Mode (`agent_role: "client"`)

| Config Field | Default | Description |
|-------------|---------|-------------|
| `agent_role` | `"standard"` | Set to `"client"` for user workstation |
| `job_poll_interval_seconds` | `10` | How often to check for new jobs |

### Standard Mode (`agent_role: "standard"`)

Legacy mode that combines both behaviors. Discovers local printers and polls for jobs.
Does not create a virtual printer or capture spool data.

---

## API Endpoints

### User-Printer Mappings (registry-service :8002)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/user-printer-mappings` | Create a new mapping |
| GET | `/user-printer-mappings?user_email=...` | List mappings for a user |
| GET | `/user-printer-mappings?client_agent_id=...` | List mappings for a client agent |
| GET | `/user-printer-mappings?organization_id=...` | List mappings for an org |
| GET | `/user-printer-mappings/{id}` | Get a specific mapping |
| PUT | `/user-printer-mappings/{id}` | Update a mapping |
| DELETE | `/user-printer-mappings/{id}` | Delete a mapping |
| GET | `/user-printer-mappings/resolve?username=...` | Resolve Windows username to email |

### Job Routing (job-service :8003)

Jobs with `printer_id: "__user_default__"` are routed automatically via user-printer mappings.
When a client agent polls (`POST /agents/jobs/poll`), the server joins `user_printer_mappings`
to find jobs that should be routed to that agent.

---

## Troubleshooting

### Virtual printer not appearing on RDP server

```powershell
# Check if OpenPrint printer exists
Get-Printer -Name "OpenPrint"

# Check if the TCP port is listening
Test-NetConnection -ComputerName 127.0.0.1 -Port 9100

# Check agent service status
Get-Service OpenPrintAgent

# View agent logs
Get-Content "C:\ProgramData\OpenPrint\agent.log" -Tail 50
```

### Print jobs not reaching client

```bash
# Check if the job was created in the cloud
curl https://your-openprint-cloud.example.com/jobs?user_email=john@example.com \
  -H "Authorization: Bearer $TOKEN"

# Check if user-printer mapping exists
curl https://your-openprint-cloud.example.com/user-printer-mappings?user_email=john@example.com \
  -H "Authorization: Bearer $TOKEN"

# Check client agent status
curl https://your-openprint-cloud.example.com/agents/<client-agent-id> \
  -H "Authorization: Bearer $TOKEN"
```

### User not identified from RDP session

The server agent identifies users in this order:
1. Windows Print Spooler job metadata (`Get-PrintJob`)
2. RDP session query (`qwinsta`)
3. Active Directory email lookup (`Get-ADUser`)
4. OpenPrint username-to-email mapping (`/user-printer-mappings/resolve`)

If the user is not identified, create a mapping with the Windows username:
```bash
curl -X POST .../user-printer-mappings \
  -d '{"user_email": "john@example.com", "user_name": "DOMAIN\\john", ...}'
```

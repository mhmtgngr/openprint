# Login to get JWT token
$loginBody = @{
    email = "admin@openprint.local"
    password = "Admin123@"
} | ConvertTo-Json

$loginResp = Invoke-RestMethod -Uri "http://192.168.31.76:18001/auth/login" -Method POST -Body $loginBody -ContentType "application/json"
Write-Host "Login OK: $($loginResp.email)"
$token = $loginResp.access_token

# Register agent with server role
$regBody = @{
    name = "YMTINBRDSH5TT1"
    version = "1.0.0"
    os = "Windows Server 2022 Datacenter"
    architecture = "amd64"
    hostname = "YMTINBRDSH5TT1"
    domain = "DIYANETVAKFI"
    username = "SYSTEM"
    agent_role = "server"
    organization_id = $loginResp.user_id
    mac_address = ""
} | ConvertTo-Json

$headers = @{
    Authorization = "Bearer $token"
}

# Try registration
try {
    # Try port 8002 (registry-service) - no auth needed (SkipPaths)
    $regResp = Invoke-RestMethod -Uri "http://192.168.31.76:8002/agents/register" -Method POST -Body $regBody -ContentType "application/json"
    Write-Host "Registration OK!"
    Write-Host "Agent ID: $($regResp.agent_id)"

    # Update config.json with the agent_id
    $configPath = "C:\ProgramData\OpenPrint\agent\config.json"
    $config = Get-Content $configPath | ConvertFrom-Json
    $config.agent_id = $regResp.agent_id
    $config | ConvertTo-Json -Depth 10 | Set-Content $configPath
    Write-Host "Config updated with agent_id: $($regResp.agent_id)"
} catch {
    Write-Host "Registration failed: $($_.Exception.Message)"
    if ($_.Exception.Response) {
        $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
        Write-Host "Response: $($reader.ReadToEnd())"
    }
}

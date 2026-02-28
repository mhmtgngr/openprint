-- OpenPrint Cloud - Extend Agents Table for Windows Print Agent

-- Add new columns to agents table for Windows agent support
ALTER TABLE agents ADD COLUMN IF NOT EXISTS session_state VARCHAR(50) DEFAULT 'active';
ALTER TABLE agents ADD COLUMN IF NOT EXISTS printer_count INTEGER DEFAULT 0;
ALTER TABLE agents ADD COLUMN IF NOT EXISTS job_queue_depth INTEGER DEFAULT 0;
ALTER TABLE agents ADD COLUMN IF NOT EXISTS boot_time TIMESTAMPTZ;
ALTER TABLE agents ADD COLUMN IF NOT EXISTS ip_address VARCHAR(45);
ALTER TABLE agents ADD COLUMN IF NOT EXISTS mac_address VARCHAR(17);
ALTER TABLE agents ADD COLUMN IF NOT EXISTS certificate_thumbprint VARCHAR(64);

-- Add index for certificate thumbprint lookups
CREATE INDEX IF NOT EXISTS idx_agents_cert_thumbprint ON agents(certificate_thumbprint) WHERE certificate_thumbprint IS NOT NULL;

-- Add index for finding agents by status and heartbeat
CREATE INDEX IF NOT EXISTS idx_agents_status_heartbeat ON agents(status, last_heartbeat) WHERE status = 'online';

-- Add comments for documentation
COMMENT ON COLUMN agents.session_state IS 'Current session state: active, idle, disconnected';
COMMENT ON COLUMN agents.printer_count IS 'Number of printers discovered by this agent';
COMMENT ON COLUMN agents.job_queue_depth IS 'Number of jobs currently queued for this agent';
COMMENT ON COLUMN agents.boot_time IS 'When the agent/machine was started';
COMMENT ON COLUMN agents.ip_address IS 'Agent IP address';
COMMENT ON COLUMN agents.mac_address IS 'Agent MAC address';
COMMENT ON COLUMN agents.certificate_thumbprint IS 'SHA-256 thumbprint of agent certificate';

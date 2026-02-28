-- Rollback: OpenPrint Cloud - Extend Agents Table for Windows Print Agent

-- Remove indexes
DROP INDEX IF EXISTS idx_agents_cert_thumbprint;
DROP INDEX IF EXISTS idx_agents_status_heartbeat;

-- Remove columns (not all databases support this easily, use ALTER TABLE DROP COLUMN if supported)
-- PostgreSQL supports dropping columns:
ALTER TABLE agents DROP COLUMN IF EXISTS session_state;
ALTER TABLE agents DROP COLUMN IF EXISTS printer_count;
ALTER TABLE agents DROP COLUMN IF EXISTS job_queue_depth;
ALTER TABLE agents DROP COLUMN IF EXISTS boot_time;
ALTER TABLE agents DROP COLUMN IF EXISTS ip_address;
ALTER TABLE agents DROP COLUMN IF EXISTS mac_address;
ALTER TABLE agents DROP COLUMN IF EXISTS certificate_thumbprint;

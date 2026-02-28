-- Rollback: OpenPrint Cloud - Discovered Printers Table

DROP TRIGGER IF EXISTS update_discovered_printers_updated_at ON discovered_printers;
DROP INDEX IF EXISTS idx_discovered_printers_capabilities;
DROP INDEX IF EXISTS idx_discovered_printers_name;
DROP INDEX IF EXISTS idx_discovered_printers_last_seen;
DROP INDEX IF EXISTS idx_discovered_printers_connection_type;
DROP INDEX IF EXISTS idx_discovered_printers_status;
DROP INDEX IF EXISTS idx_discovered_printers_agent;
DROP TABLE IF EXISTS discovered_printers;

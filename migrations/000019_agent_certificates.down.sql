-- Rollback: OpenPrint Cloud - Agent Certificates Table

-- Drop functions
DROP FUNCTION IF EXISTS revoke_certificate(UUID, TEXT);
DROP FUNCTION IF EXISTS is_certificate_revoked(VARCHAR(64));
DROP FUNCTION IF EXISTS get_active_certificate(UUID);
DROP FUNCTION IF EXISTS update_certificate_thumbprint();

-- Drop trigger
DROP TRIGGER IF EXISTS update_agent_certificates_updated_at ON agent_certificates;

-- Drop revocation log table and indexes
DROP INDEX IF EXISTS idx_cert_revocation_log_timestamp;
DROP INDEX IF EXISTS idx_cert_revocation_log_agent;
DROP INDEX IF EXISTS idx_cert_revocation_log_certificate;
DROP TABLE IF EXISTS certificate_revocation_log;

-- Drop agent_certificates table and indexes
DROP INDEX IF EXISTS idx_agent_certificates_active;
DROP INDEX IF EXISTS idx_agent_certificates_validity;
DROP INDEX IF EXISTS idx_agent_certificates_is_revoked;
DROP INDEX IF EXISTS idx_agent_certificates_serial;
DROP INDEX IF EXISTS idx_agent_certificates_thumbprint;
DROP INDEX IF EXISTS idx_agent_certificates_agent;
DROP TABLE IF EXISTS agent_certificates;

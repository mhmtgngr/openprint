-- OpenPrint Cloud - Agent Certificates Table

-- Agent certificates table (stores X.509 certificates for agents)
CREATE TABLE IF NOT EXISTS agent_certificates (
    certificate_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    serial_number TEXT NOT NULL UNIQUE,
    thumbprint VARCHAR(64) NOT NULL UNIQUE,
    subject TEXT NOT NULL,
    issuer TEXT NOT NULL,
    not_valid_before TIMESTAMPTZ NOT NULL,
    not_valid_after TIMESTAMPTZ NOT NULL,
    is_revoked BOOLEAN DEFAULT false,
    revoked_at TIMESTAMPTZ,
    revocation_reason TEXT,
    certificate_data TEXT, -- PEM-encoded certificate
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_agent_certificates_agent ON agent_certificates(agent_id);
CREATE INDEX IF NOT EXISTS idx_agent_certificates_thumbprint ON agent_certificates(thumbprint);
CREATE INDEX IF NOT EXISTS idx_agent_certificates_serial ON agent_certificates(serial_number);
CREATE INDEX IF NOT EXISTS idx_agent_certificates_is_revoked ON agent_certificates(is_revoked);
CREATE INDEX IF NOT EXISTS idx_agent_certificates_validity ON agent_certificates(not_valid_after);

-- Create index for finding active (non-revoked, valid) certificates
CREATE INDEX IF NOT EXISTS idx_agent_certificates_active ON agent_certificates(agent_id)
    WHERE is_revoked = false AND not_valid_after > NOW();

-- Create trigger for updated_at
CREATE TRIGGER update_agent_certificates_updated_at BEFORE UPDATE ON agent_certificates
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Add comments for documentation
COMMENT ON TABLE agent_certificates IS 'X.509 certificates issued to agents';
COMMENT ON COLUMN agent_certificates.certificate_id IS 'Unique certificate identifier';
COMMENT ON COLUMN agent_certificates.agent_id IS 'The agent this certificate belongs to';
COMMENT ON COLUMN agent_certificates.serial_number IS 'Certificate serial number (hex)';
COMMENT ON COLUMN agent_certificates.thumbprint IS 'SHA-256 thumbprint of the certificate';
COMMENT ON COLUMN agent_certificates.subject IS 'Certificate subject distinguished name';
COMMENT ON COLUMN agent_certificates.issuer IS 'Certificate issuer distinguished name';
COMMENT ON COLUMN agent_certificates.not_valid_before IS 'Certificate validity start time';
COMMENT ON COLUMN agent_certificates.not_valid_after IS 'Certificate validity end time';
COMMENT ON COLUMN agent_certificates.is_revoked IS 'Whether the certificate has been revoked';
COMMENT ON COLUMN agent_certificates.revoked_at IS 'When the certificate was revoked';
COMMENT ON COLUMN agent_certificates.revocation_reason IS 'Reason for revocation';
COMMENT ON COLUMN agent_certificates.certificate_data IS 'PEM-encoded certificate data';

-- Create function to check if a certificate is revoked
CREATE OR REPLACE FUNCTION is_certificate_revoked(cert_thumbprint VARCHAR(64))
RETURNS BOOLEAN AS $$
DECLARE
    revoked BOOLEAN;
BEGIN
    SELECT is_revoked INTO revoked
    FROM agent_certificates
    WHERE thumbprint = cert_thumbprint;

    RETURN COALESCE(revoked, true);
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION is_certificate_revoked IS 'Checks if a certificate thumbprint is revoked';

-- Create function to revoke a certificate
CREATE OR REPLACE FUNCTION revoke_certificate(cert_id UUID, reason TEXT)
RETURNS BOOLEAN AS $$
BEGIN
    UPDATE agent_certificates
    SET is_revoked = true,
        revoked_at = NOW(),
        revocation_reason = reason,
        updated_at = NOW()
    WHERE certificate_id = cert_id;

    RETURN FOUND;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION revoke_certificate IS 'Revokes an agent certificate';

-- Create function to get active certificate for an agent
CREATE OR REPLACE FUNCTION get_active_certificate(agent_uuid UUID)
RETURNS agent_certificates AS $$
DECLARE
    cert agent_certificates;
BEGIN
    SELECT * INTO cert
    FROM agent_certificates
    WHERE agent_id = agent_uuid
        AND is_revoked = false
        AND not_valid_after > NOW()
    ORDER BY not_valid_after DESC
    LIMIT 1;

    RETURN cert;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION get_active_certificate IS 'Returns the active certificate for an agent';

-- Create trigger to auto-update thumbprint on certificate data change
CREATE OR REPLACE FUNCTION update_certificate_thumbprint()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.certificate_data IS DISTINCT FROM OLD.certificate_data AND NEW.certificate_data IS NOT NULL THEN
        -- Extract thumbprint from certificate data
        -- This would require pgcrypto or similar for SHA256 calculation
        -- For now, the thumbprint should be set by the application
        NEW.updated_at = NOW();
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Certificate revocation log table (for audit trail)
CREATE TABLE IF NOT EXISTS certificate_revocation_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    certificate_id UUID NOT NULL REFERENCES agent_certificates(id) ON DELETE CASCADE,
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    revoked_by VARCHAR(255),
    revocation_reason TEXT NOT NULL,
    revoked_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create index for revocation log
CREATE INDEX IF NOT EXISTS idx_cert_revocation_log_certificate ON certificate_revocation_log(certificate_id);
CREATE INDEX IF NOT EXISTS idx_cert_revocation_log_agent ON certificate_revocation_log(agent_id);
CREATE INDEX IF NOT EXISTS idx_cert_revocation_log_timestamp ON certificate_revocation_log(revoked_at DESC);

COMMENT ON TABLE certificate_revocation_log IS 'Audit log for certificate revocations';
COMMENT ON COLUMN certificate_revocation_log.revoked_by IS 'User or system that revoked the certificate';

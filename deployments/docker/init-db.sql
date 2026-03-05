-- OpenPrint Database Initialization
-- Auto-generated from migrations 000001-000044
-- This script runs on first PostgreSQL container startup

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ========================================
-- Migration: 000001_create_organizations.up.sql
-- ========================================
-- OpenPrint Cloud - Organizations Table
-- Enable UUID extension (must be first migration)
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Organizations table (for multi-tenant support)
CREATE TABLE IF NOT EXISTS organizations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    plan VARCHAR(50) DEFAULT 'free', -- free, pro, enterprise
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_organizations_slug ON organizations(slug);
CREATE INDEX IF NOT EXISTS idx_organizations_plan ON organizations(plan);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create trigger for updated_at
CREATE TRIGGER update_organizations_updated_at BEFORE UPDATE ON organizations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();


-- ========================================
-- Migration: 000002_create_users.up.sql
-- ========================================
-- OpenPrint Cloud - Users Table

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password VARCHAR(255), -- Hashed password, null for SSO users
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    role VARCHAR(50) DEFAULT 'user', -- user, admin, org_admin
    organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
    is_active BOOLEAN DEFAULT true,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_organization ON users(organization_id);
CREATE INDEX IF NOT EXISTS idx_users_active ON users(is_active) WHERE is_active = true;

-- Create trigger for updated_at
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();


-- ========================================
-- Migration: 000003_create_sessions.up.sql
-- ========================================
-- OpenPrint Cloud - User Sessions Table

-- User sessions (for OAuth/SAML state management)
CREATE TABLE IF NOT EXISTS user_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(50), -- oidc, saml
    provider_user_id VARCHAR(255),
    access_token TEXT,
    refresh_token TEXT,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_user_sessions_user ON user_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_sessions_provider ON user_sessions(provider, provider_user_id);
CREATE INDEX IF NOT EXISTS idx_user_sessions_expires ON user_sessions(expires_at) WHERE expires_at IS NOT NULL;


-- ========================================
-- Migration: 000004_create_agents.up.sql
-- ========================================
-- OpenPrint Cloud - Agents Table

-- Agents table (print server agents)
CREATE TABLE IF NOT EXISTS agents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    version VARCHAR(50),
    os VARCHAR(100),
    architecture VARCHAR(50),
    hostname VARCHAR(255),
    organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
    status VARCHAR(50) DEFAULT 'offline', -- online, offline
    last_heartbeat TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_agents_organization ON agents(organization_id);
CREATE INDEX IF NOT EXISTS idx_agents_status ON agents(status);
CREATE INDEX IF NOT EXISTS idx_agents_hostname ON agents(hostname);

-- Create trigger for updated_at
CREATE TRIGGER update_agents_updated_at BEFORE UPDATE ON agents
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();


-- ========================================
-- Migration: 000005_create_printers.up.sql
-- ========================================
-- OpenPrint Cloud - Printers Table

-- Printers table
CREATE TABLE IF NOT EXISTS printers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
    status VARCHAR(50) DEFAULT 'offline', -- online, offline, busy, error
    capabilities JSONB, -- Printer capabilities (color, duplex, media, etc.)
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_printers_agent ON printers(agent_id);
CREATE INDEX IF NOT EXISTS idx_printers_organization ON printers(organization_id);
CREATE INDEX IF NOT EXISTS idx_printers_status ON printers(status);

-- Create trigger for updated_at
CREATE TRIGGER update_printers_updated_at BEFORE UPDATE ON printers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();


-- ========================================
-- Migration: 000006_create_printer_permissions.up.sql
-- ========================================
-- OpenPrint Cloud - Printer Permissions Table

CREATE TABLE IF NOT EXISTS printer_permissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    printer_id UUID NOT NULL REFERENCES printers(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    permission_type VARCHAR(50) DEFAULT 'print' CHECK (permission_type IN ('print', 'manage', 'admin')),
    granted_at TIMESTAMPTZ DEFAULT NOW(),
    granted_by UUID REFERENCES users(id),
    UNIQUE(printer_id, user_id)
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_permissions_printer ON printer_permissions(printer_id);
CREATE INDEX IF NOT EXISTS idx_permissions_user ON printer_permissions(user_id);
CREATE INDEX IF NOT EXISTS idx_permissions_granted_by ON printer_permissions(granted_by) WHERE granted_by IS NOT NULL;


-- ========================================
-- Migration: 000007_create_print_jobs.up.sql
-- ========================================
-- OpenPrint Cloud - Print Jobs Table

-- Documents table (needed as foreign key for print_jobs)
CREATE TABLE IF NOT EXISTS documents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(500) NOT NULL,
    content_type VARCHAR(100),
    size BIGINT,
    checksum VARCHAR(64),
    storage_path VARCHAR(1000), -- Path in S3/local storage
    user_email VARCHAR(255) NOT NULL,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes for documents
CREATE INDEX IF NOT EXISTS idx_documents_user ON documents(user_email);
CREATE INDEX IF NOT EXISTS idx_documents_created ON documents(created_at);
CREATE INDEX IF NOT EXISTS idx_documents_expires ON documents(expires_at) WHERE expires_at IS NOT NULL;

-- Print jobs table
CREATE TABLE IF NOT EXISTS print_jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE RESTRICT,
    printer_id UUID NOT NULL REFERENCES printers(id) ON DELETE RESTRICT,
    user_name VARCHAR(255),
    user_email VARCHAR(255) NOT NULL,
    title VARCHAR(500),
    copies INTEGER DEFAULT 1,
    color_mode VARCHAR(20) DEFAULT 'monochrome',
    duplex BOOLEAN DEFAULT false,
    media_type VARCHAR(50) DEFAULT 'a4',
    quality VARCHAR(50) DEFAULT 'normal',
    pages INTEGER,
    status VARCHAR(50) DEFAULT 'queued', -- queued, processing, pending_agent, completed, failed, cancelled, paused
    priority INTEGER DEFAULT 5, -- 1-10, higher is more important
    retries INTEGER DEFAULT 0,
    options JSONB, -- Additional print options
    agent_id UUID REFERENCES agents(id) ON DELETE SET NULL,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_print_jobs_document ON print_jobs(document_id);
CREATE INDEX IF NOT EXISTS idx_print_jobs_printer ON print_jobs(printer_id);
CREATE INDEX IF NOT EXISTS idx_print_jobs_user ON print_jobs(user_email);
CREATE INDEX IF NOT EXISTS idx_print_jobs_status ON print_jobs(status);
CREATE INDEX IF NOT EXISTS idx_print_jobs_status_priority ON print_jobs(status, priority DESC, created_at ASC);
CREATE INDEX IF NOT EXISTS idx_print_jobs_created ON print_jobs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_print_jobs_agent ON print_jobs(agent_id);

-- Create trigger for updated_at
CREATE TRIGGER update_print_jobs_updated_at BEFORE UPDATE ON print_jobs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();


-- ========================================
-- Migration: 000008_create_job_history.up.sql
-- ========================================
-- OpenPrint Cloud - Job History Table

-- Job history table
CREATE TABLE IF NOT EXISTS job_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_id UUID NOT NULL REFERENCES print_jobs(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL,
    message TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_job_history_job ON job_history(job_id);
CREATE INDEX IF NOT EXISTS idx_job_history_created ON job_history(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_job_history_status ON job_history(status);


-- ========================================
-- Migration: 000009_create_audit_logs.up.sql
-- ========================================
-- OpenPrint Cloud - Audit Log Table

-- Audit log table
CREATE TABLE IF NOT EXISTS audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(100),
    resource_id UUID,
    ip_address INET,
    user_agent TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_audit_log_user ON audit_log(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_organization ON audit_log(organization_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_resource ON audit_log(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_created ON audit_log(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_log_action ON audit_log(action);


-- ========================================
-- Migration: 000010_create_usage_stats.up.sql
-- ========================================
-- OpenPrint Cloud - Usage Stats Table

CREATE TABLE IF NOT EXISTS usage_stats (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    printer_id UUID REFERENCES printers(id) ON DELETE SET NULL,
    stat_date DATE NOT NULL,
    pages_printed INTEGER DEFAULT 0,
    color_pages INTEGER DEFAULT 0,
    jobs_count INTEGER DEFAULT 0,
    jobs_completed INTEGER DEFAULT 0,
    jobs_failed INTEGER DEFAULT 0,
    total_bytes BIGINT DEFAULT 0,
    estimated_cost DECIMAL(10,2) DEFAULT 0,
    co2_grams DECIMAL(10,2) DEFAULT 0,
    trees_saved DECIMAL(10,4) DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(org_id, user_id, printer_id, stat_date)
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_usage_stats_date ON usage_stats(stat_date);
CREATE INDEX IF NOT EXISTS idx_usage_stats_org ON usage_stats(org_id, stat_date);
CREATE INDEX IF NOT EXISTS idx_usage_stats_user ON usage_stats(user_id, stat_date) WHERE user_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_usage_stats_printer ON usage_stats(printer_id, stat_date) WHERE printer_id IS NOT NULL;


-- ========================================
-- Migration: 000011_create_api_keys.up.sql
-- ========================================
-- OpenPrint Cloud - API Keys Table

CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    key_hash VARCHAR(255) NOT NULL,
    scopes TEXT[] DEFAULT '{}',
    is_active BOOLEAN DEFAULT true,
    expires_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_api_keys_user ON api_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_org ON api_keys(org_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_active ON api_keys(is_active) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_api_keys_hash ON api_keys(key_hash);


-- ========================================
-- Migration: 000012_create_webhooks.up.sql
-- ========================================
-- OpenPrint Cloud - Webhooks Table

CREATE TABLE IF NOT EXISTS webhooks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    url VARCHAR(2048) NOT NULL,
    secret VARCHAR(255),
    events TEXT[] NOT NULL DEFAULT '{}',
    is_active BOOLEAN DEFAULT true,
    last_triggered_at TIMESTAMPTZ,
    failure_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_webhooks_org ON webhooks(org_id);
CREATE INDEX IF NOT EXISTS idx_webhooks_active ON webhooks(org_id, is_active);

-- Create trigger for updated_at
CREATE TRIGGER update_webhooks_updated_at BEFORE UPDATE ON webhooks
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();


-- ========================================
-- Migration: 000013_create_invitations.up.sql
-- ========================================
-- OpenPrint Cloud - Invitations Table

CREATE TABLE IF NOT EXISTS invitations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    role VARCHAR(50) DEFAULT 'user' CHECK (role IN ('user', 'admin')),
    invited_by UUID REFERENCES users(id),
    accepted_by UUID REFERENCES users(id),
    accepted_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_invitations_org ON invitations(org_id);
CREATE INDEX IF NOT EXISTS idx_invitations_email ON invitations(email, accepted_at);
CREATE INDEX IF NOT EXISTS idx_invitations_expires ON invitations(expires_at) WHERE accepted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_invitations_invited_by ON invitations(invited_by) WHERE invited_by IS NOT NULL;


-- ========================================
-- Migration: 000014_create_devices.up.sql
-- ========================================
-- OpenPrint Cloud - Devices Table (User mobile/web devices for push notifications)

CREATE TABLE IF NOT EXISTS devices (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_type VARCHAR(50) NOT NULL CHECK (device_type IN ('ios', 'android', 'web')),
    device_id VARCHAR(255) NOT NULL,
    push_token TEXT,
    push_provider VARCHAR(50) DEFAULT 'fcm' CHECK (push_provider IN ('fcm', 'apns', 'none')),
    is_active BOOLEAN DEFAULT true,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, device_id)
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_devices_user ON devices(user_id);
CREATE INDEX IF NOT EXISTS idx_devices_push ON devices(push_token) WHERE push_token IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_devices_active ON devices(user_id, is_active) WHERE is_active = true;


-- ========================================
-- Migration: 000015_extend_agents.up.sql
-- ========================================
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


-- ========================================
-- Migration: 000016_discovered_printers.up.sql
-- ========================================
-- OpenPrint Cloud - Discovered Printers Table

-- Discovered printers table (printers found by agents)
CREATE TABLE IF NOT EXISTS discovered_printers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    driver VARCHAR(255),
    driver_version VARCHAR(100),
    port VARCHAR(100),
    connection_type VARCHAR(20) NOT NULL DEFAULT 'local',
    status VARCHAR(50) NOT NULL DEFAULT 'idle',
    is_default BOOLEAN DEFAULT false,
    is_shared BOOLEAN DEFAULT false,
    share_name VARCHAR(255),
    location VARCHAR(255),
    capabilities JSONB,
    last_seen TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(agent_id, name)
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_discovered_printers_agent ON discovered_printers(agent_id);
CREATE INDEX IF NOT EXISTS idx_discovered_printers_status ON discovered_printers(status);
CREATE INDEX IF NOT EXISTS idx_discovered_printers_connection_type ON discovered_printers(connection_type);
CREATE INDEX IF NOT EXISTS idx_discovered_printers_last_seen ON discovered_printers(last_seen);
CREATE INDEX IF NOT EXISTS idx_discovered_printers_name ON discovered_printers(name);

-- Create index for GIN queries on capabilities JSONB
CREATE INDEX IF NOT EXISTS idx_discovered_printers_capabilities ON discovered_printers USING GIN (capabilities);

-- Create trigger for updated_at
CREATE TRIGGER update_discovered_printers_updated_at BEFORE UPDATE ON discovered_printers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create function to update updated_at column if it doesn't exist
CREATE OR REPLACE FUNCTION update_updated_at_column() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Add comments for documentation
COMMENT ON TABLE discovered_printers IS 'Printers discovered by Windows agents';
COMMENT ON COLUMN discovered_printers.agent_id IS 'The agent that discovered this printer';
COMMENT ON COLUMN discovered_printers.name IS 'Printer name as reported by the OS';
COMMENT ON COLUMN discovered_printers.connection_type IS 'Connection type: local, network, shared, wsd, lpd';
COMMENT ON COLUMN discovered_printers.status IS 'Printer status: idle, printing, busy, offline, error, out_of_paper, low_toner, door_open';
COMMENT ON COLUMN discovered_printers.capabilities IS 'Printer capabilities as JSONB';
COMMENT ON COLUMN discovered_printers.last_seen IS 'When this printer was last seen by the agent';


-- ========================================
-- Migration: 000017_job_assignments.up.sql
-- ========================================
-- OpenPrint Cloud - Job Assignments Table

-- Job assignments table (tracks assignment of jobs to agents)
CREATE TABLE IF NOT EXISTS job_assignments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_id UUID NOT NULL REFERENCES print_jobs(id) ON DELETE CASCADE,
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    status VARCHAR(50) NOT NULL DEFAULT 'assigned',
    retry_count INTEGER DEFAULT 0,
    last_heartbeat TIMESTAMPTZ DEFAULT NOW(),
    error TEXT,
    document_etag VARCHAR(64),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_job_assignments_job ON job_assignments(job_id);
CREATE INDEX IF NOT EXISTS idx_job_assignments_agent ON job_assignments(agent_id);
CREATE INDEX IF NOT EXISTS idx_job_assignments_status ON job_assignments(status);
CREATE INDEX IF NOT EXISTS idx_job_assignments_agent_status ON job_assignments(agent_id, status);
CREATE INDEX IF NOT EXISTS idx_job_assignments_assigned_at ON job_assignments(assigned_at DESC);
CREATE INDEX IF NOT EXISTS idx_job_assignments_last_heartbeat ON job_assignments(last_heartbeat);

-- Create unique index to prevent duplicate assignments
CREATE UNIQUE INDEX IF NOT EXISTS idx_job_assignments_job_agent_unique ON job_assignments(job_id, agent_id)
    WHERE status IN ('assigned', 'in_progress');

-- Create trigger for updated_at
CREATE TRIGGER update_job_assignments_updated_at BEFORE UPDATE ON job_assignments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Add comments for documentation
COMMENT ON TABLE job_assignments IS 'Tracks assignment of print jobs to agents';
COMMENT ON COLUMN job_assignments.job_id IS 'The print job being assigned';
COMMENT ON COLUMN job_assignments.agent_id IS 'The agent assigned to this job';
COMMENT ON COLUMN job_assignments.status IS 'Assignment status: assigned, in_progress, completed, failed, cancelled';
COMMENT ON COLUMN job_assignments.retry_count IS 'Number of times this assignment has been retried';
COMMENT ON COLUMN job_assignments.last_heartbeat IS 'Last heartbeat from agent for this assignment';
COMMENT ON COLUMN job_assignments.document_etag IS 'ETag for resume support';


-- ========================================
-- Migration: 000018_agent_events.up.sql
-- ========================================
-- OpenPrint Cloud - Agent Events Table

-- Agent events table (logs events from agents)
CREATE TABLE IF NOT EXISTS agent_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    event_type VARCHAR(100) NOT NULL,
    severity VARCHAR(20) NOT NULL DEFAULT 'info',
    message TEXT NOT NULL,
    details JSONB,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_agent_events_agent ON agent_events(agent_id);
CREATE INDEX IF NOT EXISTS idx_agent_events_type ON agent_events(event_type);
CREATE INDEX IF NOT EXISTS idx_agent_events_severity ON agent_events(severity);
CREATE INDEX IF NOT EXISTS idx_agent_events_timestamp ON agent_events(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_agent_events_agent_timestamp ON agent_events(agent_id, timestamp DESC);

-- Create index for GIN queries on details JSONB
CREATE INDEX IF NOT EXISTS idx_agent_events_details ON agent_events USING GIN (details);

-- Create partitioning function (optional, for large deployments)
-- This would allow partitioning by timestamp

-- Add comments for documentation
COMMENT ON TABLE agent_events IS 'Event log for agent activities';
COMMENT ON COLUMN agent_events.agent_id IS 'The agent that generated this event';
COMMENT ON COLUMN agent_events.event_type IS 'Event type: printer_added, printer_removed, job_started, job_completed, error, etc.';
COMMENT ON COLUMN agent_events.severity IS 'Severity level: debug, info, warning, error, critical';
COMMENT ON COLUMN agent_events.message IS 'Human-readable event message';
COMMENT ON COLUMN agent_events.details IS 'Additional event details as JSONB';
COMMENT ON COLUMN agent_events.timestamp IS 'When the event occurred';

-- Create function to clean up old events (retention policy)
CREATE OR REPLACE FUNCTION cleanup_old_events(retention_days INTEGER DEFAULT 90)
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM agent_events
    WHERE timestamp < NOW() - (retention_days || ' days')::INTERVAL;

    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Add comment for cleanup function
COMMENT ON FUNCTION cleanup_old_events IS 'Deletes events older than the specified retention period';


-- ========================================
-- Migration: 000019_agent_certificates.up.sql
-- ========================================
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
-- Note: Time-based predicate removed because NOW() is STABLE, not IMMUTABLE
-- Applications should filter by not_valid_after in queries
CREATE INDEX IF NOT EXISTS idx_agent_certificates_active ON agent_certificates(agent_id)
    WHERE is_revoked = false;

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
    certificate_id UUID NOT NULL REFERENCES agent_certificates(certificate_id) ON DELETE CASCADE,
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


-- ========================================
-- Migration: 000020_create_enrollment_tokens.up.sql
-- ========================================
-- Create enrollment_tokens table for agent enrollment token management
CREATE TABLE IF NOT EXISTS enrollment_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token VARCHAR(255) UNIQUE NOT NULL,
    organization_id VARCHAR(255),
    name VARCHAR(255) NOT NULL,
    created_by VARCHAR(255) NOT NULL,
    max_uses INTEGER NOT NULL DEFAULT 0, -- 0 means unlimited uses
    use_count INTEGER NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    revoked_by VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create index on token for fast lookup
CREATE INDEX idx_enrollment_tokens_token ON enrollment_tokens(token);

-- Create index on organization_id for filtering
CREATE INDEX idx_enrollment_tokens_org_id ON enrollment_tokens(organization_id);

-- Create index on expires_at for cleanup
CREATE INDEX idx_enrollment_tokens_expires_at ON enrollment_tokens(expires_at);

-- Create index on revoked_at for filtering
CREATE INDEX idx_enrollment_tokens_revoked_at ON enrollment_tokens(revoked_at);

-- Add comment
COMMENT ON TABLE enrollment_tokens IS 'Stores enrollment tokens for secure agent registration';
COMMENT ON COLUMN enrollment_tokens.max_uses IS 'Maximum number of times this token can be used (0 = unlimited)';
COMMENT ON COLUMN enrollment_tokens.expires_at IS 'Optional expiration time for the token';
COMMENT ON COLUMN enrollment_tokens.revoked_at IS 'Set when token is revoked before expiration';


-- ========================================
-- Migration: 000021_test_setup_functions.up.sql
-- ========================================
-- OpenPrint Cloud - Test Setup Helper Functions
-- This migration provides helper functions for test cleanup and data management

-- Function to truncate all tables in the correct order (respecting foreign key dependencies)
CREATE OR REPLACE FUNCTION truncate_all_tables() RETURNS void AS $$
DECLARE
    stmt TEXT;
    tbl_name TEXT;
    tables TEXT[] := ARRAY[
        'job_assignments',
        'job_history',
        'print_jobs',
        'documents',
        'user_sessions',
        'audit_log',
        'printers',
        'agents',
        'users',
        'organizations',
        'api_keys',
        'webhooks',
        'invitations',
        'devices',
        'discovered_printers',
        'agent_events',
        'agent_certificates',
        'enrollment_tokens',
        'usage_stats'
    ];
BEGIN
    -- Disable triggers for faster truncation
    SET session_replication_role = 'replica';

    -- Truncate each table with CASCADE
    FOREACH tbl_name IN ARRAY tables
    LOOP
        BEGIN
            EXECUTE format('TRUNCATE TABLE %I CASCADE', tbl_name);
        EXCEPTION WHEN undefined_table THEN
            -- Table doesn't exist, skip it
            CONTINUE;
        END;
    END LOOP;

    -- Re-enable triggers
    SET session_replication_role = 'origin';
END;
$$ LANGUAGE plpgsql;

-- Function to get all test data for a specific user email
CREATE OR REPLACE FUNCTION get_user_test_data(p_user_email VARCHAR) RETURNS TABLE(
    job_count BIGINT,
    document_count BIGINT,
    assignment_count BIGINT,
    history_count BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        (SELECT COUNT(*) FROM print_jobs WHERE user_email = p_user_email),
        (SELECT COUNT(*) FROM documents WHERE user_email = p_user_email),
        (SELECT COUNT(*) FROM job_assignments WHERE job_id IN (
            SELECT id FROM print_jobs WHERE user_email = p_user_email
        )),
        (SELECT COUNT(*) FROM job_history WHERE job_id IN (
            SELECT id FROM print_jobs WHERE user_email = p_user_email
        ));
END;
$$ LANGUAGE plpgsql;

-- Function to safely drop test data (for cleanup between tests)
CREATE OR REPLACE FUNCTION cleanup_test_data(p_user_email VARCHAR DEFAULT NULL) RETURNS void AS $$
BEGIN
    IF p_user_email IS NOT NULL THEN
        -- Delete job history first
        DELETE FROM job_history
        WHERE job_id IN (SELECT id FROM print_jobs WHERE user_email = p_user_email);

        -- Delete job assignments
        DELETE FROM job_assignments
        WHERE job_id IN (SELECT id FROM print_jobs WHERE user_email = p_user_email);

        -- Delete print jobs
        DELETE FROM print_jobs WHERE user_email = p_user_email;

        -- Delete documents
        DELETE FROM documents WHERE user_email = p_user_email;
    ELSE
        -- Truncate all tables
        PERFORM truncate_all_tables();
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Function to create a test organization
CREATE OR REPLACE FUNCTION create_test_organization(p_name VARCHAR DEFAULT 'Test Organization')
RETURNS UUID AS $$
DECLARE
    v_org_id UUID;
    v_slug VARCHAR;
BEGIN
    v_org_id := uuid_generate_v4();
    v_slug := 'test-' || substr(v_org_id::text, 1, 8);

    INSERT INTO organizations (id, name, slug, plan)
    VALUES (v_org_id, p_name, v_slug, 'free')
    RETURNING organizations.id INTO v_org_id;

    RETURN v_org_id;
END;
$$ LANGUAGE plpgsql;

-- Function to create a test user
CREATE OR REPLACE FUNCTION create_test_user(p_org_id UUID, p_email VARCHAR DEFAULT NULL)
RETURNS UUID AS $$
DECLARE
    v_user_id UUID;
    v_user_email VARCHAR;
BEGIN
    v_user_id := uuid_generate_v4();

    IF p_email IS NULL THEN
        v_user_email := 'test-' || substr(v_user_id::text, 1, 8) || '@example.com';
    ELSE
        v_user_email := p_email;
    END IF;

    INSERT INTO users (id, email, password, first_name, last_name, organization_id, is_active)
    VALUES (
        v_user_id,
        v_user_email,
        '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', -- bcrypt hash for "password"
        'Test',
        'User',
        p_org_id,
        true
    )
    RETURNING users.id INTO v_user_id;

    RETURN v_user_id;
END;
$$ LANGUAGE plpgsql;

-- Function to create a test agent
CREATE OR REPLACE FUNCTION create_test_agent(p_org_id UUID, p_name VARCHAR DEFAULT 'test-agent')
RETURNS UUID AS $$
DECLARE
    v_agent_id UUID;
BEGIN
    v_agent_id := uuid_generate_v4();

    INSERT INTO agents (id, name, version, os, architecture, hostname, organization_id, status)
    VALUES (
        v_agent_id,
        p_name,
        '1.0.0',
        'linux',
        'x86_64',
        'test-host',
        p_org_id,
        'online'
    )
    RETURNING agents.id INTO v_agent_id;

    RETURN v_agent_id;
END;
$$ LANGUAGE plpgsql;

-- Function to create a test printer
CREATE OR REPLACE FUNCTION create_test_printer(p_agent_id UUID, p_name VARCHAR DEFAULT 'Test Printer')
RETURNS UUID AS $$
DECLARE
    v_printer_id UUID;
BEGIN
    v_printer_id := uuid_generate_v4();

    INSERT INTO printers (id, name, agent_id, status, capabilities)
    VALUES (
        v_printer_id,
        p_name,
        p_agent_id,
        'online',
        '{"color": true, "duplex": true, "media": ["a4", "letter"]}'::jsonb
    )
    RETURNING printers.id INTO v_printer_id;

    RETURN v_printer_id;
END;
$$ LANGUAGE plpgsql;

-- Function to create a test document
CREATE OR REPLACE FUNCTION create_test_document(p_user_email VARCHAR DEFAULT NULL)
RETURNS UUID AS $$
DECLARE
    v_document_id UUID;
    v_user_email VARCHAR;
BEGIN
    v_document_id := uuid_generate_v4();

    IF p_user_email IS NULL THEN
        v_user_email := 'test@example.com';
    ELSE
        v_user_email := p_user_email;
    END IF;

    INSERT INTO documents (id, name, content_type, size, checksum, storage_path, user_email)
    VALUES (
        v_document_id,
        'test-document.pdf',
        'application/pdf',
        1024,
        'test-checksum-' || substr(v_document_id::text, 1, 8),
        '/test/path-' || substr(v_document_id::text, 1, 8) || '.pdf',
        v_user_email
    )
    RETURNING documents.id INTO v_document_id;

    RETURN v_document_id;
END;
$$ LANGUAGE plpgsql;

-- Grant execute permissions on these functions (useful for test users)
GRANT EXECUTE ON FUNCTION truncate_all_tables() TO PUBLIC;
GRANT EXECUTE ON FUNCTION get_user_test_data(VARCHAR) TO PUBLIC;
GRANT EXECUTE ON FUNCTION cleanup_test_data(VARCHAR) TO PUBLIC;
GRANT EXECUTE ON FUNCTION create_test_organization(VARCHAR) TO PUBLIC;
GRANT EXECUTE ON FUNCTION create_test_user(UUID, VARCHAR) TO PUBLIC;
GRANT EXECUTE ON FUNCTION create_test_agent(UUID, VARCHAR) TO PUBLIC;
GRANT EXECUTE ON FUNCTION create_test_printer(UUID, VARCHAR) TO PUBLIC;
GRANT EXECUTE ON FUNCTION create_test_document(VARCHAR) TO PUBLIC;


-- ========================================
-- Migration: 000022_create_print_quotas.up.sql
-- ========================================
-- Print Quotas Table
-- Tracks print quotas and usage for users and organizations

CREATE TABLE IF NOT EXISTS print_quotas (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    entity_id UUID NOT NULL, -- user_id or organization_id
    entity_type VARCHAR(20) NOT NULL CHECK (entity_type IN ('user', 'organization')),
    quota_type VARCHAR(50) NOT NULL, -- 'pages', 'jobs', 'color_pages', 'duplex_pages'
    period VARCHAR(20) NOT NULL CHECK (period IN ('daily', 'weekly', 'monthly', 'quarterly', 'yearly')),
    "limit" INTEGER NOT NULL DEFAULT 0, -- 0 means unlimited
    used INTEGER NOT NULL DEFAULT 0,
    reset_date TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(entity_id, entity_type, quota_type, period)
);

CREATE INDEX idx_print_quotas_entity ON print_quotas(entity_id, entity_type);
CREATE INDEX idx_print_quotas_type ON print_quotas(quota_type);
CREATE INDEX idx_print_quotas_period ON print_quotas(period);

-- Print Cost Configuration
-- Tracks cost per page for different print types

CREATE TABLE IF NOT EXISTS print_costs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    printer_id UUID REFERENCES printers(id) ON DELETE SET NULL,
    cost_type VARCHAR(50) NOT NULL, -- 'monochrome_a4', 'color_a4', 'duplex_a4', etc.
    cost_per_page DECIMAL(10, 4) NOT NULL DEFAULT 0.0000,
    currency VARCHAR(3) DEFAULT 'USD',
    effective_from TIMESTAMPTZ DEFAULT NOW(),
    effective_to TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_print_costs_org ON print_costs(organization_id);
CREATE INDEX idx_print_costs_printer ON print_costs(printer_id);
CREATE INDEX idx_print_costs_type ON print_costs(cost_type);
CREATE INDEX idx_print_costs_dates ON print_costs(effective_from, effective_to);

-- Print Job Costs
-- Tracks calculated costs for each print job

CREATE TABLE IF NOT EXISTS print_job_costs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_id UUID NOT NULL UNIQUE REFERENCES print_jobs(id) ON DELETE CASCADE,
    page_count INTEGER NOT NULL DEFAULT 0,
    color_pages INTEGER NOT NULL DEFAULT 0,
    duplex_pages INTEGER NOT NULL DEFAULT 0,
    cost DECIMAL(10, 4) NOT NULL DEFAULT 0.0000,
    currency VARCHAR(3) DEFAULT 'USD',
    calculated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_print_job_costs_job ON print_job_costs(job_id);

-- Triggers for updated_at
CREATE TRIGGER update_print_quotas_updated_at BEFORE UPDATE ON print_quotas
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_print_costs_updated_at BEFORE UPDATE ON print_costs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Functions for quota management

CREATE OR REPLACE FUNCTION check_quota(
    p_entity_id UUID,
    p_entity_type VARCHAR,
    p_quota_type VARCHAR,
    p_increment INTEGER DEFAULT 1
) RETURNS BOOLEAN AS $$
DECLARE
    v_limit INTEGER;
    v_used INTEGER;
    v_reset_date TIMESTAMPTZ;
    v_current_date TIMESTAMPTZ := NOW();
BEGIN
    -- Get quota info
    SELECT "limit", used, reset_date INTO v_limit, v_used, v_reset_date
    FROM print_quotas
    WHERE entity_id = p_entity_id
      AND entity_type = p_entity_type
      AND quota_type = p_quota_type
    FOR UPDATE;

    -- If no quota found, allow (unlimited)
    IF NOT FOUND THEN
        RETURN TRUE;
    END IF;

    -- Check if quota needs reset
    IF v_reset_date IS NOT NULL AND v_current_date >= v_reset_date THEN
        UPDATE print_quotas
        SET used = 0,
            reset_date = calculate_reset_date(period)
        WHERE entity_id = p_entity_id
          AND entity_type = p_entity_type
          AND quota_type = p_quota_type;
        v_used := 0;
    END IF;

    -- Check if limit is 0 (unlimited) or has capacity
    IF v_limit = 0 OR (v_used + p_increment) <= v_limit THEN
        -- Update usage
        UPDATE print_quotas
        SET used = used + p_increment
        WHERE entity_id = p_entity_id
          AND entity_type = p_entity_type
          AND quota_type = p_quota_type;
        RETURN TRUE;
    END IF;

    RETURN FALSE;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION calculate_reset_date(p_period VARCHAR) RETURNS TIMESTAMPTZ AS $$
DECLARE
    v_reset_date TIMESTAMPTZ;
BEGIN
    CASE p_period
        WHEN 'daily' THEN
            v_reset_date := date_trunc('day', NOW() + INTERVAL '1 day');
        WHEN 'weekly' THEN
            v_reset_date := date_trunc('week', NOW() + INTERVAL '1 week');
        WHEN 'monthly' THEN
            v_reset_date := date_trunc('month', NOW() + INTERVAL '1 month');
        WHEN 'quarterly' THEN
            v_reset_date := date_trunc('quarter', NOW() + INTERVAL '3 months');
        WHEN 'yearly' THEN
            v_reset_date := date_trunc('year', NOW() + INTERVAL '1 year');
        ELSE
            v_reset_date := NOW() + INTERVAL '1 day';
    END CASE;
    RETURN v_reset_date;
END;
$$ LANGUAGE plpgsql;

-- Function to calculate print job cost
CREATE OR REPLACE FUNCTION calculate_print_job_cost(
    p_job_id UUID,
    p_organization_id UUID,
    p_printer_id UUID,
    p_page_count INTEGER,
    p_color_pages INTEGER,
    p_duplex_pages INTEGER
) RETURNS DECIMAL AS $$
DECLARE
    v_cost DECIMAL(10, 4) := 0;
    v_monochrome_cost DECIMAL(10, 4);
    v_color_cost DECIMAL(10, 4);
    v_duplex_savings DECIMAL(10, 4) := 0.1; -- 10% savings for duplex
BEGIN
    -- Get monochrome cost
    SELECT cost_per_page INTO v_monochrome_cost
    FROM print_costs
    WHERE (organization_id = p_organization_id OR organization_id IS NULL)
      AND (printer_id = p_printer_id OR printer_id IS NULL)
      AND cost_type = 'monochrome_a4'
      AND effective_from <= NOW()
      AND (effective_to IS NULL OR effective_to > NOW())
    ORDER BY organization_id DESC, printer_id DESC
    LIMIT 1;

    -- Get color cost
    SELECT cost_per_page INTO v_color_cost
    FROM print_costs
    WHERE (organization_id = p_organization_id OR organization_id IS NULL)
      AND (printer_id = p_printer_id OR printer_id IS NULL)
      AND cost_type = 'color_a4'
      AND effective_from <= NOW()
      AND (effective_to IS NULL OR effective_to > NOW())
    ORDER BY organization_id DESC, printer_id DESC
    LIMIT 1;

    -- Calculate base cost
    v_cost := COALESCE(v_monochrome_cost, 0) * (p_page_count - p_color_pages);
    v_cost := v_cost + COALESCE(v_color_cost, 0) * p_color_pages;

    -- Apply duplex savings
    IF p_duplex_pages > 0 THEN
        v_cost := v_cost * (1 - v_duplex_savings);
    END IF;

    -- Store cost
    INSERT INTO print_job_costs (job_id, page_count, color_pages, duplex_pages, cost)
    VALUES (p_job_id, p_page_count, p_color_pages, p_duplex_pages, v_cost)
    ON CONFLICT (job_id) DO UPDATE
    SET page_count = EXCLUDED.page_count,
        color_pages = EXCLUDED.color_pages,
        duplex_pages = EXCLUDED.duplex_pages,
        cost = EXCLUDED.cost;

    RETURN v_cost;
END;
$$ LANGUAGE plpgsql;


-- ========================================
-- Migration: 000023_create_print_policies.up.sql
-- ========================================
-- Print Policies Table
-- Defines print policies for organizations

CREATE TABLE IF NOT EXISTS print_policies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT true,
    priority INTEGER DEFAULT 0, -- Higher priority policies are evaluated first
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_print_policies_org ON print_policies(organization_id);
CREATE INDEX idx_print_policies_active ON print_policies(is_active);
CREATE INDEX idx_print_policies_priority ON print_policies(priority DESC);

-- Print Policy Rules
-- Individual rules within a policy

CREATE TABLE IF NOT EXISTS print_policy_rules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    policy_id UUID NOT NULL REFERENCES print_policies(id) ON DELETE CASCADE,
    rule_type VARCHAR(50) NOT NULL, -- 'restrict_color', 'max_copies', 'allow_duplex', 'require_pin', 'time_restrictions', etc.
    rule_operator VARCHAR(20) NOT NULL, -- 'equals', 'not_equals', 'greater_than', 'less_than', 'contains', 'between'
    rule_value JSONB NOT NULL,
    rule_action VARCHAR(50) NOT NULL, -- 'allow', 'deny', 'warn', 'require_approval'
    action_value JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_print_policy_rules_policy ON print_policy_rules(policy_id);
CREATE INDEX idx_print_policy_rules_type ON print_policy_rules(rule_type);

-- Print Policy Assignments
-- Assigns policies to users, groups, or printers

CREATE TABLE IF NOT EXISTS print_policy_assignments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    policy_id UUID NOT NULL REFERENCES print_policies(id) ON DELETE CASCADE,
    entity_id UUID NOT NULL,
    entity_type VARCHAR(20) NOT NULL CHECK (entity_type IN ('user', 'group', 'printer', 'printer_group')),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(policy_id, entity_id, entity_type)
);

CREATE INDEX idx_print_policy_assignments_policy ON print_policy_assignments(policy_id);
CREATE INDEX idx_print_policy_assignments_entity ON print_policy_assignments(entity_id, entity_type);

-- Print Job Policy Evaluations
-- Logs policy evaluations for audit purposes

CREATE TABLE IF NOT EXISTS print_policy_evaluations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_id UUID REFERENCES print_jobs(id) ON DELETE CASCADE,
    policy_id UUID REFERENCES print_policies(id),
    policy_name VARCHAR(255),
    result VARCHAR(20) NOT NULL, -- 'allowed', 'denied', 'warned', 'approval_required'
    evaluated_at TIMESTAMPTZ DEFAULT NOW(),
    evaluated_by VARCHAR(100), -- 'system', 'user_id'
    details JSONB
);

CREATE INDEX idx_print_policy_evaluations_job ON print_policy_evaluations(job_id);
CREATE INDEX idx_print_policy_evaluations_result ON print_policy_evaluations(result);

-- Triggers
CREATE TRIGGER update_print_policies_updated_at BEFORE UPDATE ON print_policies
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_print_policy_rules_updated_at BEFORE UPDATE ON print_policy_rules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to evaluate print policies
CREATE OR REPLACE FUNCTION evaluate_print_policies(
    p_job_id UUID,
    p_user_id UUID,
    p_organization_id UUID,
    p_printer_id UUID,
    p_document_attributes JSONB
) RETURNS TABLE(policy_id UUID, result VARCHAR, action_value JSONB) AS $$
DECLARE
    v_policy RECORD;
    v_rule RECORD;
    v_rule_result BOOLEAN;
    v_final_result VARCHAR := 'allow';
BEGIN
    -- Get all applicable policies for the organization, user, and printer
    FOR v_policy IN
        SELECT DISTINCT p.id, p.name, p.priority
        FROM print_policies p
        INNER JOIN print_policy_assignments a ON a.policy_id = p.id
        WHERE p.organization_id = p_organization_id
          AND p.is_active = true
          AND (a.entity_type = 'organization' OR a.entity_id = p_user_id OR a.entity_id = p_printer_id)
        ORDER BY p.priority DESC
    LOOP
        -- Evaluate each rule in the policy
        v_final_result := 'allow';
        FOR v_rule IN
            SELECT rule_type, rule_operator, rule_value, rule_action, action_value
            FROM print_policy_rules
            WHERE policy_id = v_policy.id
        LOOP
            -- Evaluate rule (simplified - in production would parse JSONB and compare)
            v_rule_result := true; -- Placeholder

            IF NOT v_rule_result THEN
                IF v_rule.rule_action = 'deny' THEN
                    v_final_result := 'denied';
                    -- Log evaluation
                    INSERT INTO print_policy_evaluations (job_id, policy_id, policy_name, result, details)
                    VALUES (p_job_id, v_policy.id, v_policy.name, 'denied', v_rule.rule_value);
                    RETURN NEXT;
                    RETURN;
                ELSIF v_rule.rule_action = 'warn' THEN
                    v_final_result := 'warned';
                ELSIF v_rule.rule_action = 'require_approval' THEN
                    v_final_result := 'approval_required';
                END IF;
            END IF;
        END LOOP;

        -- Log successful evaluation
        INSERT INTO print_policy_evaluations (job_id, policy_id, policy_name, result)
        VALUES (p_job_id, v_policy.id, v_policy.name, v_final_result);

        RETURN NEXT;
    END LOOP;

    RETURN;
END;
$$ LANGUAGE plpgsql;


-- ========================================
-- Migration: 000024_create_secure_release.up.sql
-- ========================================
-- Secure Print Release Table
-- Tracks print jobs that require secure release at the printer

CREATE TABLE IF NOT EXISTS secure_print_jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_id UUID NOT NULL UNIQUE REFERENCES print_jobs(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    release_method VARCHAR(50) NOT NULL CHECK (release_method IN ('pin', 'card', 'biometric', 'nfc', 'app')),
    release_data JSONB, -- Encrypted PIN, card ID, etc.
    status VARCHAR(50) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'released', 'expired', 'cancelled')),
    expires_at TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '24 hours'),
    released_at TIMESTAMPTZ,
    released_printer_id UUID REFERENCES printers(id),
    release_attempts INTEGER DEFAULT 0,
    max_release_attempts INTEGER DEFAULT 3,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_secure_print_jobs_job ON secure_print_jobs(job_id);
CREATE INDEX idx_secure_print_jobs_user ON secure_print_jobs(user_id);
CREATE INDEX idx_secure_print_jobs_status ON secure_print_jobs(status);
CREATE INDEX idx_secure_print_jobs_expires ON secure_print_jobs(expires_at);
CREATE INDEX idx_secure_print_jobs_printer ON secure_print_jobs(released_printer_id);

-- Secure Release Attempt Logs
-- Logs all release attempts for security auditing

CREATE TABLE IF NOT EXISTS secure_release_attempts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    secure_job_id UUID NOT NULL REFERENCES secure_print_jobs(id) ON DELETE CASCADE,
    attempted_at TIMESTAMPTZ DEFAULT NOW(),
    attempted_method VARCHAR(50),
    attempted_by VARCHAR(255), -- user_id, card_id, etc.
    success BOOLEAN NOT NULL DEFAULT false,
    failure_reason VARCHAR(255),
    ip_address INET,
    printer_id UUID REFERENCES printers(id)
);

CREATE INDEX idx_secure_release_attempts_secure_job ON secure_release_attempts(secure_job_id);
CREATE INDEX idx_secure_release_attempts_success ON secure_release_attempts(success);

-- Print Release Stations
-- Registers physical release stations/kiosks

CREATE TABLE IF NOT EXISTS print_release_stations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    location VARCHAR(255),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    supported_methods VARCHAR(100)[] NOT NULL, -- ARRAY of ['pin', 'card', 'nfc', etc.]
    assigned_printers UUID[],
    is_active BOOLEAN DEFAULT true,
    last_heartbeat TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_print_release_stations_org ON print_release_stations(organization_id);
CREATE INDEX idx_print_release_stations_active ON print_release_stations(is_active);

-- Triggers
CREATE TRIGGER update_secure_print_jobs_updated_at BEFORE UPDATE ON secure_print_jobs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_print_release_stations_updated_at BEFORE UPDATE ON print_release_stations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to check and expire secure print jobs
CREATE OR REPLACE FUNCTION check_expired_secure_jobs() RETURNS INTEGER AS $$
DECLARE
    v_expired_count INTEGER;
BEGIN
    -- Update expired jobs
    UPDATE secure_print_jobs
    SET status = 'expired'
    WHERE status = 'pending'
      AND expires_at < NOW();

    GET DIAGNOSTICS v_expired_count = ROW_COUNT;
    RETURN v_expired_count;
END;
$$ LANGUAGE plpgsql;

-- Function to attempt secure release
CREATE OR REPLACE FUNCTION attempt_secure_release(
    p_secure_job_id UUID,
    p_release_method VARCHAR,
    p_release_data JSONB,
    p_printer_id UUID DEFAULT NULL,
    p_ip_address INET DEFAULT NULL
) RETURNS BOOLEAN AS $$
DECLARE
    v_secure_job RECORD;
    v_attempts_remaining INTEGER;
BEGIN
    -- Get secure job info
    SELECT * INTO v_secure_job
    FROM secure_print_jobs
    WHERE id = p_secure_job_id AND status = 'pending'
    FOR UPDATE;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'Secure job not found or already processed';
    END IF;

    -- Check if expired
    IF v_secure_job.expires_at < NOW() THEN
        UPDATE secure_print_jobs SET status = 'expired' WHERE id = p_secure_job_id;
        RAISE EXCEPTION 'Secure job has expired';
    END IF;

    -- Check release attempts
    v_attempts_remaining := v_secure_job.max_release_attempts - v_secure_job.release_attempts - 1;
    IF v_attempts_remaining < 0 THEN
        UPDATE secure_print_jobs SET status = 'cancelled' WHERE id = p_secure_job_id;
        RAISE EXCEPTION 'Maximum release attempts exceeded';
    END IF;

    -- Validate release data (simplified - in production would decrypt and compare)
    -- For now, assume validation passes if method matches

    -- Log attempt
    INSERT INTO secure_release_attempts (
        secure_job_id, attempted_method, attempted_by,
        success, failure_reason, ip_address, printer_id
    ) VALUES (
        p_secure_job_id, p_release_method,
        p_release_data->>'user_id',
        true, NULL, p_ip_address, p_printer_id
    );

    -- Update job as released
    UPDATE secure_print_jobs
    SET status = 'released',
        released_at = NOW(),
        released_printer_id = p_printer_id,
        release_attempts = release_attempts + 1
    WHERE id = p_secure_job_id;

    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;


-- ========================================
-- Migration: 000025_create_mfa.up.sql
-- ========================================
-- Multi-Factor Authentication (MFA) Table
-- Supports TOTP, SMS, Email, Hardware Tokens, and Smart Cards

CREATE TABLE IF NOT EXISTS user_mfa (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    mfa_type VARCHAR(50) NOT NULL CHECK (mfa_type IN ('totp', 'sms', 'email', 'hardware_token', 'smart_card', 'biometric', 'push')),
    is_enabled BOOLEAN DEFAULT false,
    is_primary BOOLEAN DEFAULT false,
    secret_data JSONB, -- Encrypted secret for TOTP, phone number for SMS, etc.
    verified_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    backup_codes TEXT[], -- Encrypted backup codes for recovery
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, mfa_type)
);

CREATE INDEX idx_user_mfa_user ON user_mfa(user_id);
CREATE INDEX idx_user_mfa_enabled ON user_mfa(is_enabled);
CREATE INDEX idx_user_mfa_primary ON user_mfa(is_primary);

-- MFA Verification Attempts
-- Tracks MFA verification attempts for security monitoring

CREATE TABLE IF NOT EXISTS mfa_verification_attempts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    mfa_type VARCHAR(50) NOT NULL,
    success BOOLEAN NOT NULL,
    attempted_at TIMESTAMPTZ DEFAULT NOW(),
    ip_address INET,
    user_agent TEXT,
    failure_reason VARCHAR(255)
);

CREATE INDEX idx_mfa_verification_attempts_user ON mfa_verification_attempts(user_id);
CREATE INDEX idx_mfa_verification_attempts_success ON mfa_verification_attempts(success);
CREATE INDEX idx_mfa_verification_attempts_at ON mfa_verification_attempts(attempted_at);

-- Smart Cards Table
-- Registers smart cards for user authentication

CREATE TABLE IF NOT EXISTS user_smart_cards (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    card_id VARCHAR(255) NOT NULL, -- Encrypted card identifier
    card_type VARCHAR(100) NOT NULL, -- 'piv', 'cac', 'employee_badge', etc.
    issuer_dn VARCHAR(255),
    subject_dn VARCHAR(255),
    certificate_valid_from TIMESTAMPTZ,
    certificate_valid_until TIMESTAMPTZ,
    is_active BOOLEAN DEFAULT true,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_user_smart_cards_user ON user_smart_cards(user_id);
CREATE INDEX idx_user_smart_cards_card_id ON user_smart_cards(card_id);
CREATE INDEX idx_user_smart_cards_active ON user_smart_cards(is_active);

-- Smart Card Certificate Log
-- Logs smart card certificate validations

CREATE TABLE IF NOT EXISTS smart_card_cert_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    card_id VARCHAR(255) NOT NULL,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    serial_number VARCHAR(255),
    issuer VARCHAR(255),
    subject VARCHAR(255),
    valid_from TIMESTAMPTZ,
    valid_until TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    revocation_reason VARCHAR(255),
    logged_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_smart_card_cert_log_card ON smart_card_cert_log(card_id);
CREATE INDEX idx_smart_card_cert_log_user ON smart_card_cert_log(user_id);
CREATE INDEX idx_smart_card_cert_log_serial ON smart_card_cert_log(serial_number);

-- Hardware Tokens Table
-- Registers hardware security keys (YubiKey, etc.)

CREATE TABLE IF NOT EXISTS user_hardware_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    credential_id VARCHAR(255) NOT NULL UNIQUE,
    public_key TEXT NOT NULL,
    attestation_type VARCHAR(100),
    aaguid UUID,
    sign_count INTEGER DEFAULT 0,
    is_backup BOOLEAN DEFAULT false,
    is_active BOOLEAN DEFAULT true,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_user_hardware_tokens_user ON user_hardware_tokens(user_id);
CREATE INDEX idx_user_hardware_tokens_credential ON user_hardware_tokens(credential_id);
CREATE INDEX idx_user_hardware_tokens_active ON user_hardware_tokens(is_active);

-- Triggers
CREATE TRIGGER update_user_mfa_updated_at BEFORE UPDATE ON user_mfa
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_user_smart_cards_updated_at BEFORE UPDATE ON user_smart_cards
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_user_hardware_tokens_updated_at BEFORE UPDATE ON user_hardware_tokens
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to check if user requires MFA
CREATE OR REPLACE FUNCTION user_requires_mfa(p_user_id UUID) RETURNS BOOLEAN AS $$
DECLARE
    v_enabled_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO v_enabled_count
    FROM user_mfa
    WHERE user_id = p_user_id
      AND is_enabled = true;

    RETURN v_enabled_count > 0;
END;
$$ LANGUAGE plpgsql;

-- Function to get user's primary MFA method
CREATE OR REPLACE FUNCTION get_primary_mfa_method(p_user_id UUID) RETURNS VARCHAR AS $$
DECLARE
    v_mfa_type VARCHAR;
BEGIN
    SELECT mfa_type INTO v_mfa_type
    FROM user_mfa
    WHERE user_id = p_user_id
      AND is_enabled = true
      AND is_primary = true
    LIMIT 1;

    -- If no primary, return first enabled
    IF v_mfa_type IS NULL THEN
        SELECT mfa_type INTO v_mfa_type
        FROM user_mfa
        WHERE user_id = p_user_id
          AND is_enabled = true
        LIMIT 1;
    END IF;

    RETURN v_mfa_type;
END;
$$ LANGUAGE plpgsql;

-- Function to log MFA verification attempt
CREATE OR REPLACE FUNCTION log_mfa_attempt(
    p_user_id UUID,
    p_mfa_type VARCHAR,
    p_success BOOLEAN,
    p_ip_address INET DEFAULT NULL,
    p_user_agent TEXT DEFAULT NULL,
    p_failure_reason VARCHAR DEFAULT NULL
) RETURNS UUID AS $$
DECLARE
    v_attempt_id UUID;
BEGIN
    INSERT INTO mfa_verification_attempts (
        user_id, mfa_type, success, ip_address, user_agent, failure_reason
    ) VALUES (
        p_user_id, p_mfa_type, p_success, p_ip_address, p_user_agent, p_failure_reason
    )
    RETURNING id INTO v_attempt_id;

    -- Check for too many failed attempts
    IF NOT p_success THEN
        DECLARE
            v_failed_count INTEGER;
        BEGIN
            SELECT COUNT(*) INTO v_failed_count
            FROM mfa_verification_attempts
            WHERE user_id = p_user_id
              AND success = false
              AND attempted_at > NOW() - INTERVAL '15 minutes';

            IF v_failed_count >= 10 THEN
                -- Could trigger account lockout here
                RAISE WARNING 'User % has had % failed MFA attempts in 15 minutes', p_user_id, v_failed_count;
            END IF;
        END;
    END IF;

    RETURN v_attempt_id;
END;
$$ LANGUAGE plpgsql;


-- ========================================
-- Migration: 000026_create_analytics.up.sql
-- ========================================
-- Analytics Aggregation Tables
-- Stores pre-aggregated analytics data for reporting

CREATE TABLE IF NOT EXISTS print_usage_by_day (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    printer_id UUID REFERENCES printers(id) ON DELETE SET NULL,
    date DATE NOT NULL,
    total_jobs INTEGER DEFAULT 0,
    total_pages INTEGER DEFAULT 0,
    color_pages INTEGER DEFAULT 0,
    duplex_pages INTEGER DEFAULT 0,
    total_cost DECIMAL(10, 4) DEFAULT 0,
    currency VARCHAR(3) DEFAULT 'USD',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_print_usage_by_day_unique ON print_usage_by_day(organization_id, COALESCE(user_id, '00000000-0000-0000-0000-000000000000'::UUID), COALESCE(printer_id, '00000000-0000-0000-0000-000000000000'::UUID), "date");
CREATE INDEX idx_print_usage_by_day_org ON print_usage_by_day(organization_id);
CREATE INDEX idx_print_usage_by_day_user ON print_usage_by_day(user_id);
CREATE INDEX idx_print_usage_by_day_printer ON print_usage_by_day(printer_id);
CREATE INDEX idx_print_usage_by_day_date ON print_usage_by_day("date");

CREATE TABLE IF NOT EXISTS print_usage_by_week (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    printer_id UUID REFERENCES printers(id) ON DELETE SET NULL,
    week_start DATE NOT NULL,
    week_end DATE NOT NULL,
    total_jobs INTEGER DEFAULT 0,
    total_pages INTEGER DEFAULT 0,
    color_pages INTEGER DEFAULT 0,
    duplex_pages INTEGER DEFAULT 0,
    total_cost DECIMAL(10, 4) DEFAULT 0,
    currency VARCHAR(3) DEFAULT 'USD',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_print_usage_by_week_unique ON print_usage_by_week(organization_id, COALESCE(user_id, '00000000-0000-0000-0000-000000000000'::UUID), COALESCE(printer_id, '00000000-0000-0000-0000-000000000000'::UUID), week_start);
CREATE INDEX idx_print_usage_by_week_org ON print_usage_by_week(organization_id);
CREATE INDEX idx_print_usage_by_week_user ON print_usage_by_week(user_id);
CREATE INDEX idx_print_usage_by_week_printer ON print_usage_by_week(printer_id);
CREATE INDEX idx_print_usage_by_week_start ON print_usage_by_week(week_start);

CREATE TABLE IF NOT EXISTS print_usage_by_month (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    printer_id UUID REFERENCES printers(id) ON DELETE SET NULL,
    month INTEGER NOT NULL CHECK (month BETWEEN 1 AND 12),
    year INTEGER NOT NULL,
    total_jobs INTEGER DEFAULT 0,
    total_pages INTEGER DEFAULT 0,
    color_pages INTEGER DEFAULT 0,
    duplex_pages INTEGER DEFAULT 0,
    total_cost DECIMAL(10, 4) DEFAULT 0,
    currency VARCHAR(3) DEFAULT 'USD',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_print_usage_by_month_unique ON print_usage_by_month(organization_id, COALESCE(user_id, '00000000-0000-0000-0000-000000000000'::UUID), COALESCE(printer_id, '00000000-0000-0000-0000-000000000000'::UUID), "month", year);
CREATE INDEX idx_print_usage_by_month_org ON print_usage_by_month(organization_id);
CREATE INDEX idx_print_usage_by_month_user ON print_usage_by_month(user_id);
CREATE INDEX idx_print_usage_by_month_printer ON print_usage_by_month(printer_id);
CREATE INDEX idx_print_usage_by_month_my ON print_usage_by_month("month", year);

-- Printer Performance Metrics
-- Tracks printer performance and health metrics

CREATE TABLE IF NOT EXISTS printer_metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    printer_id UUID NOT NULL REFERENCES printers(id) ON DELETE CASCADE,
    metric_date DATE NOT NULL DEFAULT CURRENT_DATE,
    total_jobs INTEGER DEFAULT 0,
    completed_jobs INTEGER DEFAULT 0,
    failed_jobs INTEGER DEFAULT 0,
    cancelled_jobs INTEGER DEFAULT 0,
    total_pages INTEGER DEFAULT 0,
    average_job_time_seconds INTEGER,
    uptime_seconds INTEGER DEFAULT 0,
    downtime_seconds INTEGER DEFAULT 0,
    error_count INTEGER DEFAULT 0,
    warning_count INTEGER DEFAULT 0,
    toner_level_percentage INTEGER,
    paper_level_percentage INTEGER,
    maintenance_required BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(printer_id, metric_date)
);

CREATE INDEX idx_printer_metrics_printer ON printer_metrics(printer_id);
CREATE INDEX idx_printer_metrics_date ON printer_metrics(metric_date);
CREATE INDEX idx_printer_metrics_maintenance ON printer_metrics(maintenance_required);

-- User Activity Summary
-- Daily summary of user print activity

CREATE TABLE IF NOT EXISTS user_activity_summary (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    activity_date DATE NOT NULL DEFAULT CURRENT_DATE,
    jobs_submitted INTEGER DEFAULT 0,
    jobs_completed INTEGER DEFAULT 0,
    jobs_cancelled INTEGER DEFAULT 0,
    pages_printed INTEGER DEFAULT 0,
    estimated_cost DECIMAL(10, 4) DEFAULT 0,
    currency VARCHAR(3) DEFAULT 'USD',
    last_print_time TIMESTAMPTZ,
    most_used_printer_id UUID REFERENCES printers(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, activity_date)
);

CREATE INDEX idx_user_activity_summary_user ON user_activity_summary(user_id);
CREATE INDEX idx_user_activity_summary_org ON user_activity_summary(organization_id);
CREATE INDEX idx_user_activity_summary_date ON user_activity_summary(activity_date);

-- Cost Center Reports
-- Aggregates costs by cost center for billing

CREATE TABLE IF NOT EXISTS cost_center_reports (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    cost_center_id VARCHAR(100) NOT NULL,
    cost_center_name VARCHAR(255),
    report_period_start DATE NOT NULL,
    report_period_end DATE NOT NULL,
    total_jobs INTEGER DEFAULT 0,
    total_pages INTEGER DEFAULT 0,
    total_cost DECIMAL(10, 4) DEFAULT 0,
    currency VARCHAR(3) DEFAULT 'USD',
    breakdown_by_printer JSONB,
    breakdown_by_user JSONB,
    generated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(organization_id, cost_center_id, report_period_start, report_period_end)
);

CREATE INDEX idx_cost_center_reports_org ON cost_center_reports(organization_id);
CREATE INDEX idx_cost_center_reports_period ON cost_center_reports(report_period_start, report_period_end);

-- Triggers
CREATE TRIGGER update_print_usage_by_day_updated_at BEFORE UPDATE ON print_usage_by_day
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_print_usage_by_week_updated_at BEFORE UPDATE ON print_usage_by_week
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_print_usage_by_month_updated_at BEFORE UPDATE ON print_usage_by_month
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_printer_metrics_updated_at BEFORE UPDATE ON printer_metrics
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_user_activity_summary_updated_at BEFORE UPDATE ON user_activity_summary
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to aggregate daily usage
CREATE OR REPLACE FUNCTION aggregate_daily_usage(p_date DATE DEFAULT CURRENT_DATE) RETURNS INTEGER AS $$
DECLARE
    v_aggregated_count INTEGER := 0;
BEGIN
    -- Aggregate from print_jobs into daily summary
    INSERT INTO print_usage_by_day (
        organization_id, user_id, printer_id, "date",
        total_jobs, total_pages, total_cost
    )
    SELECT
        u.organization_id,
        j.user_name::UUID, -- This would need to be adjusted based on actual schema
        j.printer_id::UUID,
        p_date,
        COUNT(*),
        COALESCE(SUM(j.copies), 0),
        COALESCE(SUM(c.cost), 0)
    FROM print_jobs j
    LEFT JOIN users u ON u.email = j.user_email
    LEFT JOIN print_job_costs c ON c.job_id = j.id
    WHERE DATE(j.created_at) = p_date
    GROUP BY u.organization_id, j.user_name, j.printer_id
    ON CONFLICT (organization_id, COALESCE(user_id, '00000000-0000-0000-0000-000000000000'::UUID), COALESCE(printer_id, '00000000-0000-0000-0000-000000000000'::UUID), "date")
    DO UPDATE SET
        total_jobs = EXCLUDED.total_jobs + print_usage_by_day.total_jobs,
        total_pages = EXCLUDED.total_pages + print_usage_by_day.total_pages,
        total_cost = EXCLUDED.total_cost + print_usage_by_day.total_cost;

    GET DIAGNOSTICS v_aggregated_count = ROW_COUNT;
    RETURN v_aggregated_count;
END;
$$ LANGUAGE plpgsql;

-- Function to get top printers by usage
CREATE OR REPLACE FUNCTION get_top_printers(
    p_organization_id UUID,
    p_start_date DATE,
    p_end_date DATE,
    p_limit INTEGER DEFAULT 10
) RETURNS TABLE(
    printer_id UUID,
    printer_name VARCHAR,
    total_jobs BIGINT,
    total_pages BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        p.id,
        p.name,
        COALESCE(SUM(d.total_jobs), 0)::BIGINT,
        COALESCE(SUM(d.total_pages), 0)::BIGINT
    FROM printers p
    LEFT JOIN print_usage_by_day d ON d.printer_id = p.id
        AND d.date BETWEEN p_start_date AND p_end_date
    WHERE p.organization_id = p_organization_id
    GROUP BY p.id, p.name
    ORDER BY total_pages DESC
    LIMIT p_limit;
END;
$$ LANGUAGE plpgsql;

-- Function to get top users by usage
CREATE OR REPLACE FUNCTION get_top_users(
    p_organization_id UUID,
    p_start_date DATE,
    p_end_date DATE,
    p_limit INTEGER DEFAULT 10
) RETURNS TABLE(
    user_id UUID,
    user_email VARCHAR,
    total_jobs BIGINT,
    total_pages BIGINT,
    total_cost DECIMAL
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        u.id,
        u.email,
        COALESCE(SUM(d.total_jobs), 0)::BIGINT,
        COALESCE(SUM(d.total_pages), 0)::BIGINT,
        COALESCE(SUM(d.total_cost), 0)
    FROM users u
    LEFT JOIN print_usage_by_day d ON d.user_id = u.id
        AND d.date BETWEEN p_start_date AND p_end_date
    WHERE u.organization_id = p_organization_id
    GROUP BY u.id, u.email
    ORDER BY total_pages DESC
    LIMIT p_limit;
END;
$$ LANGUAGE plpgsql;


-- ========================================
-- Migration: 000027_create_watermark_templates.up.sql
-- ========================================
-- Watermark Templates Table
-- Stores watermark templates for document processing

CREATE TABLE IF NOT EXISTS watermark_templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL CHECK (type IN ('text', 'image', 'overlay')),
    content TEXT,
    position VARCHAR(50) DEFAULT 'center' CHECK (position IN ('top-left', 'top-center', 'top-right', 'center', 'bottom-left', 'bottom-center', 'bottom-right')),
    opacity DECIMAL(3, 2) DEFAULT 0.3 CHECK (opacity >= 0 AND opacity <= 1),
    rotation INTEGER DEFAULT 0 CHECK (rotation >= 0 AND rotation <= 360),
    font_size INTEGER DEFAULT 48,
    font_color VARCHAR(7) DEFAULT '#CCCCCC',
    image_data BYTEA,
    is_default BOOLEAN DEFAULT false,
    apply_to_all BOOLEAN DEFAULT false,
    created_by VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_watermark_templates_org ON watermark_templates(organization_id);
CREATE INDEX idx_watermark_templates_default ON watermark_templates(organization_id, is_default) WHERE is_default = true;

CREATE TRIGGER update_watermark_templates_updated_at BEFORE UPDATE ON watermark_templates
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Budget Allocations Table (for cost tracking by department)

CREATE TABLE IF NOT EXISTS budget_allocations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    cost_center_id VARCHAR(100) NOT NULL,
    cost_center_name VARCHAR(255),
    budget_amount DECIMAL(12, 2) NOT NULL,
    spent_amount DECIMAL(12, 2) DEFAULT 0,
    currency VARCHAR(3) DEFAULT 'USD',
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(organization_id, cost_center_id, period_start)
);

CREATE INDEX idx_budget_allocations_org ON budget_allocations(organization_id);
CREATE INDEX idx_budget_allocations_period ON budget_allocations(period_start, period_end);

CREATE TRIGGER update_budget_allocations_updated_at BEFORE UPDATE ON budget_allocations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Scheduled Reports Table

CREATE TABLE IF NOT EXISTS scheduled_reports (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    report_type VARCHAR(50) NOT NULL,
    schedule VARCHAR(20) NOT NULL CHECK (schedule IN ('daily', 'weekly', 'monthly', 'quarterly')),
    recipients JSONB,
    format VARCHAR(10) DEFAULT 'json' CHECK (format IN ('json', 'csv', 'pdf', 'xlsx')),
    is_active BOOLEAN DEFAULT true,
    last_run_at TIMESTAMPTZ,
    next_run_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_scheduled_reports_org ON scheduled_reports(organization_id);
CREATE INDEX idx_scheduled_reports_active ON scheduled_reports(is_active) WHERE is_active = true;
CREATE INDEX idx_scheduled_reports_next_run ON scheduled_reports(next_run_at) WHERE is_active = true;

CREATE TRIGGER update_scheduled_reports_updated_at BEFORE UPDATE ON scheduled_reports
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Mobile Devices Table

CREATE TABLE IF NOT EXISTS mobile_devices (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_name VARCHAR(255) NOT NULL,
    device_type VARCHAR(50) DEFAULT 'unknown',
    device_token TEXT,
    app_version VARCHAR(50),
    os_version VARCHAR(50),
    is_active BOOLEAN DEFAULT true,
    last_seen TIMESTAMPTZ DEFAULT NOW(),
    pairing_code VARCHAR(20) UNIQUE,
    paired_printer_id UUID REFERENCES printers(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_mobile_devices_user ON mobile_devices(user_id);
CREATE INDEX idx_mobile_devices_pairing ON mobile_devices(pairing_code);
CREATE INDEX idx_mobile_devices_printer ON mobile_devices(paired_printer_id);

CREATE TRIGGER update_mobile_devices_updated_at BEFORE UPDATE ON mobile_devices
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Push Notifications Table

CREATE TABLE IF NOT EXISTS push_notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    device_id UUID NOT NULL REFERENCES mobile_devices(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    body TEXT,
    data JSONB,
    priority INTEGER DEFAULT 5 CHECK (priority >= 0 AND priority <= 10),
    ttl INTERVAL DEFAULT '1 day',
    scheduled_at TIMESTAMPTZ DEFAULT NOW(),
    sent_at TIMESTAMPTZ,
    failed_at TIMESTAMPTZ,
    error TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_push_notifications_device ON push_notifications(device_id);
CREATE INDEX idx_push_notifications_status ON push_notifications(sent_at, failed_at) WHERE sent_at IS NULL AND failed_at IS NULL;

-- API Keys Table (for developer portal)
-- Note: api_keys table is created in migration 000011 with org_id column
-- This section adds additional columns if needed

-- Add new columns to api_keys if they don't exist
DO $$
BEGIN
    -- Add key_prefix if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'api_keys' AND column_name = 'key_prefix') THEN
        ALTER TABLE api_keys ADD COLUMN key_prefix VARCHAR(8);
    END IF;

    -- Add rate_limit if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'api_keys' AND column_name = 'rate_limit') THEN
        ALTER TABLE api_keys ADD COLUMN rate_limit INTEGER DEFAULT 60;
    END IF;

    -- Add created_by if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'api_keys' AND column_name = 'created_by') THEN
        ALTER TABLE api_keys ADD COLUMN created_by VARCHAR(255);
    END IF;

    -- Add updated_at if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'api_keys' AND column_name = 'updated_at') THEN
        ALTER TABLE api_keys ADD COLUMN updated_at TIMESTAMPTZ DEFAULT NOW();
    END IF;

    -- Create trigger for updated_at if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.triggers WHERE trigger_name = 'update_api_keys_updated_at') THEN
        CREATE TRIGGER update_api_keys_updated_at BEFORE UPDATE ON api_keys
            FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
END $$;

-- API Usage Logs Table (for rate limiting and analytics)

CREATE TABLE IF NOT EXISTS api_usage_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    api_key_id UUID REFERENCES api_keys(id) ON DELETE SET NULL,
    organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
    method VARCHAR(10) NOT NULL,
    path TEXT NOT NULL,
    status_code INTEGER,
    latency_ms INTEGER,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_api_usage_logs_key ON api_usage_logs(api_key_id);
CREATE INDEX idx_api_usage_logs_org ON api_usage_logs(organization_id);
CREATE INDEX idx_api_usage_logs_created ON api_usage_logs(created_at);

-- Webhooks Table
-- Note: webhooks table is created in migration 000012 with org_id column
-- This section adds additional columns if needed

DO $$
BEGIN
    -- Add secret if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'webhooks' AND column_name = 'secret') THEN
        ALTER TABLE webhooks ADD COLUMN secret VARCHAR(255) NOT NULL DEFAULT uuid_generate_v4()::text;
    END IF;

    -- Add headers if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'webhooks' AND column_name = 'headers') THEN
        ALTER TABLE webhooks ADD COLUMN headers JSONB;
    END IF;
END $$;

-- Audit Logs Table

CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID,
    user_id UUID,
    api_key_id UUID,
    action VARCHAR(100),
    resource VARCHAR(500),
    method VARCHAR(10),
    path TEXT,
    status_code INTEGER,
    ip_address INET,
    user_agent TEXT,
    request_id VARCHAR(100) UNIQUE,
    latency_ms INTEGER,
    request_body JSONB,
    response_size INTEGER,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_org ON audit_logs(organization_id);
CREATE INDEX idx_audit_logs_user ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_created ON audit_logs(created_at);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);

-- Cost Center Column for Users

ALTER TABLE users ADD COLUMN IF NOT EXISTS cost_center VARCHAR(100);

-- Functions

-- Function to check if an API key has required scope
CREATE OR REPLACE FUNCTION check_api_key_scope(
    p_key_hash VARCHAR,
    p_required_scope VARCHAR
) RETURNS BOOLEAN AS $$
DECLARE
    v_scopes TEXT[];
BEGIN
    SELECT scopes INTO v_scopes
    FROM api_keys
    WHERE key_hash = p_key_hash
      AND is_active = true
      AND (expires_at IS NULL OR expires_at > NOW());

    IF NOT FOUND THEN
        RETURN FALSE;
    END IF;

    RETURN p_required_scope = ANY(v_scopes) OR 'admin' = ANY(v_scopes);
END;
$$ LANGUAGE plpgsql;

-- Function to record API usage
CREATE OR REPLACE FUNCTION record_api_usage(
    p_api_key_id UUID,
    p_organization_id UUID,
    p_method VARCHAR,
    p_path TEXT,
    p_status_code INTEGER,
    p_latency_ms INTEGER
) RETURNS VOID AS $$
BEGIN
    INSERT INTO api_usage_logs (
        api_key_id, organization_id, method, path, status_code, latency_ms
    ) VALUES (
        p_api_key_id, p_organization_id, p_method, p_path, p_status_code, p_latency_ms
    );
END;
$$ LANGUAGE plpgsql;

-- Function to get budget status
CREATE OR REPLACE FUNCTION get_budget_status(
    p_organization_id UUID,
    p_cost_center_id VARCHAR
) RETURNS DECIMAL AS $$
DECLARE
    v_budget DECIMAL(12,2);
    v_spent DECIMAL(12,2);
    v_remaining DECIMAL(12,2);
    v_percentage DECIMAL(5,2);
BEGIN
    SELECT COALESCE(budget_amount, 0), COALESCE(spent_amount, 0)
    INTO v_budget, v_spent
    FROM budget_allocations
    WHERE organization_id = p_organization_id
      AND cost_center_id = p_cost_center_id
      AND period_start <= NOW()
      AND period_end >= NOW()
    ORDER BY period_start DESC
    LIMIT 1;

    IF v_budget = 0 THEN
        RETURN 100; -- Unlimited
    END IF;

    v_remaining := v_budget - v_spent;
    v_percentage := (v_spent / v_budget) * 100;

    RETURN v_percentage;
END;
$$ LANGUAGE plpgsql;

-- Function to deduct from budget
CREATE OR REPLACE FUNCTION deduct_budget(
    p_organization_id UUID,
    p_cost_center_id VARCHAR,
    p_amount DECIMAL
) RETURNS BOOLEAN AS $$
DECLARE
    v_budget_id UUID;
    v_current_spent DECIMAL(12,2);
    v_budget_amount DECIMAL(12,2);
BEGIN
    SELECT id, spent_amount, budget_amount
    INTO v_budget_id, v_current_spent, v_budget_amount
    FROM budget_allocations
    WHERE organization_id = p_organization_id
      AND cost_center_id = p_cost_center_id
      AND period_start <= NOW()
      AND period_end >= NOW()
    FOR UPDATE;

    IF NOT FOUND THEN
        -- No budget set, allow spending
        RETURN TRUE;
    END IF;

    IF v_budget_amount > 0 AND (v_current_spent + p_amount) > v_budget_amount THEN
        RETURN FALSE; -- Over budget
    END IF;

    UPDATE budget_allocations
    SET spent_amount = spent_amount + p_amount,
        updated_at = NOW()
    WHERE id = v_budget_id;

    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;


-- ========================================
-- Migration: 000028_create_user_printer_mappings.up.sql
-- ========================================
-- User-printer mappings: maps RDP session users to their local client-side agents/printers
-- Used for routing print jobs captured on RDP session hosts to the user's local printer
CREATE TABLE IF NOT EXISTS user_printer_mappings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID REFERENCES organizations(id),
    -- The user who prints on the RDP session host
    user_email VARCHAR(255) NOT NULL,
    user_name VARCHAR(255),
    -- The client-side agent that manages the user's local printers
    client_agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    -- The specific local printer to route jobs to (discovered_printers.id)
    target_printer_id UUID,
    -- The printer name on the client machine (for display and matching)
    target_printer_name VARCHAR(500),
    -- The server-side agent that captures print jobs (RDP session host)
    server_agent_id UUID REFERENCES agents(id) ON DELETE SET NULL,
    -- Whether this mapping is currently active
    is_active BOOLEAN NOT NULL DEFAULT true,
    -- Whether this is the default mapping for the user
    is_default BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- A user can have one default mapping per organization
    CONSTRAINT unique_user_default_mapping UNIQUE (organization_id, user_email, is_default) DEFERRABLE INITIALLY DEFERRED
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_user_printer_mappings_email ON user_printer_mappings(user_email);
CREATE INDEX IF NOT EXISTS idx_user_printer_mappings_client_agent ON user_printer_mappings(client_agent_id);
CREATE INDEX IF NOT EXISTS idx_user_printer_mappings_server_agent ON user_printer_mappings(server_agent_id);
CREATE INDEX IF NOT EXISTS idx_user_printer_mappings_org ON user_printer_mappings(organization_id);
CREATE INDEX IF NOT EXISTS idx_user_printer_mappings_active ON user_printer_mappings(user_email, is_active) WHERE is_active = true;

-- Add agent_role column to agents table to distinguish server vs client agents
ALTER TABLE agents ADD COLUMN IF NOT EXISTS agent_role VARCHAR(50) NOT NULL DEFAULT 'standard';
-- agent_role values: 'server' (RDP session host), 'client' (user workstation), 'standard' (both/legacy)

COMMENT ON TABLE user_printer_mappings IS 'Maps users on RDP session hosts to their local printers via client-side agents';
COMMENT ON COLUMN user_printer_mappings.client_agent_id IS 'Agent running on the user workstation that manages local printers';
COMMENT ON COLUMN user_printer_mappings.server_agent_id IS 'Agent running on the RDP session host that captures print jobs';
COMMENT ON COLUMN user_printer_mappings.target_printer_id IS 'Specific discovered printer on the client agent to route jobs to';
COMMENT ON COLUMN agents.agent_role IS 'Role: server (RDP host capture), client (local print), standard (both)';


-- ========================================
-- Migration: 000029_add_printer_type_to_mappings.up.sql
-- ========================================
-- Add printer_type column to user_printer_mappings to distinguish standard vs receipt printers.
-- This allows users to have separate default mappings for normal documents and receipt/invoice output.
ALTER TABLE user_printer_mappings ADD COLUMN IF NOT EXISTS printer_type VARCHAR(50) NOT NULL DEFAULT 'standard';
-- printer_type values: 'standard' (A4/Letter, PostScript), 'receipt' (thermal, narrow paper, ESC/POS)

-- Drop the old unique constraint that only considered org+email+is_default
ALTER TABLE user_printer_mappings DROP CONSTRAINT IF EXISTS unique_user_default_mapping;

-- New unique constraint: one default mapping per user per printer_type per organization
ALTER TABLE user_printer_mappings ADD CONSTRAINT unique_user_default_per_type
    UNIQUE (organization_id, user_email, printer_type, is_default) DEFERRABLE INITIALLY DEFERRED;

CREATE INDEX IF NOT EXISTS idx_user_printer_mappings_type ON user_printer_mappings(printer_type);

COMMENT ON COLUMN user_printer_mappings.printer_type IS 'Printer type: standard (A4/Letter documents) or receipt (thermal/POS printers)';


-- ========================================
-- Migration: 000030_seed_admin_user.up.sql
-- ========================================
-- Seed Default Admin User
-- This migration creates a default admin account for initial setup
-- IMPORTANT: Change the password after first login!

-- Create default organization for the admin user
INSERT INTO organizations (
    id,
    name,
    slug,
    plan,
    created_at,
    updated_at
) VALUES (
    '00000000-0000-0000-0000-000000000001',
    'Default Organization',
    'default',
    'enterprise',
    NOW(),
    NOW()
) ON CONFLICT (id) DO NOTHING;

-- Default admin user
-- Email: admin@openprint.local
-- Password: Admin123!
-- The password hash is bcrypt hash of "Admin123!"
INSERT INTO users (
    id,
    email,
    password,
    first_name,
    last_name,
    role,
    organization_id,
    is_active,
    created_at,
    updated_at
) VALUES (
    '00000000-0000-0000-0000-000000000001',
    'admin@openprint.local',
    '$2a$12$CfucjOqA.F4RHQqYme2Oj.KiwtO9/kM79KftjEhfHuHq7aA7YQpIS', -- bcrypt hash of "Admin123!"
    'System',
    'Administrator',
    'admin',
    '00000000-0000-0000-0000-000000000001', -- Default organization
    true,
    NOW(),
    NOW()
) ON CONFLICT (email) DO NOTHING;


-- ========================================
-- Migration: 000031_extend_print_policies.up.sql
-- ========================================
-- Extend Print Policies for Enhanced Policy Engine
-- Adds JSONB columns for complex rules, actions, and scoping

-- Add new columns to existing print_policies table
ALTER TABLE print_policies
ADD COLUMN IF NOT EXISTS type VARCHAR(50) DEFAULT 'general',
ADD COLUMN IF NOT EXISTS status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'draft', 'archived')),
ADD COLUMN IF NOT EXISTS rules JSONB DEFAULT '[]'::jsonb,
ADD COLUMN IF NOT EXISTS actions JSONB DEFAULT '[]'::jsonb,
ADD COLUMN IF NOT EXISTS scope JSONB DEFAULT '{}'::jsonb,
ADD COLUMN IF NOT EXISTS created_by UUID REFERENCES users(id),
ADD COLUMN IF NOT EXISTS modified_by UUID REFERENCES users(id),
ADD COLUMN IF NOT EXISTS version INTEGER DEFAULT 1,
ADD COLUMN IF NOT EXISTS evaluated_count INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS triggered_count INTEGER DEFAULT 0;

-- Create index on policy type
CREATE INDEX IF NOT EXISTS idx_print_policies_type ON print_policies(type);

-- Create index on policy status
CREATE INDEX IF NOT EXISTS idx_print_policies_status ON print_policies(status);

-- Create GIN index on rules for JSONB queries
CREATE INDEX IF NOT EXISTS idx_print_policies_rules ON print_policies USING GIN(rules);

-- Create GIN index on scope for JSONB queries
CREATE INDEX IF NOT EXISTS idx_print_policies_scope ON print_policies USING GIN(scope);

-- Add comments for documentation
COMMENT ON COLUMN print_policies.type IS 'Policy type: quota, access, content, routing, watermark, retention, cost_center';
COMMENT ON COLUMN print_policies.status IS 'Policy status: active, inactive, draft, archived';
COMMENT ON COLUMN print_policies.rules IS 'JSON array of rule conditions for policy evaluation';
COMMENT ON COLUMN print_policies.actions IS 'JSON array of actions to take when policy is triggered';
COMMENT ON COLUMN print_policies.scope IS 'JSON object defining policy scope (users, groups, printers, etc.)';
COMMENT ON COLUMN print_policies.version IS 'Version number for optimistic locking';
COMMENT ON COLUMN print_policies.evaluated_count IS 'Number of times this policy has been evaluated';
COMMENT ON COLUMN print_policies.triggered_count IS 'Number of times this policy has been triggered';

-- Function to increment evaluation count
CREATE OR REPLACE FUNCTION increment_policy_evaluation_count(p_policy_id UUID)
RETURNS VOID AS $$
BEGIN
    UPDATE print_policies
    SET evaluated_count = evaluated_count + 1
    WHERE id = p_policy_id;
END;
$$ LANGUAGE plpgsql;

-- Function to increment triggered count
CREATE OR REPLACE FUNCTION increment_policy_triggered_count(p_policy_id UUID)
RETURNS VOID AS $$
BEGIN
    UPDATE print_policies
    SET triggered_count = triggered_count + 1
    WHERE id = p_policy_id;
END;
$$ LANGUAGE plpgsql;


-- ========================================
-- Migration: 000032_create_compliance_tables.up.sql
-- ========================================
-- Compliance Service Tables
-- Tables for FedRAMP, HIPAA, GDPR, and SOC2 compliance tracking

-- Compliance Controls Table
CREATE TABLE IF NOT EXISTS compliance_controls (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    framework VARCHAR(20) NOT NULL CHECK (framework IN ('fedramp', 'hipaa', 'gdpr', 'soc2')),
    family VARCHAR(100) NOT NULL,
    title VARCHAR(500) NOT NULL,
    description TEXT,
    implementation TEXT,
    status VARCHAR(30) NOT NULL DEFAULT 'unknown' CHECK (status IN ('compliant', 'non_compliant', 'pending', 'not_applicable', 'unknown')),
    last_assessed TIMESTAMPTZ,
    next_review TIMESTAMPTZ,
    evidence_count INTEGER DEFAULT 0,
    policies JSONB DEFAULT '[]'::jsonb,
    responsible_team VARCHAR(255),
    risk_level VARCHAR(20) DEFAULT 'medium' CHECK (risk_level IN ('low', 'medium', 'high', 'critical')),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(framework, family, id)
);

CREATE INDEX idx_compliance_controls_framework ON compliance_controls(framework);
CREATE INDEX idx_compliance_controls_status ON compliance_controls(status);
CREATE INDEX idx_compliance_controls_family ON compliance_controls(family);
CREATE INDEX idx_compliance_controls_next_review ON compliance_controls(next_review);

-- Data Breaches Table
CREATE TABLE IF NOT EXISTS data_breaches (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    discovered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reported_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    affected_records INTEGER DEFAULT 0,
    data_types JSONB DEFAULT '[]'::jsonb,
    description TEXT,
    containment_status VARCHAR(50) DEFAULT 'identifying',
    notification_sent BOOLEAN DEFAULT FALSE,
    resolved_at TIMESTAMPTZ,
    lessons_learned TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_data_breaches_discovered ON data_breaches(discovered_at);
CREATE INDEX idx_data_breaches_severity ON data_breaches(severity);
CREATE INDEX idx_data_breaches_status ON data_breaches(containment_status);

-- Remediation Plans Table
CREATE TABLE IF NOT EXISTS remediation_plans (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    control_id UUID REFERENCES compliance_controls(id) ON DELETE CASCADE,
    description TEXT NOT NULL,
    priority VARCHAR(20) NOT NULL CHECK (priority IN ('low', 'medium', 'high', 'critical')),
    target_date TIMESTAMPTZ NOT NULL,
    assignee VARCHAR(255),
    status VARCHAR(30) DEFAULT 'open' CHECK (status IN ('open', 'in_progress', 'completed', 'on_hold', 'cancelled')),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_remediation_plans_control ON remediation_plans(control_id);
CREATE INDEX idx_remediation_plans_status ON remediation_plans(status);
CREATE INDEX idx_remediation_plans_target_date ON remediation_plans(target_date);

-- Compliance Findings Table
CREATE TABLE IF NOT EXISTS compliance_findings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    control_id UUID REFERENCES compliance_controls(id) ON DELETE CASCADE,
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('info', 'low', 'medium', 'high', 'critical')),
    title VARCHAR(500) NOT NULL,
    description TEXT,
    recommendation TEXT,
    status VARCHAR(30) DEFAULT 'open' CHECK (status IN ('open', 'in_progress', 'closed', 'deferred')),
    opened_at TIMESTAMPTZ DEFAULT NOW(),
    closed_at TIMESTAMPTZ,
    created_by UUID REFERENCES users(id),
    closed_by UUID REFERENCES users(id)
);

CREATE INDEX idx_compliance_findings_control ON compliance_findings(control_id);
CREATE INDEX idx_compliance_findings_severity ON compliance_findings(severity);
CREATE INDEX idx_compliance_findings_status ON compliance_findings(status);

-- Evidence Items Table
CREATE TABLE IF NOT EXISTS compliance_evidence (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    finding_id UUID REFERENCES compliance_findings(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL, -- 'screenshot', 'document', 'log', 'config', 'other'
    description TEXT,
    file_path TEXT,
    file_hash VARCHAR(255),
    collected_at TIMESTAMPTZ DEFAULT NOW(),
    collected_by UUID REFERENCES users(id),
    metadata JSONB DEFAULT '{}'::jsonb
);

CREATE INDEX idx_compliance_evidence_finding ON compliance_evidence(finding_id);
CREATE INDEX idx_compliance_evidence_type ON compliance_evidence(type);

-- Compliance Reports Table
CREATE TABLE IF NOT EXISTS compliance_reports (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    framework VARCHAR(20) NOT NULL CHECK (framework IN ('fedramp', 'hipaa', 'gdpr', 'soc2')),
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    overall_status VARCHAR(30) NOT NULL CHECK (overall_status IN ('compliant', 'non_compliant', 'pending', 'unknown')),
    compliant_count INTEGER DEFAULT 0,
    non_compliant_count INTEGER DEFAULT 0,
    pending_count INTEGER DEFAULT 0,
    total_controls INTEGER DEFAULT 0,
    high_risk_count INTEGER DEFAULT 0,
    findings JSONB DEFAULT '[]'::jsonb,
    report_hash VARCHAR(255),
    signature TEXT,
    generated_at TIMESTAMPTZ DEFAULT NOW(),
    generated_by UUID REFERENCES users(id)
);

CREATE INDEX idx_compliance_reports_framework ON compliance_reports(framework);
CREATE INDEX idx_compliance_reports_period ON compliance_reports(period_start, period_end);

-- Extend audit_log table if not already present
ALTER TABLE audit_log
ADD COLUMN IF NOT EXISTS retention_date TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS compliance_tag VARCHAR(100);

CREATE INDEX idx_audit_log_retention ON audit_log(retention_date);
CREATE INDEX idx_audit_log_compliance_tag ON audit_log(compliance_tag);

-- Triggers
CREATE TRIGGER update_compliance_controls_updated_at BEFORE UPDATE ON compliance_controls
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_data_breaches_updated_at BEFORE UPDATE ON data_breaches
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_remediation_plans_updated_at BEFORE UPDATE ON remediation_plans
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to get pending compliance reviews
CREATE OR REPLACE FUNCTION get_pending_compliance_reviews(p_within_days INTEGER DEFAULT 30)
RETURNS TABLE (
    control_id UUID,
    framework VARCHAR(20),
    family VARCHAR(100),
    title VARCHAR(500),
    next_review TIMESTAMPTZ,
    days_until INTEGER
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        c.id,
        c.framework,
        c.family,
        c.title,
        c.next_review,
        EXTRACT(DAY FROM (c.next_review - NOW()))::INTEGER AS days_until
    FROM compliance_controls c
    WHERE c.next_review <= NOW() + (p_within_days || ' days')::INTERVAL
    ORDER BY c.next_review ASC;
END;
$$ LANGUAGE plpgsql;

-- Function to calculate compliance summary
CREATE OR REPLACE FUNCTION get_compliance_summary(p_framework VARCHAR DEFAULT NULL)
RETURNS TABLE (
    framework VARCHAR(20),
    compliant INTEGER,
    non_compliant INTEGER,
    pending INTEGER,
    total INTEGER,
    compliance_rate NUMERIC
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        COALESCE(p_framework, framework) AS framework,
        COUNT(*) FILTER (WHERE status = 'compliant')::INTEGER AS compliant,
        COUNT(*) FILTER (WHERE status = 'non_compliant')::INTEGER AS non_compliant,
        COUNT(*) FILTER (WHERE status = 'pending')::INTEGER AS pending,
        COUNT(*)::INTEGER AS total,
        CASE
            WHEN COUNT(*) > 0 THEN
                ROUND(100.0 * COUNT(*) FILTER (WHERE status = 'compliant')::NUMERIC / COUNT(*), 2)
            ELSE 0
        END AS compliance_rate
    FROM compliance_controls
    WHERE p_framework IS NULL OR framework = p_framework
    GROUP BY framework;
END;
$$ LANGUAGE plpgsql;

-- Comments for documentation
COMMENT ON TABLE compliance_controls IS 'Stores compliance control requirements for FedRAMP, HIPAA, GDPR, and SOC2';
COMMENT ON TABLE data_breaches IS 'Tracks data breach incidents for compliance reporting';
COMMENT ON TABLE remediation_plans IS 'Plans for addressing compliance gaps';
COMMENT ON TABLE compliance_findings IS 'Individual findings from compliance assessments';
COMMENT ON TABLE compliance_reports IS 'Generated compliance reports for audit purposes';


-- ========================================
-- Migration: 000033_create_m365_integration_tables.up.sql
-- ========================================
-- Microsoft 365 Integration Tables
-- Tables for Microsoft 365 (OneDrive, SharePoint) integration

-- Microsoft 365 Connections Table
-- Stores OAuth connections to Microsoft 365 for users
CREATE TABLE IF NOT EXISTS m365_connections (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_email VARCHAR(255) NOT NULL,
    tenant_id VARCHAR(255) NOT NULL DEFAULT 'common',
    access_token TEXT NOT NULL,
    refresh_token TEXT NOT NULL,
    token_expiry TIMESTAMPTZ NOT NULL,
    scopes JSONB DEFAULT '[]'::jsonb,
    connected_at TIMESTAMPTZ DEFAULT NOW(),
    last_used TIMESTAMPTZ,
    is_active BOOLEAN DEFAULT TRUE,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id)
);

CREATE INDEX idx_m365_connections_user ON m365_connections(user_id);
CREATE INDEX idx_m365_connections_tenant ON m365_connections(tenant_id);
CREATE INDEX idx_m365_connections_active ON m365_connections(is_active);
CREATE INDEX idx_m365_connections_last_used ON m365_connections(last_used);

-- Microsoft 365 Print Job Sources Table
-- Tracks print jobs submitted from Microsoft 365 sources
CREATE TABLE IF NOT EXISTS m365_print_job_sources (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_id UUID REFERENCES print_jobs(id) ON DELETE SET NULL,
    source_id VARCHAR(255) NOT NULL,
    source_type VARCHAR(50) NOT NULL CHECK (source_type IN ('onedrive', 'sharepoint', 'outlook', 'teams')),
    document_id VARCHAR(500) NOT NULL,
    document_name VARCHAR(500) NOT NULL,
    document_url TEXT NOT NULL,
    file_size BIGINT,
    mime_type VARCHAR(255),
    user_id UUID REFERENCES users(id),
    user_email VARCHAR(255),
    downloaded_at TIMESTAMPTZ DEFAULT NOW(),
    downloaded_path TEXT,
    file_hash VARCHAR(255),
    download_status VARCHAR(50) DEFAULT 'pending' CHECK (download_status IN ('pending', 'downloading', 'completed', 'failed')),
    error_message TEXT,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_m365_print_sources_job ON m365_print_job_sources(job_id);
CREATE INDEX idx_m365_print_sources_type ON m365_print_job_sources(source_type);
CREATE INDEX idx_m365_print_sources_user ON m365_print_job_sources(user_id);
CREATE INDEX idx_m365_print_sources_status ON m365_print_job_sources(download_status);

-- SharePoint Sites Cache Table
-- Caches SharePoint site information for quick access
CREATE TABLE IF NOT EXISTS m365_sharepoint_sites (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    url TEXT NOT NULL,
    web_url TEXT NOT NULL,
    tenant_id VARCHAR(255) NOT NULL,
    owner_id UUID REFERENCES users(id),
    drive_id VARCHAR(255),
    last_synced_at TIMESTAMPTZ DEFAULT NOW(),
    isAccessible BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_m365_sp_sites_tenant ON m365_sharepoint_sites(tenant_id);
CREATE INDEX idx_m365_sp_sites_owner ON m365_sharepoint_sites(owner_id);
CREATE INDEX idx_m365_sp_sites_synced ON m365_sharepoint_sites(last_synced_at);

-- OneDrive Drives Cache Table
-- Caches OneDrive drive information
CREATE TABLE IF NOT EXISTS m365_onedrive_drives (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    drive_id VARCHAR(255) NOT NULL UNIQUE,
    user_id UUID NOT NULL REFERENCES users(id),
    user_email VARCHAR(255) NOT NULL,
    drive_type VARCHAR(50) NOT NULL, -- 'personal', 'business', 'documentLibrary'
    name VARCHAR(255),
    quota_total BIGINT,
    quota_used BIGINT,
    quota_remaining BIGINT,
    last_synced_at TIMESTAMPTZ DEFAULT NOW(),
    isAccessible BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_m365_onedrive_user ON m365_onedrive_drives(user_id);
CREATE INDEX idx_m365_onedrive_synced ON m365_onedrive_drives(last_synced_at);

-- Microsoft 365 Document Cache Table
-- Caches document metadata for quick listing
CREATE TABLE IF NOT EXISTS m365_document_cache (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    document_id VARCHAR(500) NOT NULL,
    source_type VARCHAR(50) NOT NULL CHECK (source_type IN ('onedrive', 'sharepoint', 'outlook', 'teams')),
    source_id VARCHAR(255) NOT NULL, -- drive_id or site_id
    name VARCHAR(500) NOT NULL,
    path TEXT,
    web_url TEXT,
    download_url TEXT,
    mime_type VARCHAR(255),
    size BIGINT,
    created_by VARCHAR(255),
    created_at TIMESTAMPTZ,
    modified_at TIMESTAMPTZ,
    cached_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ DEFAULT NOW() + INTERVAL '1 hour',
    user_id UUID REFERENCES users(id),
    metadata JSONB DEFAULT '{}'::jsonb
);

CREATE INDEX idx_m365_doc_cache_document ON m365_document_cache(document_id);
CREATE INDEX idx_m365_doc_cache_source ON m365_document_cache(source_type, source_id);
CREATE INDEX idx_m365_doc_cache_user ON m365_document_cache(user_id);
CREATE INDEX idx_m365_doc_cache_expires ON m365_document_cache(expires_at);

-- Microsoft 365 Sync History Table
-- Tracks sync operations for audit and troubleshooting
CREATE TABLE IF NOT EXISTS m365_sync_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    operation VARCHAR(50) NOT NULL, -- 'list_files', 'download_file', 'get_site_info', etc.
    source_type VARCHAR(50),
    source_id VARCHAR(255),
    user_id UUID REFERENCES users(id),
    status VARCHAR(50) NOT NULL CHECK (status IN ('started', 'completed', 'failed', 'cancelled')),
    started_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    items_processed INTEGER DEFAULT 0,
    error_message TEXT,
    metadata JSONB DEFAULT '{}'::jsonb
);

CREATE INDEX idx_m365_sync_history_user ON m365_sync_history(user_id);
CREATE INDEX idx_m365_sync_history_status ON m365_sync_history(status);
CREATE INDEX idx_m365_sync_history_started ON m365_sync_history(started_at);

-- Triggers
CREATE TRIGGER update_m365_connections_updated_at BEFORE UPDATE ON m365_connections
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_m365_sp_sites_updated_at BEFORE UPDATE ON m365_sharepoint_sites
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_m365_onedrive_updated_at BEFORE UPDATE ON m365_onedrive_drives
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to clean expired document cache entries
CREATE OR REPLACE FUNCTION cleanup_expired_m365_cache()
RETURNS INTEGER AS $$
DECLARE
    v_count INTEGER;
BEGIN
    DELETE FROM m365_document_cache
    WHERE expires_at < NOW();

    GET DIAGNOSTICS v_count = ROW_COUNT;
    RETURN v_count;
END;
$$ LANGUAGE plpgsql;

-- Function to get user's M365 connection
CREATE OR REPLACE FUNCTION get_user_m365_connection(p_user_id UUID)
RETURNS TABLE (
    id UUID,
    user_email VARCHAR(255),
    tenant_id VARCHAR(255),
    access_token TEXT,
    refresh_token TEXT,
    token_expiry TIMESTAMPTZ,
    is_active BOOLEAN
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        c.id,
        c.user_email,
        c.tenant_id,
        c.access_token,
        c.refresh_token,
        c.token_expiry,
        c.is_active
    FROM m365_connections c
    WHERE c.user_id = p_user_id
      AND c.is_active = TRUE
    LIMIT 1;
END;
$$ LANGUAGE plpgsql;

-- Function to update last used timestamp
CREATE OR REPLACE FUNCTION update_m365_last_used(p_connection_id UUID)
RETURNS VOID AS $$
BEGIN
    UPDATE m365_connections
    SET last_used = NOW(),
        updated_at = NOW()
    WHERE id = p_connection_id;
END;
$$ LANGUAGE plpgsql;

-- Comments for documentation
COMMENT ON TABLE m365_connections IS 'Stores OAuth connections to Microsoft 365 for users';
COMMENT ON TABLE m365_print_job_sources IS 'Tracks print jobs submitted from Microsoft 365 sources';
COMMENT ON TABLE m365_sharepoint_sites IS 'Caches SharePoint site information';
COMMENT ON TABLE m365_onedrive_drives IS 'Caches OneDrive drive information';
COMMENT ON TABLE m365_document_cache IS 'Caches Microsoft 365 document metadata';
COMMENT ON TABLE m365_sync_history IS 'Tracks Microsoft 365 sync operations for audit';


-- ========================================
-- Migration: 000034_add_multi_tenant_support.up.sql
-- ========================================
-- Migration: Add Multi-Tenant Support
-- This migration adds tables and RLS policies for multi-tenancy support

-- Add tenant_id column to existing tables (if not exists)
ALTER TABLE printers ADD COLUMN IF NOT EXISTS tenant_id uuid REFERENCES organizations(id) ON DELETE SET NULL;
ALTER TABLE print_jobs ADD COLUMN IF NOT EXISTS tenant_id uuid REFERENCES organizations(id) ON DELETE SET NULL;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS tenant_id uuid REFERENCES organizations(id) ON DELETE SET NULL;

-- Create quota_configs table
CREATE TABLE IF NOT EXISTS quota_configs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    max_printers integer NOT NULL DEFAULT 100,
    max_storage_gb integer NOT NULL DEFAULT 100,
    max_jobs_per_month integer NOT NULL DEFAULT 10000,
    max_users integer NOT NULL DEFAULT 50,
    alert_threshold integer NOT NULL DEFAULT 80,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id)
);

-- Create index on tenant_id for quota_configs
CREATE INDEX IF NOT EXISTS idx_quota_configs_tenant_id ON quota_configs(tenant_id);

-- Create quota_usage table
CREATE TABLE IF NOT EXISTS quota_usage (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    printers_count integer NOT NULL DEFAULT 0,
    storage_used_gb bigint NOT NULL DEFAULT 0,  -- Stored in bytes
    jobs_this_month integer NOT NULL DEFAULT 0,
    users_count integer NOT NULL DEFAULT 0,
    month timestamptz NOT NULL DEFAULT DATE_TRUNC('month', NOW()),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, month)
);

-- Create index on tenant_id and month for quota_usage
CREATE INDEX IF NOT EXISTS idx_quota_usage_tenant_month ON quota_usage(tenant_id, month DESC);

-- Create organization_users table
CREATE TABLE IF NOT EXISTS organization_users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role text NOT NULL DEFAULT 'member',
    settings jsonb DEFAULT '{}'::jsonb,
    joined_at timestamptz NOT NULL DEFAULT NOW(),
    invited_by uuid REFERENCES users(id) ON DELETE SET NULL,
    deleted_at timestamptz,
    UNIQUE(organization_id, user_id)
);

-- Create indexes for organization_users
CREATE INDEX IF NOT EXISTS idx_organization_users_org_id ON organization_users(organization_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_organization_users_user_id ON organization_users(user_id) WHERE deleted_at IS NULL;

-- Add check constraint for role
ALTER TABLE organization_users DROP CONSTRAINT IF EXISTS organization_users_role_check;
ALTER TABLE organization_users ADD CONSTRAINT organization_users_role_check
    CHECK (role IN ('owner', 'admin', 'member', 'viewer', 'billing'));

-- Create platform_admin role (if not exists) - must be created before policies that reference it
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'platform_admin') THEN
        CREATE ROLE platform_admin;
    END IF;
END
$$;

-- Grant necessary permissions
GRANT USAGE ON SCHEMA public TO platform_admin;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO platform_admin;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO platform_admin;

-- Enable Row Level Security on tables
ALTER TABLE printers ENABLE ROW LEVEL SECURITY;
ALTER TABLE print_jobs ENABLE ROW LEVEL SECURITY;
ALTER TABLE documents ENABLE ROW LEVEL SECURITY;
ALTER TABLE quota_configs ENABLE ROW LEVEL SECURITY;
ALTER TABLE quota_usage ENABLE ROW LEVEL SECURITY;
ALTER TABLE organization_users ENABLE ROW LEVEL SECURITY;

-- Create RLS policies for printers
DROP POLICY IF EXISTS printers_tenant_isolation ON printers;
CREATE POLICY printers_tenant_isolation ON printers
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS printers_platform_admin ON printers;
CREATE POLICY printers_platform_admin ON printers
    FOR ALL
    TO platform_admin
    USING (true)
    WITH CHECK (true);

-- Create RLS policies for print_jobs
DROP POLICY IF EXISTS print_jobs_tenant_isolation ON print_jobs;
CREATE POLICY print_jobs_tenant_isolation ON print_jobs
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS print_jobs_platform_admin ON print_jobs;
CREATE POLICY print_jobs_platform_admin ON print_jobs
    FOR ALL
    TO platform_admin
    USING (true)
    WITH CHECK (true);

-- Create RLS policies for documents
DROP POLICY IF EXISTS documents_tenant_isolation ON documents;
CREATE POLICY documents_tenant_isolation ON documents
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS documents_platform_admin ON documents;
CREATE POLICY documents_platform_admin ON documents
    FOR ALL
    TO platform_admin
    USING (true)
    WITH CHECK (true);

-- Create RLS policies for quota_configs
DROP POLICY IF EXISTS quota_configs_tenant_isolation ON quota_configs;
CREATE POLICY quota_configs_tenant_isolation ON quota_configs
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS quota_configs_platform_admin ON quota_configs;
CREATE POLICY quota_configs_platform_admin ON quota_configs
    FOR ALL
    TO platform_admin
    USING (true)
    WITH CHECK (true);

-- Create RLS policies for quota_usage
DROP POLICY IF EXISTS quota_usage_tenant_isolation ON quota_usage;
CREATE POLICY quota_usage_tenant_isolation ON quota_usage
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS quota_usage_platform_admin ON quota_usage;
CREATE POLICY quota_usage_platform_admin ON quota_usage
    FOR ALL
    TO platform_admin
    USING (true)
    WITH CHECK (true);

-- Create RLS policies for organization_users
DROP POLICY IF EXISTS organization_users_tenant_isolation ON organization_users;
CREATE POLICY organization_users_tenant_isolation ON organization_users
    FOR ALL
    USING (organization_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (organization_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS organization_users_platform_admin ON organization_users;
CREATE POLICY organization_users_platform_admin ON organization_users
    FOR ALL
    TO platform_admin
    USING (true)
    WITH CHECK (true);


-- ========================================
-- Migration: 000035_create_observability_tables.up.sql
-- ========================================
-- Migration: Create observability tables
-- These tables support the OpenPrint observability stack for metrics and alerting.

-- Observability metrics table for storing historical metric data
CREATE TABLE IF NOT EXISTS observability_metrics (
    id BIGSERIAL PRIMARY KEY,
    service_name VARCHAR(100) NOT NULL,
    metric_name VARCHAR(255) NOT NULL,
    metric_value DECIMAL(20,6) NOT NULL,
    labels JSONB,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for efficient time-series queries
CREATE INDEX idx_observability_metrics_service_time ON observability_metrics(service_name, recorded_at DESC);
CREATE INDEX idx_observability_metrics_name_time ON observability_metrics(metric_name, recorded_at DESC);
CREATE INDEX idx_observability_metrics_labels_gin ON observability_metrics USING GIN(labels);

-- Alert history table for tracking fired and resolved alerts
CREATE TABLE IF NOT EXISTS alert_history (
    id BIGSERIAL PRIMARY KEY,
    alert_name VARCHAR(255) NOT NULL,
    alert_severity VARCHAR(50) NOT NULL CHECK (alert_severity IN ('critical', 'warning', 'info')),
    service_name VARCHAR(100),
    message TEXT,
    fired_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,
    status VARCHAR(50) NOT NULL DEFAULT 'firing' CHECK (status IN ('firing', 'resolved')),
    labels JSONB,
    annotations JSONB
);

-- Index for alert history queries
CREATE INDEX idx_alert_history_fired ON alert_history(fired_at DESC);
CREATE INDEX idx_alert_history_status ON alert_history(status, fired_at DESC);
CREATE INDEX idx_alert_history_service ON alert_history(service_name, fired_at DESC);

-- Quota usage table for tracking user and organization quotas
CREATE TABLE IF NOT EXISTS quota_usage_tracking (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL,
    organization_id UUID NOT NULL,
    quota_type VARCHAR(50) NOT NULL,
    usage_count BIGINT NOT NULL DEFAULT 0,
    quota_limit BIGINT NOT NULL,
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, quota_type, period_start)
);

-- Index for quota queries
CREATE INDEX IF NOT EXISTS idx_quota_usage_tracking_user ON quota_usage_tracking(user_id, period_start DESC);
CREATE INDEX IF NOT EXISTS idx_quota_usage_tracking_org ON quota_usage_tracking(organization_id, period_start DESC);
CREATE INDEX IF NOT EXISTS idx_quota_usage_tracking_period ON quota_usage_tracking(period_start, period_end);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger for automatic updated_at
DROP TRIGGER IF EXISTS update_quota_usage_tracking_updated_at ON quota_usage_tracking;
CREATE TRIGGER update_quota_usage_tracking_updated_at
    BEFORE UPDATE ON quota_usage_tracking
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Audit log enrichment table for additional observability data
CREATE TABLE IF NOT EXISTS audit_log_enrichment (
    audit_log_id BIGINT PRIMARY KEY,
    trace_id VARCHAR(64),
    span_id VARCHAR(64),
    parent_span_id VARCHAR(64),
    duration_ms INTEGER,
    client_ip INET,
    user_agent TEXT,
    request_size_bytes BIGINT,
    response_size_bytes BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for trace lookups
CREATE INDEX idx_audit_enrichment_trace ON audit_log_enrichment(trace_id);
CREATE INDEX idx_audit_enrichment_created ON audit_log_enrichment(created_at DESC);

-- Service performance metrics table (daily aggregates)
CREATE TABLE IF NOT EXISTS service_performance_daily (
    id BIGSERIAL PRIMARY KEY,
    service_name VARCHAR(100) NOT NULL,
    date DATE NOT NULL,
    total_requests BIGINT NOT NULL DEFAULT 0,
    successful_requests BIGINT NOT NULL DEFAULT 0,
    failed_requests BIGINT NOT NULL DEFAULT 0,
    avg_duration_ms DECIMAL(10,2),
    p95_duration_ms DECIMAL(10,2),
    p99_duration_ms DECIMAL(10,2),
    total_data_in_bytes BIGINT NOT NULL DEFAULT 0,
    total_data_out_bytes BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(service_name, date)
);

-- Index for daily performance queries
CREATE INDEX idx_service_perf_service_date ON service_performance_daily(service_name, date DESC);

-- SLA compliance tracking table
CREATE TABLE IF NOT EXISTS sla_compliance (
    id BIGSERIAL PRIMARY KEY,
    service_name VARCHAR(100) NOT NULL,
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    availability_target DECIMAL(5,4) NOT NULL DEFAULT 0.9990, -- 99.9%
    availability_actual DECIMAL(5,4),
    uptime_target_seconds BIGINT NOT NULL,
    uptime_actual_seconds BIGINT NOT NULL,
    downtime_seconds BIGINT NOT NULL DEFAULT 0,
    sla_met BOOLEAN NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(service_name, period_start)
);

-- Index for SLA queries
CREATE INDEX idx_sla_service_period ON sla_compliance(service_name, period_start DESC);

-- Comments for documentation
COMMENT ON TABLE observability_metrics IS 'Stores metric samples for long-term analysis and backup';
COMMENT ON TABLE alert_history IS 'History of all fired and resolved alerts from Prometheus/AlertManager';
COMMENT ON TABLE quota_usage_tracking IS 'Tracks usage-based quotas for users and organizations';
COMMENT ON TABLE audit_log_enrichment IS 'Enriches audit logs with trace context and performance data';
COMMENT ON TABLE service_performance_daily IS 'Daily aggregated performance metrics for each service';
COMMENT ON TABLE sla_compliance IS 'Tracks SLA compliance for service availability guarantees';


-- ========================================
-- Migration: 000036_add_environmental_metrics.up.sql
-- ========================================
-- Migration: 009_add_environmental_metrics
-- Tracks environmental impact of printing for sustainability reporting

CREATE TABLE IF NOT EXISTS environmental_metrics (
    id BIGSERIAL PRIMARY KEY,
    organization_id UUID NOT NULL,
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    total_pages_printed BIGINT DEFAULT 0,
    pages_saved_duplex BIGINT DEFAULT 0,
    pages_saved_nprint BIGINT DEFAULT 0,
    estimated_paper_kg DECIMAL(10, 4) DEFAULT 0,
    estimated_co2_kg DECIMAL(10, 4) DEFAULT 0,
    estimated_trees_saved DECIMAL(10, 4) DEFAULT 0,
    estimated_water_liters DECIMAL(10, 4) DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(organization_id, period_start)
);

CREATE INDEX idx_environmental_metrics_org ON environmental_metrics(organization_id, period_start DESC);

-- Function to calculate carbon footprint for a period
CREATE OR REPLACE FUNCTION calculate_carbon_footprint(
    p_organization_id UUID,
    p_start_date DATE,
    p_end_date DATE
) RETURNS JSONB AS $$
DECLARE
    v_total_pages BIGINT;
    v_duplex_pages BIGINT;
    v_paper_kg DECIMAL;
    v_co2_kg DECIMAL;
    v_trees DECIMAL;
BEGIN
    SELECT COALESCE(SUM(total_pages), 0), COALESCE(SUM(duplex_pages), 0)
    INTO v_total_pages, v_duplex_pages
    FROM print_usage_by_day
    WHERE organization_id = p_organization_id
      AND date BETWEEN p_start_date AND p_end_date;

    -- Estimates: 1 page = 5g paper, 1 kg paper = 1.2 kg CO2, 1 tree = 8333 pages
    v_paper_kg := (v_total_pages * 5.0) / 1000.0;
    v_co2_kg := v_paper_kg * 1.2;
    v_trees := v_total_pages / 8333.0;

    INSERT INTO environmental_metrics (
        organization_id, period_start, period_end,
        total_pages_printed, pages_saved_duplex,
        estimated_paper_kg, estimated_co2_kg, estimated_trees_saved
    ) VALUES (
        p_organization_id, p_start_date, p_end_date,
        v_total_pages, v_duplex_pages,
        v_paper_kg, v_co2_kg, v_trees
    ) ON CONFLICT (organization_id, period_start)
    DO UPDATE SET
        total_pages_printed = EXCLUDED.total_pages_printed,
        pages_saved_duplex = EXCLUDED.pages_saved_duplex,
        estimated_paper_kg = EXCLUDED.estimated_paper_kg,
        estimated_co2_kg = EXCLUDED.estimated_co2_kg,
        estimated_trees_saved = EXCLUDED.estimated_trees_saved,
        updated_at = NOW();

    RETURN jsonb_build_object(
        'total_pages', v_total_pages,
        'duplex_saved', v_duplex_pages,
        'paper_kg', v_paper_kg,
        'co2_kg', v_co2_kg,
        'trees_saved', v_trees
    );
END;
$$ LANGUAGE plpgsql;


-- ========================================
-- Migration: 000037_create_quotas_table.up.sql
-- ========================================
-- Migration: 001_create_quotas_table
-- Note: The main observability tables are in 001_create_observability_tables.up.sql
-- This migration adds quota usage tracking if not already present

-- quota_usage_tracking is created in 001_create_observability_tables.up.sql
-- This file exists to pair with 001_create_quotas_table.down.sql
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'quota_usage_tracking') THEN
        CREATE TABLE quota_usage_tracking (
            id BIGSERIAL PRIMARY KEY,
            user_id UUID NOT NULL,
            organization_id UUID NOT NULL,
            quota_type VARCHAR(50) NOT NULL,
            usage_count BIGINT NOT NULL DEFAULT 0,
            quota_limit BIGINT NOT NULL,
            period_start TIMESTAMPTZ NOT NULL,
            period_end TIMESTAMPTZ NOT NULL,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
            updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
            UNIQUE(user_id, quota_type, period_start)
        );
        CREATE INDEX idx_quota_usage_tracking_user ON quota_usage_tracking(user_id, period_start DESC);
        CREATE INDEX idx_quota_usage_tracking_org ON quota_usage_tracking(organization_id, period_start DESC);
    END IF;
END $$;


-- ========================================
-- Migration: 000038_create_quota_transactions_table.up.sql
-- ========================================
-- Migration: 002_create_quota_transactions_table
-- Tracks individual quota transactions for audit and analytics

CREATE TABLE IF NOT EXISTS quota_transactions (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL,
    organization_id UUID NOT NULL,
    quota_type VARCHAR(50) NOT NULL,
    change_amount INTEGER NOT NULL,
    previous_usage INTEGER NOT NULL,
    new_usage INTEGER NOT NULL,
    reason VARCHAR(255),
    job_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_quota_transactions_user ON quota_transactions(user_id, created_at DESC);
CREATE INDEX idx_quota_transactions_org ON quota_transactions(organization_id, created_at DESC);
CREATE INDEX idx_quota_transactions_job ON quota_transactions(job_id);


-- ========================================
-- Migration: 000039_create_costs_table.up.sql
-- ========================================
-- Migration: 003_create_costs_table
-- Tracks real-time cost accumulation per organization

CREATE TABLE IF NOT EXISTS cost_tracking (
    id BIGSERIAL PRIMARY KEY,
    organization_id UUID NOT NULL,
    user_id UUID,
    job_id UUID,
    cost_type VARCHAR(50) NOT NULL,
    amount DECIMAL(10, 4) NOT NULL DEFAULT 0,
    currency VARCHAR(3) DEFAULT 'USD',
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_cost_tracking_org ON cost_tracking(organization_id, period_start DESC);
CREATE INDEX idx_cost_tracking_user ON cost_tracking(user_id, period_start DESC);
CREATE INDEX idx_cost_tracking_job ON cost_tracking(job_id);


-- ========================================
-- Migration: 000040_create_held_jobs_table.up.sql
-- ========================================
-- Migration: 004_create_held_jobs_table
-- Stores jobs held for secure release at a printer

CREATE TABLE IF NOT EXISTS held_jobs (
    id BIGSERIAL PRIMARY KEY,
    job_id UUID NOT NULL,
    user_id UUID NOT NULL,
    printer_id UUID,
    hold_reason VARCHAR(100) NOT NULL DEFAULT 'secure_release',
    release_code VARCHAR(20),
    held_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    released_at TIMESTAMPTZ,
    expired_at TIMESTAMPTZ,
    ttl_seconds INTEGER DEFAULT 3600,
    status VARCHAR(20) NOT NULL DEFAULT 'held' CHECK (status IN ('held', 'released', 'expired', 'cancelled'))
);

CREATE INDEX idx_held_jobs_job ON held_jobs(job_id);
CREATE INDEX idx_held_jobs_user ON held_jobs(user_id);
CREATE INDEX idx_held_jobs_printer ON held_jobs(printer_id);
CREATE INDEX idx_held_jobs_status ON held_jobs(status) WHERE status = 'held';
CREATE INDEX idx_held_jobs_release_code ON held_jobs(release_code) WHERE release_code IS NOT NULL;


-- ========================================
-- Migration: 000041_create_watermark_templates_table.up.sql
-- ========================================
-- Migration: 005_create_watermark_templates_table
-- Audit trail for watermark application

CREATE TABLE IF NOT EXISTS watermark_audit_log (
    id BIGSERIAL PRIMARY KEY,
    template_id UUID,
    job_id UUID,
    user_id UUID,
    organization_id UUID,
    action VARCHAR(50) NOT NULL,
    details JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_watermark_audit_template ON watermark_audit_log(template_id);
CREATE INDEX idx_watermark_audit_job ON watermark_audit_log(job_id);
CREATE INDEX idx_watermark_audit_org ON watermark_audit_log(organization_id, created_at DESC);


-- ========================================
-- Migration: 000042_create_report_schedules_table.up.sql
-- ========================================
-- Migration: 006_create_report_schedules_table
-- Tracks report delivery history

CREATE TABLE IF NOT EXISTS report_schedules (
    id BIGSERIAL PRIMARY KEY,
    organization_id UUID NOT NULL,
    report_type VARCHAR(50) NOT NULL,
    schedule_cron VARCHAR(100) NOT NULL,
    recipients JSONB,
    format VARCHAR(10) DEFAULT 'pdf',
    is_active BOOLEAN DEFAULT true,
    last_run_at TIMESTAMPTZ,
    next_run_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS report_deliveries (
    id BIGSERIAL PRIMARY KEY,
    schedule_id BIGINT REFERENCES report_schedules(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'generating', 'delivered', 'failed')),
    recipient VARCHAR(255),
    file_path TEXT,
    file_size_bytes BIGINT,
    error_message TEXT,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_report_schedules_org ON report_schedules(organization_id);
CREATE INDEX idx_report_schedules_active ON report_schedules(is_active, next_run_at) WHERE is_active = true;
CREATE INDEX idx_report_deliveries_schedule ON report_deliveries(schedule_id, created_at DESC);
CREATE INDEX idx_report_deliveries_status ON report_deliveries(status) WHERE status IN ('pending', 'generating');


-- ========================================
-- Migration: 000043_create_mobile_devices_table.up.sql
-- ========================================
-- Migration: 007_create_mobile_devices_table
-- Tracks print jobs submitted from mobile devices

CREATE TABLE IF NOT EXISTS mobile_print_jobs (
    id BIGSERIAL PRIMARY KEY,
    device_id UUID NOT NULL,
    job_id UUID NOT NULL,
    source_app VARCHAR(100),
    document_name VARCHAR(255),
    submitted_via VARCHAR(50) DEFAULT 'mobile_app',
    location_lat DECIMAL(10, 7),
    location_lon DECIMAL(10, 7),
    nearest_printer_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_mobile_print_jobs_device ON mobile_print_jobs(device_id, created_at DESC);
CREATE INDEX idx_mobile_print_jobs_job ON mobile_print_jobs(job_id);


-- ========================================
-- Migration: 000044_create_api_keys_table.up.sql
-- ========================================
-- Migration: 008_create_api_keys_table
-- Fine-grained API key permissions for the developer portal

CREATE TABLE IF NOT EXISTS api_key_permissions (
    id BIGSERIAL PRIMARY KEY,
    api_key_id UUID NOT NULL,
    resource VARCHAR(100) NOT NULL,
    actions TEXT[] NOT NULL DEFAULT '{}',
    conditions JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_api_key_permissions_key ON api_key_permissions(api_key_id);
CREATE INDEX idx_api_key_permissions_resource ON api_key_permissions(resource);



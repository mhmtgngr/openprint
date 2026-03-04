-- OpenPrint Cloud - Rate Limiting Tables
-- Migration: 000035_create_rate_limit_tables.up.sql

-- Rate limit policies table
CREATE TABLE IF NOT EXISTS rate_limit_policies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    priority INTEGER DEFAULT 0,
    scope VARCHAR(50) NOT NULL CHECK (scope IN ('global', 'endpoint', 'ip', 'user', 'api_key', 'organization')),
    identifier VARCHAR(255) NOT NULL DEFAULT '*',
    methods JSONB DEFAULT '[]'::jsonb,
    path_pattern VARCHAR(500),
    request_limit BIGINT NOT NULL,
    "window" INTEGER NOT NULL, -- seconds (quoted because "window" is a reserved keyword in PostgreSQL 16)
    burst_limit BIGINT DEFAULT 0,
    burst_duration INTEGER DEFAULT 0, -- seconds
    enable_queue BOOLEAN DEFAULT false,
    max_queue_size INTEGER DEFAULT 100,
    circuit_breaker_threshold INTEGER DEFAULT 0,
    circuit_breaker_timeout INTEGER DEFAULT 0, -- seconds
    severity VARCHAR(20) DEFAULT 'medium' CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    action VARCHAR(50) DEFAULT 'reject' CHECK (action IN ('reject', 'throttle', 'queue', 'alert_only')),
    throttle_rate DOUBLE PRECISION DEFAULT 0.0,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_rate_limit_policies_scope ON rate_limit_policies(scope);
CREATE INDEX idx_rate_limit_policies_identifier ON rate_limit_policies(identifier);
CREATE INDEX idx_rate_limit_policies_active ON rate_limit_policies(is_active);
CREATE INDEX idx_rate_limit_policies_priority ON rate_limit_policies(priority DESC);

-- Rate limit violations table
CREATE TABLE IF NOT EXISTS rate_limit_violations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    policy_id UUID REFERENCES rate_limit_policies(id) ON DELETE SET NULL,
    policy_name VARCHAR(255),
    identifier VARCHAR(255) NOT NULL,
    identifier_type VARCHAR(50) NOT NULL,
    path VARCHAR(500),
    method VARCHAR(10),
    current BIGINT NOT NULL,
    "limit" BIGINT NOT NULL, -- quoted because "limit" is a reserved keyword in PostgreSQL
    severity VARCHAR(20),
    occurred_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_rate_limit_violations_identifier ON rate_limit_violations(identifier);
CREATE INDEX idx_rate_limit_violations_policy_id ON rate_limit_violations(policy_id);
CREATE INDEX idx_rate_limit_violations_occurred_at ON rate_limit_violations(occurred_at DESC);
CREATE INDEX idx_rate_limit_violations_severity ON rate_limit_violations(severity);

-- Trusted clients table
CREATE TABLE IF NOT EXISTS trusted_clients (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    api_key VARCHAR(255) UNIQUE,
    ip_whitelist JSONB DEFAULT '[]'::jsonb,
    description TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    last_used_at TIMESTAMPTZ
);

CREATE INDEX idx_trusted_clients_api_key ON trusted_clients(api_key);
CREATE INDEX idx_trusted_clients_active ON trusted_clients(is_active);

-- Circuit breaker state table (optional - mainly stored in Redis, this is for persistence)
CREATE TABLE IF NOT EXISTS circuit_breaker_states (
    path VARCHAR(500) PRIMARY KEY,
    state VARCHAR(20) NOT NULL CHECK (state IN ('closed', 'open', 'half_open')),
    failure_count INTEGER DEFAULT 0,
    last_failure_at TIMESTAMPTZ,
    opened_at TIMESTAMPTZ,
    closes_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_circuit_breaker_states_state ON circuit_breaker_states(state);

-- Rate limit statistics table (for historical analysis)
CREATE TABLE IF NOT EXISTS rate_limit_stats (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    policy_id UUID REFERENCES rate_limit_policies(id) ON DELETE SET NULL,
    identifier VARCHAR(255),
    identifier_type VARCHAR(50),
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    requests_allowed BIGINT DEFAULT 0,
    requests_denied BIGINT DEFAULT 0,
    peak_usage BIGINT DEFAULT 0,
    avg_usage DOUBLE PRECISION DEFAULT 0.0,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_rate_limit_stats_policy_id ON rate_limit_stats(policy_id);
CREATE INDEX idx_rate_limit_stats_period ON rate_limit_stats(period_start DESC);

-- Trigger for updated_at on rate_limit_policies
CREATE TRIGGER update_rate_limit_policies_updated_at
    BEFORE UPDATE ON rate_limit_policies
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Trigger for updated_at on trusted_clients
CREATE TRIGGER update_trusted_clients_updated_at
    BEFORE UPDATE ON trusted_clients
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to clean up old violations
CREATE OR REPLACE FUNCTION cleanup_old_violations(days_to_keep INTEGER DEFAULT 30)
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM rate_limit_violations
    WHERE occurred_at < NOW() - (days_to_keep || ' days')::INTERVAL;

    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Function to clean up old stats
CREATE OR REPLACE FUNCTION cleanup_old_stats(days_to_keep INTEGER DEFAULT 90)
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM rate_limit_stats
    WHERE period_start < NOW() - (days_to_keep || ' days')::INTERVAL;

    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Insert default policies (use quoted column names)
INSERT INTO rate_limit_policies (name, description, priority, scope, identifier, request_limit, "window", burst_limit, burst_duration, severity, action) VALUES
    ('Global Default', 'Default rate limit for all requests', 0, 'global', '*', 10000, 3600, 100, 60, 'low', 'reject'),
    ('IP Default', 'Default rate limit per IP address', 10, 'ip', '*', 100, 60, 20, 10, 'medium', 'reject'),
    ('User Default', 'Default rate limit per user', 20, 'user', '*', 1000, 3600, 100, 60, 'medium', 'reject'),
    ('API Key Default', 'Default rate limit per API key', 30, 'api_key', '*', 5000, 3600, 200, 60, 'low', 'reject'),
    ('Auth Endpoint Strict', 'Strict rate limit for authentication endpoints', 100, 'endpoint', '*', 10, 60, 3, 10, 'high', 'reject')
ON CONFLICT DO NOTHING;

-- Update policies to set specific path pattern for auth endpoint
UPDATE rate_limit_policies SET path_pattern = '/api/auth/%' WHERE name = 'Auth Endpoint Strict';

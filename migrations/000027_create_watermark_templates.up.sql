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

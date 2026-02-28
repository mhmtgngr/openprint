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

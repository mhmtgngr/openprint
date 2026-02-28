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

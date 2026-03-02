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

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

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

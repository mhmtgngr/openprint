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

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

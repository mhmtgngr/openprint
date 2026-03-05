-- Migration: 001_create_quotas_table
-- Down migration

DROP TABLE IF EXISTS quota_usage_tracking CASCADE;

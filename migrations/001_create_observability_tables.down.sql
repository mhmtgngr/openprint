-- Rollback migration: Drop observability tables

-- Drop in reverse order of creation
DROP TABLE IF EXISTS sla_compliance CASCADE;
DROP TABLE IF EXISTS service_performance_daily CASCADE;
DROP TABLE IF EXISTS audit_log_enrichment CASCADE;
DROP TABLE IF EXISTS quota_usage CASCADE;
DROP TABLE IF EXISTS alert_history CASCADE;
DROP TABLE IF EXISTS observability_metrics CASCADE;

-- Drop the trigger function
DROP FUNCTION IF EXISTS update_updated_at_column CASCADE;

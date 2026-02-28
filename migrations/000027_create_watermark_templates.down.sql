-- Down migration for watermark templates and related tables

DROP TABLE IF EXISTS audit_logs CASCADE;
DROP TABLE IF EXISTS webhooks CASCADE;
DROP TABLE IF EXISTS api_usage_logs CASCADE;
DROP TABLE IF EXISTS api_keys CASCADE;
DROP TABLE IF EXISTS push_notifications CASCADE;
DROP TABLE IF EXISTS mobile_devices CASCADE;
DROP TABLE IF EXISTS scheduled_reports CASCADE;
DROP TABLE IF EXISTS budget_allocations CASCADE;
DROP TABLE IF EXISTS watermark_templates CASCADE;

DROP FUNCTION IF EXISTS check_api_key_scope(VARCHAR, VARCHAR);
DROP FUNCTION IF EXISTS record_api_usage(UUID, UUID, VARCHAR, TEXT, INTEGER, INTEGER);
DROP FUNCTION IF EXISTS get_budget_status(UUID, VARCHAR);
DROP FUNCTION IF EXISTS deduct_budget(UUID, VARCHAR, DECIMAL);

-- Remove cost_center column from users (optional)
-- ALTER TABLE users DROP COLUMN IF EXISTS cost_center;

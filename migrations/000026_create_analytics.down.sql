-- Rollback analytics migration

DROP FUNCTION IF EXISTS get_top_users CASCADE;
DROP FUNCTION IF EXISTS get_top_printers CASCADE;
DROP FUNCTION IF EXISTS aggregate_daily_usage CASCADE;

DROP TABLE IF EXISTS cost_center_reports;
DROP TABLE IF EXISTS user_activity_summary;
DROP TABLE IF EXISTS printer_metrics;
DROP TABLE IF EXISTS print_usage_by_month;
DROP TABLE IF EXISTS print_usage_by_week;
DROP TABLE IF EXISTS print_usage_by_day;

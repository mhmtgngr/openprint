-- Rollback print quotas migration

DROP FUNCTION IF EXISTS calculate_print_job_cost CASCADE;
DROP FUNCTION IF EXISTS calculate_reset_date CASCADE;
DROP FUNCTION IF EXISTS check_quota CASCADE;

DROP TABLE IF EXISTS print_job_costs;
DROP TABLE IF EXISTS print_costs;
DROP TABLE IF EXISTS print_quotas;

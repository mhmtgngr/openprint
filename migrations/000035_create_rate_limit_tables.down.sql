-- OpenPrint Cloud - Rate Limiting Tables Rollback
-- Migration: 000035_create_rate_limit_tables.down.sql

-- Drop tables in reverse order of creation
DROP TABLE IF EXISTS rate_limit_stats CASCADE;
DROP TABLE IF EXISTS circuit_breaker_states CASCADE;
DROP TABLE IF EXISTS trusted_clients CASCADE;
DROP TABLE IF EXISTS rate_limit_violations CASCADE;
DROP TABLE IF EXISTS rate_limit_policies CASCADE;

-- Drop cleanup functions
DROP FUNCTION IF EXISTS cleanup_old_violations(INTEGER);
DROP FUNCTION IF EXISTS cleanup_old_stats(INTEGER);

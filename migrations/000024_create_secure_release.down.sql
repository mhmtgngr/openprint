-- Rollback secure release migration

DROP FUNCTION IF EXISTS attempt_secure_release CASCADE;
DROP FUNCTION IF EXISTS check_expired_secure_jobs CASCADE;

DROP TABLE IF EXISTS print_release_stations;
DROP TABLE IF EXISTS secure_release_attempts;
DROP TABLE IF EXISTS secure_print_jobs;

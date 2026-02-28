-- Rollback print_jobs migration

-- Drop trigger
DROP TRIGGER IF EXISTS update_print_jobs_updated_at ON print_jobs;

-- Drop tables
DROP TABLE IF EXISTS print_jobs;
DROP TABLE IF EXISTS documents;

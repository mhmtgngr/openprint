-- Rollback: OpenPrint Cloud - Job Assignments Table

DROP TRIGGER IF EXISTS update_job_assignments_updated_at ON job_assignments;
DROP INDEX IF EXISTS idx_job_assignments_job_agent_unique;
DROP INDEX IF EXISTS idx_job_assignments_last_heartbeat;
DROP INDEX IF EXISTS idx_job_assignments_assigned_at;
DROP INDEX IF EXISTS idx_job_assignments_agent_status;
DROP INDEX IF EXISTS idx_job_assignments_status;
DROP INDEX IF EXISTS idx_job_assignments_agent;
DROP INDEX IF EXISTS idx_job_assignments_job;
DROP TABLE IF EXISTS job_assignments;

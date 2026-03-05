-- Remove query performance indexes

DROP INDEX IF EXISTS idx_users_email;
DROP INDEX IF EXISTS idx_users_org_id;
DROP INDEX IF EXISTS idx_users_created_at;

DROP INDEX IF EXISTS idx_devices_org_id;
DROP INDEX IF EXISTS idx_devices_status;
DROP INDEX IF EXISTS idx_devices_last_seen;

DROP INDEX IF EXISTS idx_jobs_user_id;
DROP INDEX IF EXISTS idx_jobs_device_id;
DROP INDEX IF EXISTS idx_jobs_status;
DROP INDEX IF EXISTS idx_jobs_created_at;

DROP INDEX IF EXISTS idx_documents_job_id;
DROP INDEX IF EXISTS idx_documents_storage_id;
DROP INDEX IF EXISTS idx_documents_created_at;

DROP INDEX IF EXISTS idx_controls_framework;
DROP INDEX IF EXISTS idx_controls_status;
DROP INDEX IF EXISTS idx_controls_org_id;
DROP INDEX IF EXISTS idx_controls_last_review;

DROP INDEX IF EXISTS idx_audit_org_id;
DROP INDEX IF EXISTS idx_audit_event_type;
DROP INDEX IF EXISTS idx_audit_created_at;
DROP INDEX IF EXISTS idx_audit_actor_id;

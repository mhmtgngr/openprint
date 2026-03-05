-- Add query performance indexes for common lookups

-- Indexes for users table
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_org_id ON users(organization_id);
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at);

-- Indexes for devices table
CREATE INDEX IF NOT EXISTS idx_devices_org_id ON devices(organization_id);
CREATE INDEX IF NOT EXISTS idx_devices_status ON devices(status);
CREATE INDEX IF NOT EXISTS idx_devices_last_seen ON devices(last_seen);

-- Indexes for jobs table
CREATE INDEX IF NOT EXISTS idx_jobs_user_id ON jobs(user_id);
CREATE INDEX IF NOT EXISTS idx_jobs_device_id ON jobs(device_id);
CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);
CREATE INDEX IF NOT EXISTS idx_jobs_created_at ON jobs(created_at);

-- Indexes for documents table
CREATE INDEX IF NOT EXISTS idx_documents_job_id ON documents(job_id);
CREATE INDEX IF NOT EXISTS idx_documents_storage_id ON documents(storage_id);
CREATE INDEX IF NOT EXISTS idx_documents_created_at ON documents(created_at);

-- Indexes for compliance controls table
CREATE INDEX IF NOT EXISTS idx_controls_framework ON controls(framework);
CREATE INDEX IF NOT EXISTS idx_controls_status ON controls(status);
CREATE INDEX IF NOT EXISTS idx_controls_org_id ON controls(organization_id);
CREATE INDEX IF NOT EXISTS idx_controls_last_review ON controls(last_review_date);

-- Indexes for audit events table
CREATE INDEX IF NOT EXISTS idx_audit_org_id ON audit_events(organization_id);
CREATE INDEX IF NOT EXISTS idx_audit_event_type ON audit_events(event_type);
CREATE INDEX IF NOT EXISTS idx_audit_created_at ON audit_events(created_at);
CREATE INDEX IF NOT EXISTS idx_audit_actor_id ON audit_events(actor_id);

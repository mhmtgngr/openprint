-- OpenPrint Cloud - Job Assignments Table

-- Job assignments table (tracks assignment of jobs to agents)
CREATE TABLE IF NOT EXISTS job_assignments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_id UUID NOT NULL REFERENCES print_jobs(id) ON DELETE CASCADE,
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    status VARCHAR(50) NOT NULL DEFAULT 'assigned',
    retry_count INTEGER DEFAULT 0,
    last_heartbeat TIMESTAMPTZ DEFAULT NOW(),
    error TEXT,
    document_etag VARCHAR(64),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_job_assignments_job ON job_assignments(job_id);
CREATE INDEX IF NOT EXISTS idx_job_assignments_agent ON job_assignments(agent_id);
CREATE INDEX IF NOT EXISTS idx_job_assignments_status ON job_assignments(status);
CREATE INDEX IF NOT EXISTS idx_job_assignments_agent_status ON job_assignments(agent_id, status);
CREATE INDEX IF NOT EXISTS idx_job_assignments_assigned_at ON job_assignments(assigned_at DESC);
CREATE INDEX IF NOT EXISTS idx_job_assignments_last_heartbeat ON job_assignments(last_heartbeat);

-- Create unique index to prevent duplicate assignments
CREATE UNIQUE INDEX IF NOT EXISTS idx_job_assignments_job_agent_unique ON job_assignments(job_id, agent_id)
    WHERE status IN ('assigned', 'in_progress');

-- Create trigger for updated_at
CREATE TRIGGER update_job_assignments_updated_at BEFORE UPDATE ON job_assignments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Add comments for documentation
COMMENT ON TABLE job_assignments IS 'Tracks assignment of print jobs to agents';
COMMENT ON COLUMN job_assignments.job_id IS 'The print job being assigned';
COMMENT ON COLUMN job_assignments.agent_id IS 'The agent assigned to this job';
COMMENT ON COLUMN job_assignments.status IS 'Assignment status: assigned, in_progress, completed, failed, cancelled';
COMMENT ON COLUMN job_assignments.retry_count IS 'Number of times this assignment has been retried';
COMMENT ON COLUMN job_assignments.last_heartbeat IS 'Last heartbeat from agent for this assignment';
COMMENT ON COLUMN job_assignments.document_etag IS 'ETag for resume support';

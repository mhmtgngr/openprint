-- Migration: 004_create_held_jobs_table
-- Stores jobs held for secure release at a printer

CREATE TABLE IF NOT EXISTS held_jobs (
    id BIGSERIAL PRIMARY KEY,
    job_id UUID NOT NULL,
    user_id UUID NOT NULL,
    printer_id UUID,
    hold_reason VARCHAR(100) NOT NULL DEFAULT 'secure_release',
    release_code VARCHAR(20),
    held_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    released_at TIMESTAMPTZ,
    expired_at TIMESTAMPTZ,
    ttl_seconds INTEGER DEFAULT 3600,
    status VARCHAR(20) NOT NULL DEFAULT 'held' CHECK (status IN ('held', 'released', 'expired', 'cancelled'))
);

CREATE INDEX idx_held_jobs_job ON held_jobs(job_id);
CREATE INDEX idx_held_jobs_user ON held_jobs(user_id);
CREATE INDEX idx_held_jobs_printer ON held_jobs(printer_id);
CREATE INDEX idx_held_jobs_status ON held_jobs(status) WHERE status = 'held';
CREATE INDEX idx_held_jobs_release_code ON held_jobs(release_code) WHERE release_code IS NOT NULL;

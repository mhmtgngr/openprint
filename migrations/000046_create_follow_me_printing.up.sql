-- Follow-me print pools - groups of printers where jobs can be released
CREATE TABLE IF NOT EXISTS follow_me_pools (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    organization_id UUID NOT NULL,
    location VARCHAR(255),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_follow_me_pools_org ON follow_me_pools(organization_id);

-- Pool membership (which printers belong to which pool)
CREATE TABLE IF NOT EXISTS follow_me_pool_printers (
    pool_id UUID NOT NULL REFERENCES follow_me_pools(id) ON DELETE CASCADE,
    printer_id UUID NOT NULL,
    priority INTEGER DEFAULT 0,
    PRIMARY KEY (pool_id, printer_id)
);

-- Follow-me jobs (jobs submitted to a pool, not a specific printer)
CREATE TABLE IF NOT EXISTS follow_me_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL,
    pool_id UUID NOT NULL REFERENCES follow_me_pools(id),
    user_id UUID NOT NULL,
    user_email VARCHAR(255) NOT NULL,
    document_name VARCHAR(500) NOT NULL,
    page_count INTEGER DEFAULT 0,
    copies INTEGER DEFAULT 1,
    color BOOLEAN DEFAULT false,
    duplex BOOLEAN DEFAULT false,
    status VARCHAR(50) DEFAULT 'waiting',
    released_at_printer UUID,
    released_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '24 hours'),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_follow_me_jobs_user ON follow_me_jobs(user_id, status);
CREATE INDEX idx_follow_me_jobs_pool ON follow_me_jobs(pool_id, status);
CREATE INDEX idx_follow_me_jobs_expires ON follow_me_jobs(expires_at) WHERE status = 'waiting';

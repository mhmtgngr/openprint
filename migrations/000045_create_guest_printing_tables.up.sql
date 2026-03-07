-- Guest print tokens for visitor access
CREATE TABLE IF NOT EXISTS guest_print_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token VARCHAR(64) NOT NULL UNIQUE,
    email VARCHAR(255),
    name VARCHAR(255),
    organization_id UUID NOT NULL,
    created_by UUID NOT NULL,
    printer_ids UUID[] DEFAULT '{}',
    max_pages INTEGER DEFAULT 10,
    max_jobs INTEGER DEFAULT 5,
    pages_used INTEGER DEFAULT 0,
    jobs_used INTEGER DEFAULT 0,
    color_allowed BOOLEAN DEFAULT false,
    duplex_required BOOLEAN DEFAULT false,
    expires_at TIMESTAMPTZ NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMPTZ
);

CREATE INDEX idx_guest_tokens_token ON guest_print_tokens(token);
CREATE INDEX idx_guest_tokens_org ON guest_print_tokens(organization_id);
CREATE INDEX idx_guest_tokens_active ON guest_print_tokens(is_active, expires_at);

-- Guest print jobs tracking
CREATE TABLE IF NOT EXISTS guest_print_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token_id UUID NOT NULL REFERENCES guest_print_tokens(id) ON DELETE CASCADE,
    document_name VARCHAR(500) NOT NULL,
    page_count INTEGER DEFAULT 0,
    printer_id UUID,
    status VARCHAR(50) DEFAULT 'pending',
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    error_message TEXT
);

CREATE INDEX idx_guest_jobs_token ON guest_print_jobs(token_id);

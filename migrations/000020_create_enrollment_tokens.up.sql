-- Create enrollment_tokens table for agent enrollment token management
CREATE TABLE IF NOT EXISTS enrollment_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token VARCHAR(255) UNIQUE NOT NULL,
    organization_id VARCHAR(255),
    name VARCHAR(255) NOT NULL,
    created_by VARCHAR(255) NOT NULL,
    max_uses INTEGER NOT NULL DEFAULT 0, -- 0 means unlimited uses
    use_count INTEGER NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    revoked_by VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create index on token for fast lookup
CREATE INDEX idx_enrollment_tokens_token ON enrollment_tokens(token);

-- Create index on organization_id for filtering
CREATE INDEX idx_enrollment_tokens_org_id ON enrollment_tokens(organization_id);

-- Create index on expires_at for cleanup
CREATE INDEX idx_enrollment_tokens_expires_at ON enrollment_tokens(expires_at);

-- Create index on revoked_at for filtering
CREATE INDEX idx_enrollment_tokens_revoked_at ON enrollment_tokens(revoked_at);

-- Add comment
COMMENT ON TABLE enrollment_tokens IS 'Stores enrollment tokens for secure agent registration';
COMMENT ON COLUMN enrollment_tokens.max_uses IS 'Maximum number of times this token can be used (0 = unlimited)';
COMMENT ON COLUMN enrollment_tokens.expires_at IS 'Optional expiration time for the token';
COMMENT ON COLUMN enrollment_tokens.revoked_at IS 'Set when token is revoked before expiration';

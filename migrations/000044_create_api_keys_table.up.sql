-- Migration: 008_create_api_keys_table
-- Fine-grained API key permissions for the developer portal

CREATE TABLE IF NOT EXISTS api_key_permissions (
    id BIGSERIAL PRIMARY KEY,
    api_key_id UUID NOT NULL,
    resource VARCHAR(100) NOT NULL,
    actions TEXT[] NOT NULL DEFAULT '{}',
    conditions JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_api_key_permissions_key ON api_key_permissions(api_key_id);
CREATE INDEX idx_api_key_permissions_resource ON api_key_permissions(resource);

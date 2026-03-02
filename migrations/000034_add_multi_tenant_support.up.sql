-- Migration: Add Multi-Tenant Support
-- This migration adds tables and RLS policies for multi-tenancy support

-- Add tenant_id column to existing tables (if not exists)
ALTER TABLE printers ADD COLUMN IF NOT EXISTS tenant_id uuid REFERENCES organizations(id) ON DELETE SET NULL;
ALTER TABLE print_jobs ADD COLUMN IF NOT EXISTS tenant_id uuid REFERENCES organizations(id) ON DELETE SET NULL;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS tenant_id uuid REFERENCES organizations(id) ON DELETE SET NULL;

-- Create quota_configs table
CREATE TABLE IF NOT EXISTS quota_configs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    max_printers integer NOT NULL DEFAULT 100,
    max_storage_gb integer NOT NULL DEFAULT 100,
    max_jobs_per_month integer NOT NULL DEFAULT 10000,
    max_users integer NOT NULL DEFAULT 50,
    alert_threshold integer NOT NULL DEFAULT 80,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id)
);

-- Create index on tenant_id for quota_configs
CREATE INDEX IF NOT EXISTS idx_quota_configs_tenant_id ON quota_configs(tenant_id);

-- Create quota_usage table
CREATE TABLE IF NOT EXISTS quota_usage (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    printers_count integer NOT NULL DEFAULT 0,
    storage_used_gb bigint NOT NULL DEFAULT 0,  -- Stored in bytes
    jobs_this_month integer NOT NULL DEFAULT 0,
    users_count integer NOT NULL DEFAULT 0,
    month timestamptz NOT NULL DEFAULT DATE_TRUNC('month', NOW()),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, month)
);

-- Create index on tenant_id and month for quota_usage
CREATE INDEX IF NOT EXISTS idx_quota_usage_tenant_month ON quota_usage(tenant_id, month DESC);

-- Create organization_users table
CREATE TABLE IF NOT EXISTS organization_users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role text NOT NULL DEFAULT 'member',
    settings jsonb DEFAULT '{}'::jsonb,
    joined_at timestamptz NOT NULL DEFAULT NOW(),
    invited_by uuid REFERENCES users(id) ON DELETE SET NULL,
    deleted_at timestamptz,
    UNIQUE(organization_id, user_id)
);

-- Create indexes for organization_users
CREATE INDEX IF NOT EXISTS idx_organization_users_org_id ON organization_users(organization_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_organization_users_user_id ON organization_users(user_id) WHERE deleted_at IS NULL;

-- Add check constraint for role
ALTER TABLE organization_users DROP CONSTRAINT IF EXISTS organization_users_role_check;
ALTER TABLE organization_users ADD CONSTRAINT organization_users_role_check
    CHECK (role IN ('owner', 'admin', 'member', 'viewer', 'billing'));

-- Create platform_admin role (if not exists) - must be created before policies that reference it
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'platform_admin') THEN
        CREATE ROLE platform_admin;
    END IF;
END
$$;

-- Grant necessary permissions
GRANT USAGE ON SCHEMA public TO platform_admin;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO platform_admin;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO platform_admin;

-- Enable Row Level Security on tables
ALTER TABLE printers ENABLE ROW LEVEL SECURITY;
ALTER TABLE print_jobs ENABLE ROW LEVEL SECURITY;
ALTER TABLE documents ENABLE ROW LEVEL SECURITY;
ALTER TABLE quota_configs ENABLE ROW LEVEL SECURITY;
ALTER TABLE quota_usage ENABLE ROW LEVEL SECURITY;
ALTER TABLE organization_users ENABLE ROW LEVEL SECURITY;

-- Create RLS policies for printers
DROP POLICY IF EXISTS printers_tenant_isolation ON printers;
CREATE POLICY printers_tenant_isolation ON printers
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS printers_platform_admin ON printers;
CREATE POLICY printers_platform_admin ON printers
    FOR ALL
    TO platform_admin
    USING (true)
    WITH CHECK (true);

-- Create RLS policies for print_jobs
DROP POLICY IF EXISTS print_jobs_tenant_isolation ON print_jobs;
CREATE POLICY print_jobs_tenant_isolation ON print_jobs
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS print_jobs_platform_admin ON print_jobs;
CREATE POLICY print_jobs_platform_admin ON print_jobs
    FOR ALL
    TO platform_admin
    USING (true)
    WITH CHECK (true);

-- Create RLS policies for documents
DROP POLICY IF EXISTS documents_tenant_isolation ON documents;
CREATE POLICY documents_tenant_isolation ON documents
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS documents_platform_admin ON documents;
CREATE POLICY documents_platform_admin ON documents
    FOR ALL
    TO platform_admin
    USING (true)
    WITH CHECK (true);

-- Create RLS policies for quota_configs
DROP POLICY IF EXISTS quota_configs_tenant_isolation ON quota_configs;
CREATE POLICY quota_configs_tenant_isolation ON quota_configs
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS quota_configs_platform_admin ON quota_configs;
CREATE POLICY quota_configs_platform_admin ON quota_configs
    FOR ALL
    TO platform_admin
    USING (true)
    WITH CHECK (true);

-- Create RLS policies for quota_usage
DROP POLICY IF EXISTS quota_usage_tenant_isolation ON quota_usage;
CREATE POLICY quota_usage_tenant_isolation ON quota_usage
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS quota_usage_platform_admin ON quota_usage;
CREATE POLICY quota_usage_platform_admin ON quota_usage
    FOR ALL
    TO platform_admin
    USING (true)
    WITH CHECK (true);

-- Create RLS policies for organization_users
DROP POLICY IF EXISTS organization_users_tenant_isolation ON organization_users;
CREATE POLICY organization_users_tenant_isolation ON organization_users
    FOR ALL
    USING (organization_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (organization_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

DROP POLICY IF EXISTS organization_users_platform_admin ON organization_users;
CREATE POLICY organization_users_platform_admin ON organization_users
    FOR ALL
    TO platform_admin
    USING (true)
    WITH CHECK (true);

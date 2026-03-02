-- Rollback: Remove Multi-Tenant Support

-- Drop RLS policies
DROP POLICY IF EXISTS printers_tenant_isolation ON printers;
DROP POLICY IF EXISTS printers_platform_admin ON printers;
DROP POLICY IF EXISTS print_jobs_tenant_isolation ON print_jobs;
DROP POLICY IF EXISTS print_jobs_platform_admin ON print_jobs;
DROP POLICY IF EXISTS documents_tenant_isolation ON documents;
DROP POLICY IF EXISTS documents_platform_admin ON documents;
DROP POLICY IF EXISTS quota_configs_tenant_isolation ON quota_configs;
DROP POLICY IF EXISTS quota_configs_platform_admin ON quota_configs;
DROP POLICY IF EXISTS quota_usage_tenant_isolation ON quota_usage;
DROP POLICY IF EXISTS quota_usage_platform_admin ON quota_usage;
DROP POLICY IF EXISTS organization_users_tenant_isolation ON organization_users;
DROP POLICY IF EXISTS organization_users_platform_admin ON organization_users;

-- Disable RLS on tables
ALTER TABLE printers DISABLE ROW LEVEL SECURITY;
ALTER TABLE print_jobs DISABLE ROW LEVEL SECURITY;
ALTER TABLE documents DISABLE ROW LEVEL SECURITY;
ALTER TABLE quota_configs DISABLE ROW LEVEL SECURITY;
ALTER TABLE quota_usage DISABLE ROW LEVEL SECURITY;
ALTER TABLE organization_users DISABLE ROW LEVEL SECURITY;

-- Drop organization_users table
DROP TABLE IF EXISTS organization_users CASCADE;

-- Drop quota_usage table
DROP TABLE IF EXISTS quota_usage CASCADE;

-- Drop quota_configs table
DROP TABLE IF EXISTS quota_configs CASCADE;

-- Drop tenant_id columns from existing tables
ALTER TABLE printers DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE print_jobs DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE documents DROP COLUMN IF EXISTS tenant_id;

-- Drop platform_admin role
DROP ROLE IF EXISTS platform_admin;

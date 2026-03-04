-- OpenPrint Cloud - Rollback organization_id Nullable Change
-- Migration: 000036_make_organization_id_nullable.down.sql
--
-- This rollback removes the global policy support features.

-- Step 1: Drop the validation trigger
DROP TRIGGER IF EXISTS validate_print_policy_trigger ON print_policies;

-- Step 2: Drop the validation function
DROP FUNCTION IF EXISTS validate_global_policy();

-- Step 3: Drop the is_global_policy function
DROP FUNCTION IF EXISTS is_global_policy(UUID);

-- Step 4: Drop the organization_id index
DROP INDEX IF EXISTS idx_print_policies_scope_org_id;

-- Step 5: Recreate the original scope index (without the org_id specific index)
DROP INDEX IF EXISTS idx_print_policies_scope;
CREATE INDEX idx_print_policies_scope ON print_policies USING GIN(scope);

-- Step 6: Restore the NOT NULL constraint on organization_id
-- First, update any NULL values to a default organization if possible
-- In production, you should handle this based on business requirements
ALTER TABLE print_policies ALTER COLUMN organization_id SET NOT NULL;

-- Step 7: Restore original comments
COMMENT ON COLUMN print_policies.organization_id IS 'Organization ID';
COMMENT ON COLUMN print_policies.scope IS 'JSON object defining policy scope (users, groups, printers, etc.)';

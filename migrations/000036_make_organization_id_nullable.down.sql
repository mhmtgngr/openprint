-- OpenPrint Cloud - Rollback organization_id Nullable Change
-- Migration: 000036_make_organization_id_nullable.down.sql
--
-- This rollback removes the global policy support features.

-- Drop the validation trigger
DROP TRIGGER IF EXISTS validate_print_policy_trigger ON print_policies;

-- Drop the validation function
DROP FUNCTION IF EXISTS validate_global_policy();

-- Drop the is_global_policy function
DROP FUNCTION IF EXISTS is_global_policy(UUID);

-- Drop the organization_id index
DROP INDEX IF EXISTS idx_print_policies_scope_org_id;

-- Recreate the original scope index (without the org_id specific index)
DROP INDEX IF EXISTS idx_print_policies_scope;
CREATE INDEX idx_print_policies_scope ON print_policies USING GIN(scope);

-- Update any NULL or empty organization_id values in existing policies
-- Set them to a default organization if needed, or they will be handled by application logic
-- Note: This is a data migration decision that should be made based on business requirements

-- Restore original comment
COMMENT ON COLUMN print_policies.scope IS 'JSON object defining policy scope (users, groups, printers, etc.)';

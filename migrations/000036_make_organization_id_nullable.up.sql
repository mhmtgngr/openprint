-- OpenPrint Cloud - Make organization_id Nullable in print_policies
-- Migration: 000036_make_organization_id_nullable.up.sql
--
-- This migration allows NULL organization_id values in the print_policies table,
-- enabling global/system-wide policies that apply to all organizations.

-- Step 1: Make the organization_id column nullable
ALTER TABLE print_policies ALTER COLUMN organization_id DROP NOT NULL;

-- Step 2: Update any NULL organization_id values in the scope JSONB to empty strings
-- to maintain data consistency before making the column nullable
UPDATE print_policies
SET scope = jsonb_set(
    COALESCE(scope, '{}'::jsonb),
    '{organization_id}',
    to_jsonb(COALESCE(scope->>'organization_id', ''))
)
WHERE scope IS NOT NULL;

-- Step 3: Drop the existing scope index for rebuild
DROP INDEX IF EXISTS idx_print_policies_scope;

-- Step 4: Recreate the GIN index on scope with updated structure
CREATE INDEX idx_print_policies_scope ON print_policies USING GIN(scope);

-- Step 5: Add index for organization_id lookups within the JSONB scope
CREATE INDEX idx_print_policies_scope_org_id ON print_policies((scope->>'organization_id')) WHERE scope->>'organization_id' IS NOT NULL AND scope->>'organization_id' != '';

-- Step 6: Add comment documenting that NULL or empty organization_id means global policy
COMMENT ON COLUMN print_policies.organization_id IS 'Organization ID (nullable). NULL indicates a global/system-wide policy applicable to all organizations.';
COMMENT ON COLUMN print_policies.scope IS 'JSON object defining policy scope (users, groups, printers, etc.). Empty or null organization_id indicates a global/system-wide policy applicable to all organizations.';

-- Step 7: Add function to check if a policy is global
CREATE OR REPLACE FUNCTION is_global_policy(policy_id UUID)
RETURNS BOOLEAN AS $$
DECLARE
    v_org_id TEXT;
BEGIN
    SELECT scope->>'organization_id'
    INTO v_org_id
    FROM print_policies
    WHERE id = policy_id;

    RETURN v_org_id IS NULL OR v_org_id = '';
END;
$$ LANGUAGE plpgsql;

-- Step 8: Add trigger to ensure only admins can create global policies
-- This is a placeholder - actual implementation depends on the auth system
CREATE OR REPLACE FUNCTION validate_global_policy()
RETURNS TRIGGER AS $$
BEGIN
    -- If organization_id is null or empty, this is a global policy
    IF NEW.scope->>'organization_id' IS NULL OR (NEW.scope->>'organization_id') = '' THEN
        -- In production, add check here for admin privileges
        -- For now, we just log a notice
        RAISE NOTICE 'Creating global policy: %', NEW.name;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Step 9: Create trigger for policy validation
DROP TRIGGER IF EXISTS validate_print_policy_trigger ON print_policies;
CREATE TRIGGER validate_print_policy_trigger
    BEFORE INSERT OR UPDATE ON print_policies
    FOR EACH ROW
    EXECUTE FUNCTION validate_global_policy();

-- Remove Extended Print Policy Columns

-- Drop helper functions
DROP FUNCTION IF EXISTS increment_policy_evaluation_count(UUID);
DROP FUNCTION IF EXISTS increment_policy_triggered_count(UUID);

-- Remove indexes
DROP INDEX IF EXISTS idx_print_policies_scope;
DROP INDEX IF EXISTS idx_print_policies_rules;
DROP INDEX IF EXISTS idx_print_policies_status;
DROP INDEX IF EXISTS idx_print_policies_type;

-- Remove columns from print_policies table
ALTER TABLE print_policies
DROP COLUMN IF EXISTS type,
DROP COLUMN IF EXISTS status,
DROP COLUMN IF EXISTS rules,
DROP COLUMN IF EXISTS actions,
DROP COLUMN IF EXISTS scope,
DROP COLUMN IF EXISTS created_by,
DROP COLUMN IF EXISTS modified_by,
DROP COLUMN IF EXISTS version,
DROP COLUMN IF EXISTS evaluated_count,
DROP COLUMN IF EXISTS triggered_count;

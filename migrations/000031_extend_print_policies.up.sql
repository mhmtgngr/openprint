-- Extend Print Policies for Enhanced Policy Engine
-- Adds JSONB columns for complex rules, actions, and scoping

-- Add new columns to existing print_policies table
ALTER TABLE print_policies
ADD COLUMN IF NOT EXISTS type VARCHAR(50) DEFAULT 'general',
ADD COLUMN IF NOT EXISTS status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'draft', 'archived')),
ADD COLUMN IF NOT EXISTS rules JSONB DEFAULT '[]'::jsonb,
ADD COLUMN IF NOT EXISTS actions JSONB DEFAULT '[]'::jsonb,
ADD COLUMN IF NOT EXISTS scope JSONB DEFAULT '{}'::jsonb,
ADD COLUMN IF NOT EXISTS created_by UUID REFERENCES users(id),
ADD COLUMN IF NOT EXISTS modified_by UUID REFERENCES users(id),
ADD COLUMN IF NOT EXISTS version INTEGER DEFAULT 1,
ADD COLUMN IF NOT EXISTS evaluated_count INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS triggered_count INTEGER DEFAULT 0;

-- Create index on policy type
CREATE INDEX IF NOT EXISTS idx_print_policies_type ON print_policies(type);

-- Create index on policy status
CREATE INDEX IF NOT EXISTS idx_print_policies_status ON print_policies(status);

-- Create GIN index on rules for JSONB queries
CREATE INDEX IF NOT EXISTS idx_print_policies_rules ON print_policies USING GIN(rules);

-- Create GIN index on scope for JSONB queries
CREATE INDEX IF NOT EXISTS idx_print_policies_scope ON print_policies USING GIN(scope);

-- Add comments for documentation
COMMENT ON COLUMN print_policies.type IS 'Policy type: quota, access, content, routing, watermark, retention, cost_center';
COMMENT ON COLUMN print_policies.status IS 'Policy status: active, inactive, draft, archived';
COMMENT ON COLUMN print_policies.rules IS 'JSON array of rule conditions for policy evaluation';
COMMENT ON COLUMN print_policies.actions IS 'JSON array of actions to take when policy is triggered';
COMMENT ON COLUMN print_policies.scope IS 'JSON object defining policy scope (users, groups, printers, etc.)';
COMMENT ON COLUMN print_policies.version IS 'Version number for optimistic locking';
COMMENT ON COLUMN print_policies.evaluated_count IS 'Number of times this policy has been evaluated';
COMMENT ON COLUMN print_policies.triggered_count IS 'Number of times this policy has been triggered';

-- Function to increment evaluation count
CREATE OR REPLACE FUNCTION increment_policy_evaluation_count(p_policy_id UUID)
RETURNS VOID AS $$
BEGIN
    UPDATE print_policies
    SET evaluated_count = evaluated_count + 1
    WHERE id = p_policy_id;
END;
$$ LANGUAGE plpgsql;

-- Function to increment triggered count
CREATE OR REPLACE FUNCTION increment_policy_triggered_count(p_policy_id UUID)
RETURNS VOID AS $$
BEGIN
    UPDATE print_policies
    SET triggered_count = triggered_count + 1
    WHERE id = p_policy_id;
END;
$$ LANGUAGE plpgsql;

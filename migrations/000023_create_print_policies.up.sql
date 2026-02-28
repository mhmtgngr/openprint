-- Print Policies Table
-- Defines print policies for organizations

CREATE TABLE IF NOT EXISTS print_policies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT true,
    priority INTEGER DEFAULT 0, -- Higher priority policies are evaluated first
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_print_policies_org ON print_policies(organization_id);
CREATE INDEX idx_print_policies_active ON print_policies(is_active);
CREATE INDEX idx_print_policies_priority ON print_policies(priority DESC);

-- Print Policy Rules
-- Individual rules within a policy

CREATE TABLE IF NOT EXISTS print_policy_rules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    policy_id UUID NOT NULL REFERENCES print_policies(id) ON DELETE CASCADE,
    rule_type VARCHAR(50) NOT NULL, -- 'restrict_color', 'max_copies', 'allow_duplex', 'require_pin', 'time_restrictions', etc.
    rule_operator VARCHAR(20) NOT NULL, -- 'equals', 'not_equals', 'greater_than', 'less_than', 'contains', 'between'
    rule_value JSONB NOT NULL,
    rule_action VARCHAR(50) NOT NULL, -- 'allow', 'deny', 'warn', 'require_approval'
    action_value JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_print_policy_rules_policy ON print_policy_rules(policy_id);
CREATE INDEX idx_print_policy_rules_type ON print_policy_rules(rule_type);

-- Print Policy Assignments
-- Assigns policies to users, groups, or printers

CREATE TABLE IF NOT EXISTS print_policy_assignments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    policy_id UUID NOT NULL REFERENCES print_policies(id) ON DELETE CASCADE,
    entity_id UUID NOT NULL,
    entity_type VARCHAR(20) NOT NULL CHECK (entity_type IN ('user', 'group', 'printer', 'printer_group')),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(policy_id, entity_id, entity_type)
);

CREATE INDEX idx_print_policy_assignments_policy ON print_policy_assignments(policy_id);
CREATE INDEX idx_print_policy_assignments_entity ON print_policy_assignments(entity_id, entity_type);

-- Print Job Policy Evaluations
-- Logs policy evaluations for audit purposes

CREATE TABLE IF NOT EXISTS print_policy_evaluations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_id UUID REFERENCES print_jobs(id) ON DELETE CASCADE,
    policy_id UUID REFERENCES print_policies(id),
    policy_name VARCHAR(255),
    result VARCHAR(20) NOT NULL, -- 'allowed', 'denied', 'warned', 'approval_required'
    evaluated_at TIMESTAMPTZ DEFAULT NOW(),
    evaluated_by VARCHAR(100), -- 'system', 'user_id'
    details JSONB
);

CREATE INDEX idx_print_policy_evaluations_job ON print_policy_evaluations(job_id);
CREATE INDEX idx_print_policy_evaluations_result ON print_policy_evaluations(result);

-- Triggers
CREATE TRIGGER update_print_policies_updated_at BEFORE UPDATE ON print_policies
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_print_policy_rules_updated_at BEFORE UPDATE ON print_policy_rules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to evaluate print policies
CREATE OR REPLACE FUNCTION evaluate_print_policies(
    p_job_id UUID,
    p_user_id UUID,
    p_organization_id UUID,
    p_printer_id UUID,
    p_document_attributes JSONB
) RETURNS TABLE(policy_id UUID, result VARCHAR, action_value JSONB) AS $$
DECLARE
    v_policy RECORD;
    v_rule RECORD;
    v_rule_result BOOLEAN;
    v_final_result VARCHAR := 'allow';
BEGIN
    -- Get all applicable policies for the organization, user, and printer
    FOR v_policy IN
        SELECT DISTINCT p.id, p.name, p.priority
        FROM print_policies p
        INNER JOIN print_policy_assignments a ON a.policy_id = p.id
        WHERE p.organization_id = p_organization_id
          AND p.is_active = true
          AND (a.entity_type = 'organization' OR a.entity_id = p_user_id OR a.entity_id = p_printer_id)
        ORDER BY p.priority DESC
    LOOP
        -- Evaluate each rule in the policy
        v_final_result := 'allow';
        FOR v_rule IN
            SELECT rule_type, rule_operator, rule_value, rule_action, action_value
            FROM print_policy_rules
            WHERE policy_id = v_policy.id
        LOOP
            -- Evaluate rule (simplified - in production would parse JSONB and compare)
            v_rule_result := true; -- Placeholder

            IF NOT v_rule_result THEN
                IF v_rule.rule_action = 'deny' THEN
                    v_final_result := 'denied';
                    -- Log evaluation
                    INSERT INTO print_policy_evaluations (job_id, policy_id, policy_name, result, details)
                    VALUES (p_job_id, v_policy.id, v_policy.name, 'denied', v_rule.rule_value);
                    RETURN NEXT;
                    RETURN;
                ELSIF v_rule.rule_action = 'warn' THEN
                    v_final_result := 'warned';
                ELSIF v_rule.rule_action = 'require_approval' THEN
                    v_final_result := 'approval_required';
                END IF;
            END IF;
        END LOOP;

        -- Log successful evaluation
        INSERT INTO print_policy_evaluations (job_id, policy_id, policy_name, result)
        VALUES (p_job_id, v_policy.id, v_policy.name, v_final_result);

        RETURN NEXT;
    END LOOP;

    RETURN;
END;
$$ LANGUAGE plpgsql;

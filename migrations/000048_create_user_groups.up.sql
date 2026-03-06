-- User groups for policy and quota assignment
CREATE TABLE IF NOT EXISTS user_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    organization_id UUID NOT NULL,
    color VARCHAR(7) DEFAULT '#6366F1',
    is_active BOOLEAN DEFAULT true,
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(organization_id, name)
);

CREATE INDEX idx_user_groups_org ON user_groups(organization_id) WHERE is_active = true;

-- Group membership
CREATE TABLE IF NOT EXISTS user_group_members (
    group_id UUID NOT NULL REFERENCES user_groups(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    added_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    added_by UUID,
    PRIMARY KEY (group_id, user_id)
);

CREATE INDEX idx_group_members_user ON user_group_members(user_id);

-- Group-based printer access
CREATE TABLE IF NOT EXISTS group_printer_access (
    group_id UUID NOT NULL REFERENCES user_groups(id) ON DELETE CASCADE,
    printer_id UUID NOT NULL,
    can_color BOOLEAN DEFAULT true,
    can_duplex BOOLEAN DEFAULT true,
    max_pages_per_job INTEGER,
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (group_id, printer_id)
);

-- User-printer mappings: maps RDP session users to their local client-side agents/printers
-- Used for routing print jobs captured on RDP session hosts to the user's local printer
CREATE TABLE IF NOT EXISTS user_printer_mappings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID REFERENCES organizations(id),
    -- The user who prints on the RDP session host
    user_email VARCHAR(255) NOT NULL,
    user_name VARCHAR(255),
    -- The client-side agent that manages the user's local printers
    client_agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    -- The specific local printer to route jobs to (discovered_printers.id)
    target_printer_id UUID,
    -- The printer name on the client machine (for display and matching)
    target_printer_name VARCHAR(500),
    -- The server-side agent that captures print jobs (RDP session host)
    server_agent_id UUID REFERENCES agents(id) ON DELETE SET NULL,
    -- Whether this mapping is currently active
    is_active BOOLEAN NOT NULL DEFAULT true,
    -- Whether this is the default mapping for the user
    is_default BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- A user can have one default mapping per organization
    CONSTRAINT unique_user_default_mapping UNIQUE (organization_id, user_email, is_default) DEFERRABLE INITIALLY DEFERRED
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_user_printer_mappings_email ON user_printer_mappings(user_email);
CREATE INDEX IF NOT EXISTS idx_user_printer_mappings_client_agent ON user_printer_mappings(client_agent_id);
CREATE INDEX IF NOT EXISTS idx_user_printer_mappings_server_agent ON user_printer_mappings(server_agent_id);
CREATE INDEX IF NOT EXISTS idx_user_printer_mappings_org ON user_printer_mappings(organization_id);
CREATE INDEX IF NOT EXISTS idx_user_printer_mappings_active ON user_printer_mappings(user_email, is_active) WHERE is_active = true;

-- Add agent_role column to agents table to distinguish server vs client agents
ALTER TABLE agents ADD COLUMN IF NOT EXISTS agent_role VARCHAR(50) NOT NULL DEFAULT 'standard';
-- agent_role values: 'server' (RDP session host), 'client' (user workstation), 'standard' (both/legacy)

COMMENT ON TABLE user_printer_mappings IS 'Maps users on RDP session hosts to their local printers via client-side agents';
COMMENT ON COLUMN user_printer_mappings.client_agent_id IS 'Agent running on the user workstation that manages local printers';
COMMENT ON COLUMN user_printer_mappings.server_agent_id IS 'Agent running on the RDP session host that captures print jobs';
COMMENT ON COLUMN user_printer_mappings.target_printer_id IS 'Specific discovered printer on the client agent to route jobs to';
COMMENT ON COLUMN agents.agent_role IS 'Role: server (RDP host capture), client (local print), standard (both)';

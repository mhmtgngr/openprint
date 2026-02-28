-- OpenPrint Cloud - Printers Table

-- Printers table
CREATE TABLE IF NOT EXISTS printers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
    status VARCHAR(50) DEFAULT 'offline', -- online, offline, busy, error
    capabilities JSONB, -- Printer capabilities (color, duplex, media, etc.)
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_printers_agent ON printers(agent_id);
CREATE INDEX IF NOT EXISTS idx_printers_organization ON printers(organization_id);
CREATE INDEX IF NOT EXISTS idx_printers_status ON printers(status);

-- Create trigger for updated_at
CREATE TRIGGER update_printers_updated_at BEFORE UPDATE ON printers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

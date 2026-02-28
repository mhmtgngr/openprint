-- OpenPrint Cloud - Discovered Printers Table

-- Discovered printers table (printers found by agents)
CREATE TABLE IF NOT EXISTS discovered_printers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    driver VARCHAR(255),
    driver_version VARCHAR(100),
    port VARCHAR(100),
    connection_type VARCHAR(20) NOT NULL DEFAULT 'local',
    status VARCHAR(50) NOT NULL DEFAULT 'idle',
    is_default BOOLEAN DEFAULT false,
    is_shared BOOLEAN DEFAULT false,
    share_name VARCHAR(255),
    location VARCHAR(255),
    capabilities JSONB,
    last_seen TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(agent_id, name)
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_discovered_printers_agent ON discovered_printers(agent_id);
CREATE INDEX IF NOT EXISTS idx_discovered_printers_status ON discovered_printers(status);
CREATE INDEX IF NOT EXISTS idx_discovered_printers_connection_type ON discovered_printers(connection_type);
CREATE INDEX IF NOT EXISTS idx_discovered_printers_last_seen ON discovered_printers(last_seen);
CREATE INDEX IF NOT EXISTS idx_discovered_printers_name ON discovered_printers(name);

-- Create index for GIN queries on capabilities JSONB
CREATE INDEX IF NOT EXISTS idx_discovered_printers_capabilities ON discovered_printers USING GIN (capabilities);

-- Create trigger for updated_at
CREATE TRIGGER update_discovered_printers_updated_at BEFORE UPDATE ON discovered_printers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create function to update updated_at column if it doesn't exist
CREATE OR REPLACE FUNCTION update_updated_at_column() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Add comments for documentation
COMMENT ON TABLE discovered_printers IS 'Printers discovered by Windows agents';
COMMENT ON COLUMN discovered_printers.agent_id IS 'The agent that discovered this printer';
COMMENT ON COLUMN discovered_printers.name IS 'Printer name as reported by the OS';
COMMENT ON COLUMN discovered_printers.connection_type IS 'Connection type: local, network, shared, wsd, lpd';
COMMENT ON COLUMN discovered_printers.status IS 'Printer status: idle, printing, busy, offline, error, out_of_paper, low_toner, door_open';
COMMENT ON COLUMN discovered_printers.capabilities IS 'Printer capabilities as JSONB';
COMMENT ON COLUMN discovered_printers.last_seen IS 'When this printer was last seen by the agent';

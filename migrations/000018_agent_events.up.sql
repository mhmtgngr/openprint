-- OpenPrint Cloud - Agent Events Table

-- Agent events table (logs events from agents)
CREATE TABLE IF NOT EXISTS agent_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    event_type VARCHAR(100) NOT NULL,
    severity VARCHAR(20) NOT NULL DEFAULT 'info',
    message TEXT NOT NULL,
    details JSONB,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_agent_events_agent ON agent_events(agent_id);
CREATE INDEX IF NOT EXISTS idx_agent_events_type ON agent_events(event_type);
CREATE INDEX IF NOT EXISTS idx_agent_events_severity ON agent_events(severity);
CREATE INDEX IF NOT EXISTS idx_agent_events_timestamp ON agent_events(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_agent_events_agent_timestamp ON agent_events(agent_id, timestamp DESC);

-- Create index for GIN queries on details JSONB
CREATE INDEX IF NOT EXISTS idx_agent_events_details ON agent_events USING GIN (details);

-- Create partitioning function (optional, for large deployments)
-- This would allow partitioning by timestamp

-- Add comments for documentation
COMMENT ON TABLE agent_events IS 'Event log for agent activities';
COMMENT ON COLUMN agent_events.agent_id IS 'The agent that generated this event';
COMMENT ON COLUMN agent_events.event_type IS 'Event type: printer_added, printer_removed, job_started, job_completed, error, etc.';
COMMENT ON COLUMN agent_events.severity IS 'Severity level: debug, info, warning, error, critical';
COMMENT ON COLUMN agent_events.message IS 'Human-readable event message';
COMMENT ON COLUMN agent_events.details IS 'Additional event details as JSONB';
COMMENT ON COLUMN agent_events.timestamp IS 'When the event occurred';

-- Create function to clean up old events (retention policy)
CREATE OR REPLACE FUNCTION cleanup_old_events(retention_days INTEGER DEFAULT 90)
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM agent_events
    WHERE timestamp < NOW() - (retention_days || ' days')::INTERVAL;

    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Add comment for cleanup function
COMMENT ON FUNCTION cleanup_old_events IS 'Deletes events older than the specified retention period';

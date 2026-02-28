-- Rollback: OpenPrint Cloud - Agent Events Table

DROP FUNCTION IF EXISTS cleanup_old_events(INTEGER);
DROP INDEX IF EXISTS idx_agent_events_details;
DROP INDEX IF EXISTS idx_agent_events_agent_timestamp;
DROP INDEX IF EXISTS idx_agent_events_timestamp;
DROP INDEX IF EXISTS idx_agent_events_severity;
DROP INDEX IF EXISTS idx_agent_events_type;
DROP INDEX IF EXISTS idx_agent_events_agent;
DROP TABLE IF EXISTS agent_events;

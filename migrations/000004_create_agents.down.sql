-- Rollback agents migration

-- Drop trigger
DROP TRIGGER IF EXISTS update_agents_updated_at ON agents;

-- Drop table
DROP TABLE IF EXISTS agents;

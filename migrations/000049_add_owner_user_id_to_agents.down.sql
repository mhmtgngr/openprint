-- Rollback migration - Remove owner_user_id from agents table

DROP INDEX IF EXISTS idx_agents_owner_user_id;

ALTER TABLE agents DROP COLUMN IF EXISTS owner_user_id;

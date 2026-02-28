-- Rollback webhooks migration

-- Drop trigger
DROP TRIGGER IF EXISTS update_webhooks_updated_at ON webhooks;

-- Drop table
DROP TABLE IF EXISTS webhooks;

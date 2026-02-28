-- Rollback users migration

-- Drop trigger
DROP TRIGGER IF EXISTS update_users_updated_at ON users;

-- Drop table
DROP TABLE IF EXISTS users;

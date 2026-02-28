-- Rollback organizations migration

-- Drop trigger
DROP TRIGGER IF EXISTS update_organizations_updated_at ON organizations;

-- Drop function (only if this is the last migration using it)
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop table
DROP TABLE IF EXISTS organizations;

-- Drop extension (only if this is the last migration)
-- DROP EXTENSION IF EXISTS "uuid-ossp";

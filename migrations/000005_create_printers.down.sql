-- Rollback printers migration

-- Drop trigger
DROP TRIGGER IF EXISTS update_printers_updated_at ON printers;

-- Drop table
DROP TABLE IF EXISTS printers;

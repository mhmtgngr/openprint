ALTER TABLE user_printer_mappings DROP CONSTRAINT IF EXISTS unique_user_default_per_type;
ALTER TABLE user_printer_mappings ADD CONSTRAINT unique_user_default_mapping
    UNIQUE (organization_id, user_email, is_default) DEFERRABLE INITIALLY DEFERRED;
DROP INDEX IF EXISTS idx_user_printer_mappings_type;
ALTER TABLE user_printer_mappings DROP COLUMN IF EXISTS printer_type;

-- Add printer_type column to user_printer_mappings to distinguish standard vs receipt printers.
-- This allows users to have separate default mappings for normal documents and receipt/invoice output.
ALTER TABLE user_printer_mappings ADD COLUMN IF NOT EXISTS printer_type VARCHAR(50) NOT NULL DEFAULT 'standard';
-- printer_type values: 'standard' (A4/Letter, PostScript), 'receipt' (thermal, narrow paper, ESC/POS)

-- Drop the old unique constraint that only considered org+email+is_default
ALTER TABLE user_printer_mappings DROP CONSTRAINT IF EXISTS unique_user_default_mapping;

-- New unique constraint: one default mapping per user per printer_type per organization
ALTER TABLE user_printer_mappings ADD CONSTRAINT unique_user_default_per_type
    UNIQUE (organization_id, user_email, printer_type, is_default) DEFERRABLE INITIALLY DEFERRED;

CREATE INDEX IF NOT EXISTS idx_user_printer_mappings_type ON user_printer_mappings(printer_type);

COMMENT ON COLUMN user_printer_mappings.printer_type IS 'Printer type: standard (A4/Letter documents) or receipt (thermal/POS printers)';

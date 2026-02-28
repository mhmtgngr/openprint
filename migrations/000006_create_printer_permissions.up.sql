-- OpenPrint Cloud - Printer Permissions Table

CREATE TABLE IF NOT EXISTS printer_permissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    printer_id UUID NOT NULL REFERENCES printers(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    permission_type VARCHAR(50) DEFAULT 'print' CHECK (permission_type IN ('print', 'manage', 'admin')),
    granted_at TIMESTAMPTZ DEFAULT NOW(),
    granted_by UUID REFERENCES users(id),
    UNIQUE(printer_id, user_id)
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_permissions_printer ON printer_permissions(printer_id);
CREATE INDEX IF NOT EXISTS idx_permissions_user ON printer_permissions(user_id);
CREATE INDEX IF NOT EXISTS idx_permissions_granted_by ON printer_permissions(granted_by) WHERE granted_by IS NOT NULL;

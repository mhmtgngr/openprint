-- Printer supply levels (toner, paper, ink, drums, etc.)
CREATE TABLE IF NOT EXISTS printer_supplies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    printer_id UUID NOT NULL,
    supply_type VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    level_percent INTEGER DEFAULT 100 CHECK (level_percent >= 0 AND level_percent <= 100),
    status VARCHAR(50) DEFAULT 'ok',
    part_number VARCHAR(100),
    estimated_pages_remaining INTEGER,
    last_replaced_at TIMESTAMPTZ,
    alert_threshold INTEGER DEFAULT 15,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_supplies_printer ON printer_supplies(printer_id);
CREATE INDEX idx_supplies_low ON printer_supplies(level_percent) WHERE level_percent <= 15;

-- Printer maintenance schedules
CREATE TABLE IF NOT EXISTS printer_maintenance (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    printer_id UUID NOT NULL,
    maintenance_type VARCHAR(100) NOT NULL,
    description TEXT,
    scheduled_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ,
    assigned_to VARCHAR(255),
    status VARCHAR(50) DEFAULT 'scheduled',
    notes TEXT,
    recurrence VARCHAR(50),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_maintenance_printer ON printer_maintenance(printer_id);
CREATE INDEX idx_maintenance_scheduled ON printer_maintenance(scheduled_at) WHERE status = 'scheduled';

-- Print driver packages
CREATE TABLE IF NOT EXISTS print_drivers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    manufacturer VARCHAR(255) NOT NULL,
    model_pattern VARCHAR(255),
    os VARCHAR(50) NOT NULL,
    architecture VARCHAR(20) DEFAULT 'x64',
    version VARCHAR(50) NOT NULL,
    file_path VARCHAR(500),
    file_size_bytes BIGINT,
    checksum_sha256 VARCHAR(64),
    is_universal BOOLEAN DEFAULT false,
    is_latest BOOLEAN DEFAULT true,
    release_notes TEXT,
    uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    uploaded_by UUID
);

CREATE INDEX idx_drivers_model ON print_drivers(manufacturer, model_pattern);
CREATE INDEX idx_drivers_os ON print_drivers(os, architecture);
CREATE INDEX idx_drivers_latest ON print_drivers(is_latest) WHERE is_latest = true;

-- Driver-printer assignments
CREATE TABLE IF NOT EXISTS printer_driver_assignments (
    printer_id UUID NOT NULL,
    driver_id UUID NOT NULL REFERENCES print_drivers(id) ON DELETE CASCADE,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (printer_id, driver_id)
);

-- Supply order history
CREATE TABLE IF NOT EXISTS supply_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    printer_id UUID NOT NULL,
    supply_type VARCHAR(50) NOT NULL,
    part_number VARCHAR(100),
    quantity INTEGER DEFAULT 1,
    order_status VARCHAR(50) DEFAULT 'pending',
    ordered_by UUID,
    ordered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    delivered_at TIMESTAMPTZ,
    notes TEXT
);

CREATE INDEX idx_supply_orders_printer ON supply_orders(printer_id);
CREATE INDEX idx_supply_orders_status ON supply_orders(order_status);

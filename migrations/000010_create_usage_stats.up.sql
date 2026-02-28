-- OpenPrint Cloud - Usage Stats Table

CREATE TABLE IF NOT EXISTS usage_stats (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    printer_id UUID REFERENCES printers(id) ON DELETE SET NULL,
    stat_date DATE NOT NULL,
    pages_printed INTEGER DEFAULT 0,
    color_pages INTEGER DEFAULT 0,
    jobs_count INTEGER DEFAULT 0,
    jobs_completed INTEGER DEFAULT 0,
    jobs_failed INTEGER DEFAULT 0,
    total_bytes BIGINT DEFAULT 0,
    estimated_cost DECIMAL(10,2) DEFAULT 0,
    co2_grams DECIMAL(10,2) DEFAULT 0,
    trees_saved DECIMAL(10,4) DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(org_id, user_id, printer_id, stat_date)
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_usage_stats_date ON usage_stats(stat_date);
CREATE INDEX IF NOT EXISTS idx_usage_stats_org ON usage_stats(org_id, stat_date);
CREATE INDEX IF NOT EXISTS idx_usage_stats_user ON usage_stats(user_id, stat_date) WHERE user_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_usage_stats_printer ON usage_stats(printer_id, stat_date) WHERE printer_id IS NOT NULL;

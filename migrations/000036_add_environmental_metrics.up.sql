-- Migration: 009_add_environmental_metrics
-- Tracks environmental impact of printing for sustainability reporting

CREATE TABLE IF NOT EXISTS environmental_metrics (
    id BIGSERIAL PRIMARY KEY,
    organization_id UUID NOT NULL,
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    total_pages_printed BIGINT DEFAULT 0,
    pages_saved_duplex BIGINT DEFAULT 0,
    pages_saved_nprint BIGINT DEFAULT 0,
    estimated_paper_kg DECIMAL(10, 4) DEFAULT 0,
    estimated_co2_kg DECIMAL(10, 4) DEFAULT 0,
    estimated_trees_saved DECIMAL(10, 4) DEFAULT 0,
    estimated_water_liters DECIMAL(10, 4) DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(organization_id, period_start)
);

CREATE INDEX idx_environmental_metrics_org ON environmental_metrics(organization_id, period_start DESC);

-- Function to calculate carbon footprint for a period
CREATE OR REPLACE FUNCTION calculate_carbon_footprint(
    p_organization_id UUID,
    p_start_date DATE,
    p_end_date DATE
) RETURNS JSONB AS $$
DECLARE
    v_total_pages BIGINT;
    v_duplex_pages BIGINT;
    v_paper_kg DECIMAL;
    v_co2_kg DECIMAL;
    v_trees DECIMAL;
BEGIN
    SELECT COALESCE(SUM(total_pages), 0), COALESCE(SUM(duplex_pages), 0)
    INTO v_total_pages, v_duplex_pages
    FROM print_usage_by_day
    WHERE organization_id = p_organization_id
      AND date BETWEEN p_start_date AND p_end_date;

    -- Estimates: 1 page = 5g paper, 1 kg paper = 1.2 kg CO2, 1 tree = 8333 pages
    v_paper_kg := (v_total_pages * 5.0) / 1000.0;
    v_co2_kg := v_paper_kg * 1.2;
    v_trees := v_total_pages / 8333.0;

    INSERT INTO environmental_metrics (
        organization_id, period_start, period_end,
        total_pages_printed, pages_saved_duplex,
        estimated_paper_kg, estimated_co2_kg, estimated_trees_saved
    ) VALUES (
        p_organization_id, p_start_date, p_end_date,
        v_total_pages, v_duplex_pages,
        v_paper_kg, v_co2_kg, v_trees
    ) ON CONFLICT (organization_id, period_start)
    DO UPDATE SET
        total_pages_printed = EXCLUDED.total_pages_printed,
        pages_saved_duplex = EXCLUDED.pages_saved_duplex,
        estimated_paper_kg = EXCLUDED.estimated_paper_kg,
        estimated_co2_kg = EXCLUDED.estimated_co2_kg,
        estimated_trees_saved = EXCLUDED.estimated_trees_saved,
        updated_at = NOW();

    RETURN jsonb_build_object(
        'total_pages', v_total_pages,
        'duplex_saved', v_duplex_pages,
        'paper_kg', v_paper_kg,
        'co2_kg', v_co2_kg,
        'trees_saved', v_trees
    );
END;
$$ LANGUAGE plpgsql;

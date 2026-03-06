-- Migration: 009_add_environmental_metrics
-- Down migration

DROP TABLE IF EXISTS environmental_metrics CASCADE;

DROP FUNCTION IF EXISTS calculate_carbon_footprint CASCADE;

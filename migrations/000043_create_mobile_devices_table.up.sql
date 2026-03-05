-- Migration: 007_create_mobile_devices_table
-- Tracks print jobs submitted from mobile devices

CREATE TABLE IF NOT EXISTS mobile_print_jobs (
    id BIGSERIAL PRIMARY KEY,
    device_id UUID NOT NULL,
    job_id UUID NOT NULL,
    source_app VARCHAR(100),
    document_name VARCHAR(255),
    submitted_via VARCHAR(50) DEFAULT 'mobile_app',
    location_lat DECIMAL(10, 7),
    location_lon DECIMAL(10, 7),
    nearest_printer_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_mobile_print_jobs_device ON mobile_print_jobs(device_id, created_at DESC);
CREATE INDEX idx_mobile_print_jobs_job ON mobile_print_jobs(job_id);

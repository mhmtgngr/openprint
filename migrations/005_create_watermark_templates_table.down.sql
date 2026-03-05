-- Migration: 005_create_watermark_templates_table
-- Down migration

DROP TABLE IF EXISTS watermark_audit_log CASCADE;

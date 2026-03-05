-- Migration: 006_create_report_schedules_table
-- Down migration

DROP TABLE IF EXISTS report_deliveries CASCADE;
DROP TABLE IF EXISTS report_schedules CASCADE;

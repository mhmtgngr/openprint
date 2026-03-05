-- Migration: 008_create_api_keys_table
-- Down migration

DROP TABLE IF EXISTS api_key_permissions CASCADE;

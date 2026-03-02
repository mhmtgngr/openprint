-- Remove Microsoft 365 Integration Tables

-- Drop functions
DROP FUNCTION IF EXISTS update_m365_last_used(UUID);
DROP FUNCTION IF EXISTS get_user_m365_connection(UUID);
DROP FUNCTION IF EXISTS cleanup_expired_m365_cache();

-- Drop indexes
DROP INDEX IF EXISTS idx_m365_sync_history_started;
DROP INDEX IF EXISTS idx_m365_sync_history_status;
DROP INDEX IF EXISTS idx_m365_sync_history_user;
DROP INDEX IF EXISTS idx_m365_doc_cache_expires;
DROP INDEX IF EXISTS idx_m365_doc_cache_user;
DROP INDEX IF EXISTS idx_m365_doc_cache_source;
DROP INDEX IF EXISTS idx_m365_doc_cache_document;
DROP INDEX IF EXISTS idx_m365_onedrive_synced;
DROP INDEX IF EXISTS idx_m365_onedrive_user;
DROP INDEX IF EXISTS idx_m365_sp_sites_synced;
DROP INDEX IF EXISTS idx_m365_sp_sites_owner;
DROP INDEX IF EXISTS idx_m365_sp_sites_tenant;
DROP INDEX IF EXISTS idx_m365_print_sources_status;
DROP INDEX IF EXISTS idx_m365_print_sources_user;
DROP INDEX IF EXISTS idx_m365_print_sources_type;
DROP INDEX IF EXISTS idx_m365_print_sources_job;
DROP INDEX IF EXISTS idx_m365_connections_last_used;
DROP INDEX IF EXISTS idx_m365_connections_active;
DROP INDEX IF EXISTS idx_m365_connections_tenant;
DROP INDEX IF EXISTS idx_m365_connections_user;

-- Drop tables
DROP TABLE IF EXISTS m365_sync_history;
DROP TABLE IF EXISTS m365_document_cache;
DROP TABLE IF EXISTS m365_onedrive_drives;
DROP TABLE IF EXISTS m365_sharepoint_sites;
DROP TABLE IF EXISTS m365_print_job_sources;
DROP TABLE IF EXISTS m365_connections;

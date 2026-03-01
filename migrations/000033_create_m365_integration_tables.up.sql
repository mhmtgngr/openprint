-- Microsoft 365 Integration Tables
-- Tables for Microsoft 365 (OneDrive, SharePoint) integration

-- Microsoft 365 Connections Table
-- Stores OAuth connections to Microsoft 365 for users
CREATE TABLE IF NOT EXISTS m365_connections (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_email VARCHAR(255) NOT NULL,
    tenant_id VARCHAR(255) NOT NULL DEFAULT 'common',
    access_token TEXT NOT NULL,
    refresh_token TEXT NOT NULL,
    token_expiry TIMESTAMPTZ NOT NULL,
    scopes JSONB DEFAULT '[]'::jsonb,
    connected_at TIMESTAMPTZ DEFAULT NOW(),
    last_used TIMESTAMPTZ,
    is_active BOOLEAN DEFAULT TRUE,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id)
);

CREATE INDEX idx_m365_connections_user ON m365_connections(user_id);
CREATE INDEX idx_m365_connections_tenant ON m365_connections(tenant_id);
CREATE INDEX idx_m365_connections_active ON m365_connections(is_active);
CREATE INDEX idx_m365_connections_last_used ON m365_connections(last_used);

-- Microsoft 365 Print Job Sources Table
-- Tracks print jobs submitted from Microsoft 365 sources
CREATE TABLE IF NOT EXISTS m365_print_job_sources (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_id UUID REFERENCES print_jobs(id) ON DELETE SET NULL,
    source_id VARCHAR(255) NOT NULL,
    source_type VARCHAR(50) NOT NULL CHECK (source_type IN ('onedrive', 'sharepoint', 'outlook', 'teams')),
    document_id VARCHAR(500) NOT NULL,
    document_name VARCHAR(500) NOT NULL,
    document_url TEXT NOT NULL,
    file_size BIGINT,
    mime_type VARCHAR(255),
    user_id UUID REFERENCES users(id),
    user_email VARCHAR(255),
    downloaded_at TIMESTAMPTZ DEFAULT NOW(),
    downloaded_path TEXT,
    file_hash VARCHAR(255),
    download_status VARCHAR(50) DEFAULT 'pending' CHECK (download_status IN ('pending', 'downloading', 'completed', 'failed')),
    error_message TEXT,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_m365_print_sources_job ON m365_print_job_sources(job_id);
CREATE INDEX idx_m365_print_sources_type ON m365_print_job_sources(source_type);
CREATE INDEX idx_m365_print_sources_user ON m365_print_job_sources(user_id);
CREATE INDEX idx_m365_print_sources_status ON m365_print_job_sources(download_status);

-- SharePoint Sites Cache Table
-- Caches SharePoint site information for quick access
CREATE TABLE IF NOT EXISTS m365_sharepoint_sites (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    site_id VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    url TEXT NOT NULL,
    web_url TEXT NOT NULL,
    tenant_id VARCHAR(255) NOT NULL,
    owner_id UUID REFERENCES users(id),
    drive_id VARCHAR(255),
    last_synced_at TIMESTAMPTZ DEFAULT NOW(),
    isAccessible BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_m365_sp_sites_tenant ON m365_sharepoint_sites(tenant_id);
CREATE INDEX idx_m365_sp_sites_owner ON m365_sharepoint_sites(owner_id);
CREATE INDEX idx_m365_sp_sites_synced ON m365_sharepoint_sites(last_synced_at);

-- OneDrive Drives Cache Table
-- Caches OneDrive drive information
CREATE TABLE IF NOT EXISTS m365_onedrive_drives (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    drive_id VARCHAR(255) NOT NULL UNIQUE,
    user_id UUID NOT NULL REFERENCES users(id),
    user_email VARCHAR(255) NOT NULL,
    drive_type VARCHAR(50) NOT NULL, -- 'personal', 'business', 'documentLibrary'
    name VARCHAR(255),
    quota_total BIGINT,
    quota_used BIGINT,
    quota_remaining BIGINT,
    last_synced_at TIMESTAMPTZ DEFAULT NOW(),
    isAccessible BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_m365_onedrive_user ON m365_onedrive_drives(user_id);
CREATE INDEX idx_m365_onedrive_synced ON m365_onedrive_drives(last_synced_at);

-- Microsoft 365 Document Cache Table
-- Caches document metadata for quick listing
CREATE TABLE IF NOT EXISTS m365_document_cache (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    document_id VARCHAR(500) NOT NULL,
    source_type VARCHAR(50) NOT NULL CHECK (source_type IN ('onedrive', 'sharepoint', 'outlook', 'teams')),
    source_id VARCHAR(255) NOT NULL, -- drive_id or site_id
    name VARCHAR(500) NOT NULL,
    path TEXT,
    web_url TEXT,
    download_url TEXT,
    mime_type VARCHAR(255),
    size BIGINT,
    created_by VARCHAR(255),
    created_at TIMESTAMPTZ,
    modified_at TIMESTAMPTZ,
    cached_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ DEFAULT NOW() + INTERVAL '1 hour',
    user_id UUID REFERENCES users(id),
    metadata JSONB DEFAULT '{}'::jsonb
);

CREATE INDEX idx_m365_doc_cache_document ON m365_document_cache(document_id);
CREATE INDEX idx_m365_doc_cache_source ON m365_document_cache(source_type, source_id);
CREATE INDEX idx_m365_doc_cache_user ON m365_document_cache(user_id);
CREATE INDEX idx_m365_doc_cache_expires ON m365_document_cache(expires_at);

-- Microsoft 365 Sync History Table
-- Tracks sync operations for audit and troubleshooting
CREATE TABLE IF NOT EXISTS m365_sync_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    operation VARCHAR(50) NOT NULL, -- 'list_files', 'download_file', 'get_site_info', etc.
    source_type VARCHAR(50),
    source_id VARCHAR(255),
    user_id UUID REFERENCES users(id),
    status VARCHAR(50) NOT NULL CHECK (status IN ('started', 'completed', 'failed', 'cancelled')),
    started_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    items_processed INTEGER DEFAULT 0,
    error_message TEXT,
    metadata JSONB DEFAULT '{}'::jsonb
);

CREATE INDEX idx_m365_sync_history_user ON m365_sync_history(user_id);
CREATE INDEX idx_m365_sync_history_status ON m365_sync_history(status);
CREATE INDEX idx_m365_sync_history_started ON m365_sync_history(started_at);

-- Triggers
CREATE TRIGGER update_m365_connections_updated_at BEFORE UPDATE ON m365_connections
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_m365_sp_sites_updated_at BEFORE UPDATE ON m365_sharepoint_sites
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_m365_onedrive_updated_at BEFORE UPDATE ON m365_onedrive_drives
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to clean expired document cache entries
CREATE OR REPLACE FUNCTION cleanup_expired_m365_cache()
RETURNS INTEGER AS $$
DECLARE
    v_count INTEGER;
BEGIN
    DELETE FROM m365_document_cache
    WHERE expires_at < NOW();

    GET DIAGNOSTICS v_count = ROW_COUNT;
    RETURN v_count;
END;
$$ LANGUAGE plpgsql;

-- Function to get user's M365 connection
CREATE OR REPLACE FUNCTION get_user_m365_connection(p_user_id UUID)
RETURNS TABLE (
    id UUID,
    user_email VARCHAR(255),
    tenant_id VARCHAR(255),
    access_token TEXT,
    refresh_token TEXT,
    token_expiry TIMESTAMPTZ,
    is_active BOOLEAN
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        c.id,
        c.user_email,
        c.tenant_id,
        c.access_token,
        c.refresh_token,
        c.token_expiry,
        c.is_active
    FROM m365_connections c
    WHERE c.user_id = p_user_id
      AND c.is_active = TRUE
    LIMIT 1;
END;
$$ LANGUAGE plpgsql;

-- Function to update last used timestamp
CREATE OR REPLACE FUNCTION update_m365_last_used(p_connection_id UUID)
RETURNS VOID AS $$
BEGIN
    UPDATE m365_connections
    SET last_used = NOW(),
        updated_at = NOW()
    WHERE id = p_connection_id;
END;
$$ LANGUAGE plpgsql;

-- Comments for documentation
COMMENT ON TABLE m365_connections IS 'Stores OAuth connections to Microsoft 365 for users';
COMMENT ON TABLE m365_print_job_sources IS 'Tracks print jobs submitted from Microsoft 365 sources';
COMMENT ON TABLE m365_sharepoint_sites IS 'Caches SharePoint site information';
COMMENT ON TABLE m365_onedrive_drives IS 'Caches OneDrive drive information';
COMMENT ON TABLE m365_document_cache IS 'Caches Microsoft 365 document metadata';
COMMENT ON TABLE m365_sync_history IS 'Tracks Microsoft 365 sync operations for audit';

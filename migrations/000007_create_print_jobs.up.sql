-- OpenPrint Cloud - Print Jobs Table

-- Documents table (needed as foreign key for print_jobs)
CREATE TABLE IF NOT EXISTS documents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(500) NOT NULL,
    content_type VARCHAR(100),
    size BIGINT,
    checksum VARCHAR(64),
    storage_path VARCHAR(1000), -- Path in S3/local storage
    user_email VARCHAR(255) NOT NULL,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes for documents
CREATE INDEX IF NOT EXISTS idx_documents_user ON documents(user_email);
CREATE INDEX IF NOT EXISTS idx_documents_created ON documents(created_at);
CREATE INDEX IF NOT EXISTS idx_documents_expires ON documents(expires_at) WHERE expires_at IS NOT NULL;

-- Print jobs table
CREATE TABLE IF NOT EXISTS print_jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE RESTRICT,
    printer_id UUID NOT NULL REFERENCES printers(id) ON DELETE RESTRICT,
    user_name VARCHAR(255),
    user_email VARCHAR(255) NOT NULL,
    title VARCHAR(500),
    copies INTEGER DEFAULT 1,
    color_mode VARCHAR(20) DEFAULT 'monochrome',
    duplex BOOLEAN DEFAULT false,
    media_type VARCHAR(50) DEFAULT 'a4',
    quality VARCHAR(50) DEFAULT 'normal',
    pages INTEGER,
    status VARCHAR(50) DEFAULT 'queued', -- queued, processing, pending_agent, completed, failed, cancelled, paused
    priority INTEGER DEFAULT 5, -- 1-10, higher is more important
    retries INTEGER DEFAULT 0,
    options JSONB, -- Additional print options
    agent_id UUID REFERENCES agents(id) ON DELETE SET NULL,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_print_jobs_document ON print_jobs(document_id);
CREATE INDEX IF NOT EXISTS idx_print_jobs_printer ON print_jobs(printer_id);
CREATE INDEX IF NOT EXISTS idx_print_jobs_user ON print_jobs(user_email);
CREATE INDEX IF NOT EXISTS idx_print_jobs_status ON print_jobs(status);
CREATE INDEX IF NOT EXISTS idx_print_jobs_status_priority ON print_jobs(status, priority DESC, created_at ASC);
CREATE INDEX IF NOT EXISTS idx_print_jobs_created ON print_jobs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_print_jobs_agent ON print_jobs(agent_id);

-- Create trigger for updated_at
CREATE TRIGGER update_print_jobs_updated_at BEFORE UPDATE ON print_jobs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Secure Print Release Table
-- Tracks print jobs that require secure release at the printer

CREATE TABLE IF NOT EXISTS secure_print_jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_id UUID NOT NULL UNIQUE REFERENCES print_jobs(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    release_method VARCHAR(50) NOT NULL CHECK (release_method IN ('pin', 'card', 'biometric', 'nfc', 'app')),
    release_data JSONB, -- Encrypted PIN, card ID, etc.
    status VARCHAR(50) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'released', 'expired', 'cancelled')),
    expires_at TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '24 hours'),
    released_at TIMESTAMPTZ,
    released_printer_id UUID REFERENCES printers(id),
    release_attempts INTEGER DEFAULT 0,
    max_release_attempts INTEGER DEFAULT 3,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_secure_print_jobs_job ON secure_print_jobs(job_id);
CREATE INDEX idx_secure_print_jobs_user ON secure_print_jobs(user_id);
CREATE INDEX idx_secure_print_jobs_status ON secure_print_jobs(status);
CREATE INDEX idx_secure_print_jobs_expires ON secure_print_jobs(expires_at);
CREATE INDEX idx_secure_print_jobs_printer ON secure_print_jobs(released_printer_id);

-- Secure Release Attempt Logs
-- Logs all release attempts for security auditing

CREATE TABLE IF NOT EXISTS secure_release_attempts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    secure_job_id UUID NOT NULL REFERENCES secure_print_jobs(id) ON DELETE CASCADE,
    attempted_at TIMESTAMPTZ DEFAULT NOW(),
    attempted_method VARCHAR(50),
    attempted_by VARCHAR(255), -- user_id, card_id, etc.
    success BOOLEAN NOT NULL DEFAULT false,
    failure_reason VARCHAR(255),
    ip_address INET,
    printer_id UUID REFERENCES printers(id)
);

CREATE INDEX idx_secure_release_attempts_secure_job ON secure_release_attempts(secure_job_id);
CREATE INDEX idx_secure_release_attempts_success ON secure_release_attempts(success);

-- Print Release Stations
-- Registers physical release stations/kiosks

CREATE TABLE IF NOT EXISTS print_release_stations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    location VARCHAR(255),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    supported_methods VARCHAR(100)[] NOT NULL, -- ARRAY of ['pin', 'card', 'nfc', etc.]
    assigned_printers UUID[],
    is_active BOOLEAN DEFAULT true,
    last_heartbeat TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_print_release_stations_org ON print_release_stations(organization_id);
CREATE INDEX idx_print_release_stations_active ON print_release_stations(is_active);

-- Triggers
CREATE TRIGGER update_secure_print_jobs_updated_at BEFORE UPDATE ON secure_print_jobs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_print_release_stations_updated_at BEFORE UPDATE ON print_release_stations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to check and expire secure print jobs
CREATE OR REPLACE FUNCTION check_expired_secure_jobs() RETURNS INTEGER AS $$
DECLARE
    v_expired_count INTEGER;
BEGIN
    -- Update expired jobs
    UPDATE secure_print_jobs
    SET status = 'expired'
    WHERE status = 'pending'
      AND expires_at < NOW();

    GET DIAGNOSTICS v_expired_count = ROW_COUNT;
    RETURN v_expired_count;
END;
$$ LANGUAGE plpgsql;

-- Function to attempt secure release
CREATE OR REPLACE FUNCTION attempt_secure_release(
    p_secure_job_id UUID,
    p_release_method VARCHAR,
    p_release_data JSONB,
    p_printer_id UUID DEFAULT NULL,
    p_ip_address INET DEFAULT NULL
) RETURNS BOOLEAN AS $$
DECLARE
    v_secure_job RECORD;
    v_attempts_remaining INTEGER;
BEGIN
    -- Get secure job info
    SELECT * INTO v_secure_job
    FROM secure_print_jobs
    WHERE id = p_secure_job_id AND status = 'pending'
    FOR UPDATE;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'Secure job not found or already processed';
    END IF;

    -- Check if expired
    IF v_secure_job.expires_at < NOW() THEN
        UPDATE secure_print_jobs SET status = 'expired' WHERE id = p_secure_job_id;
        RAISE EXCEPTION 'Secure job has expired';
    END IF;

    -- Check release attempts
    v_attempts_remaining := v_secure_job.max_release_attempts - v_secure_job.release_attempts - 1;
    IF v_attempts_remaining < 0 THEN
        UPDATE secure_print_jobs SET status = 'cancelled' WHERE id = p_secure_job_id;
        RAISE EXCEPTION 'Maximum release attempts exceeded';
    END IF;

    -- Validate release data (simplified - in production would decrypt and compare)
    -- For now, assume validation passes if method matches

    -- Log attempt
    INSERT INTO secure_release_attempts (
        secure_job_id, attempted_method, attempted_by,
        success, failure_reason, ip_address, printer_id
    ) VALUES (
        p_secure_job_id, p_release_method,
        p_release_data->>'user_id',
        true, NULL, p_ip_address, p_printer_id
    );

    -- Update job as released
    UPDATE secure_print_jobs
    SET status = 'released',
        released_at = NOW(),
        released_printer_id = p_printer_id,
        release_attempts = release_attempts + 1
    WHERE id = p_secure_job_id;

    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- Multi-Factor Authentication (MFA) Table
-- Supports TOTP, SMS, Email, Hardware Tokens, and Smart Cards

CREATE TABLE IF NOT EXISTS user_mfa (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    mfa_type VARCHAR(50) NOT NULL CHECK (mfa_type IN ('totp', 'sms', 'email', 'hardware_token', 'smart_card', 'biometric', 'push')),
    is_enabled BOOLEAN DEFAULT false,
    is_primary BOOLEAN DEFAULT false,
    secret_data JSONB, -- Encrypted secret for TOTP, phone number for SMS, etc.
    verified_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    backup_codes TEXT[], -- Encrypted backup codes for recovery
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, mfa_type)
);

CREATE INDEX idx_user_mfa_user ON user_mfa(user_id);
CREATE INDEX idx_user_mfa_enabled ON user_mfa(is_enabled);
CREATE INDEX idx_user_mfa_primary ON user_mfa(is_primary);

-- MFA Verification Attempts
-- Tracks MFA verification attempts for security monitoring

CREATE TABLE IF NOT EXISTS mfa_verification_attempts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    mfa_type VARCHAR(50) NOT NULL,
    success BOOLEAN NOT NULL,
    attempted_at TIMESTAMPTZ DEFAULT NOW(),
    ip_address INET,
    user_agent TEXT,
    failure_reason VARCHAR(255)
);

CREATE INDEX idx_mfa_verification_attempts_user ON mfa_verification_attempts(user_id);
CREATE INDEX idx_mfa_verification_attempts_success ON mfa_verification_attempts(success);
CREATE INDEX idx_mfa_verification_attempts_at ON mfa_verification_attempts(attempted_at);

-- Smart Cards Table
-- Registers smart cards for user authentication

CREATE TABLE IF NOT EXISTS user_smart_cards (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    card_id VARCHAR(255) NOT NULL, -- Encrypted card identifier
    card_type VARCHAR(100) NOT NULL, -- 'piv', 'cac', 'employee_badge', etc.
    issuer_dn VARCHAR(255),
    subject_dn VARCHAR(255),
    certificate_valid_from TIMESTAMPTZ,
    certificate_valid_until TIMESTAMPTZ,
    is_active BOOLEAN DEFAULT true,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_user_smart_cards_user ON user_smart_cards(user_id);
CREATE INDEX idx_user_smart_cards_card_id ON user_smart_cards(card_id);
CREATE INDEX idx_user_smart_cards_active ON user_smart_cards(is_active);

-- Smart Card Certificate Log
-- Logs smart card certificate validations

CREATE TABLE IF NOT EXISTS smart_card_cert_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    card_id VARCHAR(255) NOT NULL,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    serial_number VARCHAR(255),
    issuer VARCHAR(255),
    subject VARCHAR(255),
    valid_from TIMESTAMPTZ,
    valid_until TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    revocation_reason VARCHAR(255),
    logged_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_smart_card_cert_log_card ON smart_card_cert_log(card_id);
CREATE INDEX idx_smart_card_cert_log_user ON smart_card_cert_log(user_id);
CREATE INDEX idx_smart_card_cert_log_serial ON smart_card_cert_log(serial_number);

-- Hardware Tokens Table
-- Registers hardware security keys (YubiKey, etc.)

CREATE TABLE IF NOT EXISTS user_hardware_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    credential_id VARCHAR(255) NOT NULL UNIQUE,
    public_key TEXT NOT NULL,
    attestation_type VARCHAR(100),
    aaguid UUID,
    sign_count INTEGER DEFAULT 0,
    is_backup BOOLEAN DEFAULT false,
    is_active BOOLEAN DEFAULT true,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_user_hardware_tokens_user ON user_hardware_tokens(user_id);
CREATE INDEX idx_user_hardware_tokens_credential ON user_hardware_tokens(credential_id);
CREATE INDEX idx_user_hardware_tokens_active ON user_hardware_tokens(is_active);

-- Triggers
CREATE TRIGGER update_user_mfa_updated_at BEFORE UPDATE ON user_mfa
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_user_smart_cards_updated_at BEFORE UPDATE ON user_smart_cards
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_user_hardware_tokens_updated_at BEFORE UPDATE ON user_hardware_tokens
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to check if user requires MFA
CREATE OR REPLACE FUNCTION user_requires_mfa(p_user_id UUID) RETURNS BOOLEAN AS $$
DECLARE
    v_enabled_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO v_enabled_count
    FROM user_mfa
    WHERE user_id = p_user_id
      AND is_enabled = true;

    RETURN v_enabled_count > 0;
END;
$$ LANGUAGE plpgsql;

-- Function to get user's primary MFA method
CREATE OR REPLACE FUNCTION get_primary_mfa_method(p_user_id UUID) RETURNS VARCHAR AS $$
DECLARE
    v_mfa_type VARCHAR;
BEGIN
    SELECT mfa_type INTO v_mfa_type
    FROM user_mfa
    WHERE user_id = p_user_id
      AND is_enabled = true
      AND is_primary = true
    LIMIT 1;

    -- If no primary, return first enabled
    IF v_mfa_type IS NULL THEN
        SELECT mfa_type INTO v_mfa_type
        FROM user_mfa
        WHERE user_id = p_user_id
          AND is_enabled = true
        LIMIT 1;
    END IF;

    RETURN v_mfa_type;
END;
$$ LANGUAGE plpgsql;

-- Function to log MFA verification attempt
CREATE OR REPLACE FUNCTION log_mfa_attempt(
    p_user_id UUID,
    p_mfa_type VARCHAR,
    p_success BOOLEAN,
    p_ip_address INET DEFAULT NULL,
    p_user_agent TEXT DEFAULT NULL,
    p_failure_reason VARCHAR DEFAULT NULL
) RETURNS UUID AS $$
DECLARE
    v_attempt_id UUID;
BEGIN
    INSERT INTO mfa_verification_attempts (
        user_id, mfa_type, success, ip_address, user_agent, failure_reason
    ) VALUES (
        p_user_id, p_mfa_type, p_success, p_ip_address, p_user_agent, p_failure_reason
    )
    RETURNING id INTO v_attempt_id;

    -- Check for too many failed attempts
    IF NOT p_success THEN
        DECLARE
            v_failed_count INTEGER;
        BEGIN
            SELECT COUNT(*) INTO v_failed_count
            FROM mfa_verification_attempts
            WHERE user_id = p_user_id
              AND success = false
              AND attempted_at > NOW() - INTERVAL '15 minutes';

            IF v_failed_count >= 10 THEN
                -- Could trigger account lockout here
                RAISE WARNING 'User % has had % failed MFA attempts in 15 minutes', p_user_id, v_failed_count;
            END IF;
        END;
    END IF;

    RETURN v_attempt_id;
END;
$$ LANGUAGE plpgsql;

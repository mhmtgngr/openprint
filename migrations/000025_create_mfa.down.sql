-- Rollback MFA migration

DROP FUNCTION IF EXISTS log_mfa_attempt CASCADE;
DROP FUNCTION IF EXISTS get_primary_mfa_method CASCADE;
DROP FUNCTION IF EXISTS user_requires_mfa CASCADE;

DROP TABLE IF EXISTS user_hardware_tokens;
DROP TABLE IF EXISTS smart_card_cert_log;
DROP TABLE IF EXISTS user_smart_cards;
DROP TABLE IF EXISTS mfa_verification_attempts;
DROP TABLE IF EXISTS user_mfa;

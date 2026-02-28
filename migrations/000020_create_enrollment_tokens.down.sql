-- Drop enrollment_tokens table
DROP INDEX IF EXISTS idx_enrollment_tokens_revoked_at;
DROP INDEX IF EXISTS idx_enrollment_tokens_expires_at;
DROP INDEX IF EXISTS idx_enrollment_tokens_org_id;
DROP INDEX IF EXISTS idx_enrollment_tokens_token;
DROP TABLE IF EXISTS enrollment_tokens;

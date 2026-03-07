-- Migration: 005_create_watermark_templates_table
-- Audit trail for watermark application

CREATE TABLE IF NOT EXISTS watermark_audit_log (
    id BIGSERIAL PRIMARY KEY,
    template_id UUID,
    job_id UUID,
    user_id UUID,
    organization_id UUID,
    action VARCHAR(50) NOT NULL,
    details JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_watermark_audit_template ON watermark_audit_log(template_id);
CREATE INDEX idx_watermark_audit_job ON watermark_audit_log(job_id);
CREATE INDEX idx_watermark_audit_org ON watermark_audit_log(organization_id, created_at DESC);

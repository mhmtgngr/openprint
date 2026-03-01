-- Remove Compliance Service Tables

-- Drop functions
DROP FUNCTION IF EXISTS get_compliance_summary(VARCHAR);
DROP FUNCTION IF EXISTS get_pending_compliance_reviews(INTEGER);

-- Drop indexes
DROP INDEX IF EXISTS idx_audit_log_compliance_tag;
DROP INDEX IF EXISTS idx_audit_log_retention;
DROP INDEX IF EXISTS idx_compliance_reports_period;
DROP INDEX IF EXISTS idx_compliance_reports_framework;
DROP INDEX IF EXISTS idx_compliance_evidence_type;
DROP INDEX IF EXISTS idx_compliance_evidence_finding;
DROP INDEX IF EXISTS idx_compliance_findings_status;
DROP INDEX IF EXISTS idx_compliance_findings_severity;
DROP INDEX IF EXISTS idx_compliance_findings_control;
DROP INDEX IF EXISTS idx_remediation_plans_target_date;
DROP INDEX IF EXISTS idx_remediation_plans_status;
DROP INDEX IF EXISTS idx_remediation_plans_control;
DROP INDEX IF EXISTS idx_data_breaches_status;
DROP INDEX IF EXISTS idx_data_breaches_severity;
DROP INDEX IF EXISTS idx_data_breaches_discovered;
DROP INDEX IF EXISTS idx_compliance_controls_next_review;
DROP INDEX IF EXISTS idx_compliance_controls_family;
DROP INDEX IF EXISTS idx_compliance_controls_status;
DROP INDEX IF EXISTS idx_compliance_controls_framework;

-- Remove columns from audit_log
ALTER TABLE audit_log
DROP COLUMN IF EXISTS compliance_tag,
DROP COLUMN IF EXISTS retention_date;

-- Drop tables
DROP TABLE IF EXISTS compliance_reports;
DROP TABLE IF EXISTS compliance_evidence;
DROP TABLE IF EXISTS compliance_findings;
DROP TABLE IF EXISTS remediation_plans;
DROP TABLE IF EXISTS data_breaches;
DROP TABLE IF EXISTS compliance_controls;

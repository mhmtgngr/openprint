-- Make user foreign key columns nullable in compliance tables
-- This allows creating compliance records without requiring an associated user,
-- which is needed for system-generated reports and automated compliance tracking.

-- compliance_reports: Make generated_by nullable
ALTER TABLE compliance_reports
    ALTER COLUMN generated_by DROP NOT NULL;

-- compliance_findings: Make created_by and closed_by nullable
ALTER TABLE compliance_findings
    ALTER COLUMN created_by DROP NOT NULL,
    ALTER COLUMN closed_by DROP NOT NULL;

-- compliance_evidence: Make collected_by nullable
ALTER TABLE compliance_evidence
    ALTER COLUMN collected_by DROP NOT NULL;

-- Add comments to document the change
COMMENT ON COLUMN compliance_reports.generated_by IS 'User who generated the report (nullable for system-generated reports)';
COMMENT ON COLUMN compliance_findings.created_by IS 'User who opened the finding (nullable for system-generated findings)';
COMMENT ON COLUMN compliance_findings.closed_by IS 'User who closed the finding (nullable for auto-closed findings)';
COMMENT ON COLUMN compliance_evidence.collected_by IS 'User who collected the evidence (nullable for auto-collected evidence)';

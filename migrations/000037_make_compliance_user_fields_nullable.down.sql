-- Rollback: Make user foreign key columns NOT NULL in compliance tables
-- WARNING: This rollback will fail if there are any NULL values in these columns.
-- Ensure all NULL values are updated before rolling back.

-- compliance_reports: Make generated_by NOT NULL
ALTER TABLE compliance_reports
    ALTER COLUMN generated_by SET NOT NULL;

-- compliance_findings: Make created_by and closed_by NOT NULL
ALTER TABLE compliance_findings
    ALTER COLUMN created_by SET NOT NULL,
    ALTER COLUMN closed_by SET NOT NULL;

-- compliance_evidence: Make collected_by NOT NULL
ALTER TABLE compliance_evidence
    ALTER COLUMN collected_by SET NOT NULL;

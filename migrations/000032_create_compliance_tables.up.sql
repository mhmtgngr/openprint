-- Compliance Service Tables
-- Tables for FedRAMP, HIPAA, GDPR, and SOC2 compliance tracking

-- Compliance Controls Table
CREATE TABLE IF NOT EXISTS compliance_controls (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    framework VARCHAR(20) NOT NULL CHECK (framework IN ('fedramp', 'hipaa', 'gdpr', 'soc2')),
    family VARCHAR(100) NOT NULL,
    title VARCHAR(500) NOT NULL,
    description TEXT,
    implementation TEXT,
    status VARCHAR(30) NOT NULL DEFAULT 'unknown' CHECK (status IN ('compliant', 'non_compliant', 'pending', 'not_applicable', 'unknown')),
    last_assessed TIMESTAMPTZ,
    next_review TIMESTAMPTZ,
    evidence_count INTEGER DEFAULT 0,
    policies JSONB DEFAULT '[]'::jsonb,
    responsible_team VARCHAR(255),
    risk_level VARCHAR(20) DEFAULT 'medium' CHECK (risk_level IN ('low', 'medium', 'high', 'critical')),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(framework, family, id)
);

CREATE INDEX idx_compliance_controls_framework ON compliance_controls(framework);
CREATE INDEX idx_compliance_controls_status ON compliance_controls(status);
CREATE INDEX idx_compliance_controls_family ON compliance_controls(family);
CREATE INDEX idx_compliance_controls_next_review ON compliance_controls(next_review);

-- Data Breaches Table
CREATE TABLE IF NOT EXISTS data_breaches (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    discovered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reported_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    affected_records INTEGER DEFAULT 0,
    data_types JSONB DEFAULT '[]'::jsonb,
    description TEXT,
    containment_status VARCHAR(50) DEFAULT 'identifying',
    notification_sent BOOLEAN DEFAULT FALSE,
    resolved_at TIMESTAMPTZ,
    lessons_learned TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_data_breaches_discovered ON data_breaches(discovered_at);
CREATE INDEX idx_data_breaches_severity ON data_breaches(severity);
CREATE INDEX idx_data_breaches_status ON data_breaches(containment_status);

-- Remediation Plans Table
CREATE TABLE IF NOT EXISTS remediation_plans (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    control_id UUID REFERENCES compliance_controls(id) ON DELETE CASCADE,
    description TEXT NOT NULL,
    priority VARCHAR(20) NOT NULL CHECK (priority IN ('low', 'medium', 'high', 'critical')),
    target_date TIMESTAMPTZ NOT NULL,
    assignee VARCHAR(255),
    status VARCHAR(30) DEFAULT 'open' CHECK (status IN ('open', 'in_progress', 'completed', 'on_hold', 'cancelled')),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_remediation_plans_control ON remediation_plans(control_id);
CREATE INDEX idx_remediation_plans_status ON remediation_plans(status);
CREATE INDEX idx_remediation_plans_target_date ON remediation_plans(target_date);

-- Compliance Findings Table
CREATE TABLE IF NOT EXISTS compliance_findings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    control_id UUID REFERENCES compliance_controls(id) ON DELETE CASCADE,
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('info', 'low', 'medium', 'high', 'critical')),
    title VARCHAR(500) NOT NULL,
    description TEXT,
    recommendation TEXT,
    status VARCHAR(30) DEFAULT 'open' CHECK (status IN ('open', 'in_progress', 'closed', 'deferred')),
    opened_at TIMESTAMPTZ DEFAULT NOW(),
    closed_at TIMESTAMPTZ,
    created_by UUID REFERENCES users(id),
    closed_by UUID REFERENCES users(id)
);

CREATE INDEX idx_compliance_findings_control ON compliance_findings(control_id);
CREATE INDEX idx_compliance_findings_severity ON compliance_findings(severity);
CREATE INDEX idx_compliance_findings_status ON compliance_findings(status);

-- Evidence Items Table
CREATE TABLE IF NOT EXISTS compliance_evidence (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    finding_id UUID REFERENCES compliance_findings(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL, -- 'screenshot', 'document', 'log', 'config', 'other'
    description TEXT,
    file_path TEXT,
    file_hash VARCHAR(255),
    collected_at TIMESTAMPTZ DEFAULT NOW(),
    collected_by UUID REFERENCES users(id),
    metadata JSONB DEFAULT '{}'::jsonb
);

CREATE INDEX idx_compliance_evidence_finding ON compliance_evidence(finding_id);
CREATE INDEX idx_compliance_evidence_type ON compliance_evidence(type);

-- Compliance Reports Table
CREATE TABLE IF NOT EXISTS compliance_reports (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    framework VARCHAR(20) NOT NULL CHECK (framework IN ('fedramp', 'hipaa', 'gdpr', 'soc2')),
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    overall_status VARCHAR(30) NOT NULL CHECK (overall_status IN ('compliant', 'non_compliant', 'pending', 'unknown')),
    compliant_count INTEGER DEFAULT 0,
    non_compliant_count INTEGER DEFAULT 0,
    pending_count INTEGER DEFAULT 0,
    total_controls INTEGER DEFAULT 0,
    high_risk_count INTEGER DEFAULT 0,
    findings JSONB DEFAULT '[]'::jsonb,
    report_hash VARCHAR(255),
    signature TEXT,
    generated_at TIMESTAMPTZ DEFAULT NOW(),
    generated_by UUID REFERENCES users(id)
);

CREATE INDEX idx_compliance_reports_framework ON compliance_reports(framework);
CREATE INDEX idx_compliance_reports_period ON compliance_reports(period_start, period_end);

-- Extend audit_log table if not already present
ALTER TABLE audit_log
ADD COLUMN IF NOT EXISTS retention_date TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS compliance_tag VARCHAR(100);

CREATE INDEX idx_audit_log_retention ON audit_log(retention_date);
CREATE INDEX idx_audit_log_compliance_tag ON audit_log(compliance_tag);

-- Triggers
CREATE TRIGGER update_compliance_controls_updated_at BEFORE UPDATE ON compliance_controls
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_data_breaches_updated_at BEFORE UPDATE ON data_breaches
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_remediation_plans_updated_at BEFORE UPDATE ON remediation_plans
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to get pending compliance reviews
CREATE OR REPLACE FUNCTION get_pending_compliance_reviews(p_within_days INTEGER DEFAULT 30)
RETURNS TABLE (
    control_id UUID,
    framework VARCHAR(20),
    family VARCHAR(100),
    title VARCHAR(500),
    next_review TIMESTAMPTZ,
    days_until INTEGER
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        c.id,
        c.framework,
        c.family,
        c.title,
        c.next_review,
        EXTRACT(DAY FROM (c.next_review - NOW()))::INTEGER AS days_until
    FROM compliance_controls c
    WHERE c.next_review <= NOW() + (p_within_days || ' days')::INTERVAL
    ORDER BY c.next_review ASC;
END;
$$ LANGUAGE plpgsql;

-- Function to calculate compliance summary
CREATE OR REPLACE FUNCTION get_compliance_summary(p_framework VARCHAR DEFAULT NULL)
RETURNS TABLE (
    framework VARCHAR(20),
    compliant INTEGER,
    non_compliant INTEGER,
    pending INTEGER,
    total INTEGER,
    compliance_rate NUMERIC
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        COALESCE(p_framework, framework) AS framework,
        COUNT(*) FILTER (WHERE status = 'compliant')::INTEGER AS compliant,
        COUNT(*) FILTER (WHERE status = 'non_compliant')::INTEGER AS non_compliant,
        COUNT(*) FILTER (WHERE status = 'pending')::INTEGER AS pending,
        COUNT(*)::INTEGER AS total,
        CASE
            WHEN COUNT(*) > 0 THEN
                ROUND(100.0 * COUNT(*) FILTER (WHERE status = 'compliant')::NUMERIC / COUNT(*), 2)
            ELSE 0
        END AS compliance_rate
    FROM compliance_controls
    WHERE p_framework IS NULL OR framework = p_framework
    GROUP BY framework;
END;
$$ LANGUAGE plpgsql;

-- Comments for documentation
COMMENT ON TABLE compliance_controls IS 'Stores compliance control requirements for FedRAMP, HIPAA, GDPR, and SOC2';
COMMENT ON TABLE data_breaches IS 'Tracks data breach incidents for compliance reporting';
COMMENT ON TABLE remediation_plans IS 'Plans for addressing compliance gaps';
COMMENT ON TABLE compliance_findings IS 'Individual findings from compliance assessments';
COMMENT ON TABLE compliance_reports IS 'Generated compliance reports for audit purposes';

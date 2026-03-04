/**
 * Compliance Feature Type Definitions
 * Based on the compliance-service API schema
 */

// Compliance Frameworks
export type ComplianceFramework = 'fedramp' | 'hipaa' | 'gdpr' | 'soc2' | 'all';

// Compliance Status
export type ComplianceStatus =
  | 'compliant'
  | 'non_compliant'
  | 'in_progress'
  | 'pending'
  | 'not_applicable'
  | 'unknown';

// Report Status
export type ReportStatus = 'complete' | 'generating' | 'failed';

// Audit Event Categories
export type AuditCategory =
  | 'authentication'
  | 'authorization'
  | 'data_access'
  | 'data_modification'
  | 'system'
  | 'compliance'
  | 'security';

// Audit Event Outcomes
export type AuditOutcome = 'success' | 'failure' | 'error';

// Containment Status for Data Breaches
export type ContainmentStatus =
  | 'identifying'
  | 'contained'
  | 'eradication'
  | 'recovery'
  | 'closed';

// Risk Levels
export type RiskLevel = 'low' | 'medium' | 'high' | 'critical';

// Checklist Item Status
export type ChecklistStatus = 'pass' | 'fail' | 'warning' | 'pending';

// ============================================================================
// Compliance Control Types
// ============================================================================

export interface ComplianceControl {
  id: string;
  framework: ComplianceFramework;
  family: string;
  title: string;
  description: string;
  implementation: string;
  status: ComplianceStatus;
  last_assessed: string;
  next_review: string;
  responsible_team?: string;
  risk_level?: RiskLevel;
  created_at: string;
  updated_at: string;
}

export interface ComplianceControlsListResponse {
  controls: ComplianceControl[];
  page: number;
  limit: number;
  total: number;
}

export interface CreateControlRequest {
  framework: ComplianceFramework;
  family: string;
  title: string;
  description: string;
  implementation: string;
  status: ComplianceStatus;
  next_review: string;
  responsible_team?: string;
  risk_level?: RiskLevel;
}

export interface UpdateControlStatusRequest {
  status: ComplianceStatus;
  last_assessed?: string;
  next_review?: string;
}

// ============================================================================
// Audit Event Types
// ============================================================================

export interface AuditEvent {
  id: string;
  timestamp: string;
  event_type: string;
  category: AuditCategory;
  user_id: string;
  user_name: string;
  resource_id: string;
  resource_type: string;
  action: string;
  outcome: AuditOutcome;
  ip_address: string;
  details?: Record<string, any>;
}

export interface AuditFilter {
  limit?: number;
  offset?: number;
  start_time?: Date;
  end_time?: Date;
  user_id?: string;
  event_type?: string;
  category?: AuditCategory;
}

export interface AuditEventsResponse {
  events: AuditEvent[];
  limit: number;
  offset: number;
  total: number;
}

// ============================================================================
// Compliance Report Types
// ============================================================================

export interface ComplianceReport {
  id: string;
  framework: ComplianceFramework;
  period_start: string;
  period_end: string;
  overall_status: ComplianceStatus;
  compliant_count: number;
  non_compliant_count: number;
  pending_count: number;
  total_controls: number;
  high_risk_count: number;
  findings: ComplianceFinding[];
  generated_by?: string;
  generated_at: string;
}

export interface ComplianceFinding {
  id: string;
  control_id: string;
  severity: RiskLevel;
  title: string;
  description: string;
  remediation: string;
  status: ComplianceStatus;
  created_at: string;
}

export interface GenerateReportRequest {
  framework: ComplianceFramework;
  period_start: string;
  period_end: string;
  generated_by?: string;
}

// ============================================================================
// Data Breach Types
// ============================================================================

export interface DataBreach {
  id: string;
  title: string;
  description: string;
  severity: RiskLevel;
  affected_records: number;
  discovered_at: string;
  reported_at?: string;
  contained_at?: string;
  containment_status: ContainmentStatus;
  root_cause?: string;
  remediation_steps?: string[];
  reported_by?: string;
  assigned_to?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateDataBreachRequest {
  title: string;
  description: string;
  severity: RiskLevel;
  affected_records: number;
  root_cause?: string;
  remediation_steps?: string[];
}

// ============================================================================
// Compliance Overview Types
// ============================================================================

export interface ComplianceOverview {
  fedramp: FrameworkStatus;
  hipaa: FrameworkStatus;
  gdpr: FrameworkStatus;
  soc2: FrameworkStatus;
  total_logs: number;
  compliant_standards: number;
  pending_actions: number;
}

export interface FrameworkStatus {
  status: ComplianceStatus;
  last_audit: string;
}

export interface ComplianceSummary {
  frameworks: Array<{
    framework: ComplianceFramework;
    status: ComplianceStatus;
  }>;
}

// ============================================================================
// Pending Reviews Types
// ============================================================================

export interface PendingReviewsResponse {
  controls: ComplianceControl[];
  count: number;
  days_ahead: number;
}

// ============================================================================
// Checklist Types
// ============================================================================

export interface ChecklistItem {
  name: string;
  status: ChecklistStatus;
}

export interface ComplianceChecklistResponse {
  checklist: ChecklistItem[];
}

// ============================================================================
// Risk Assessment Types
// ============================================================================

export interface RiskAssessment {
  risk_score: number;
  level: 'low' | 'medium' | 'high';
  mitigations: string[];
}

// ============================================================================
// Data Retention Types
// ============================================================================

export interface DataRetentionPolicy {
  enabled: boolean;
  period_days: number;
  period_unit: 'days' | 'months' | 'years';
}

// ============================================================================
// Security Settings Types
// ============================================================================

export interface SecuritySettings {
  encryption_enabled: boolean;
  encryption_algorithm: 'AES-256' | 'AES-128' | 'ChaCha20';
  two_factor_enabled: boolean;
  session_timeout_minutes: number;
  ip_whitelist: IPWhitelistEntry[];
}

export interface IPWhitelistEntry {
  id?: string;
  ip: string;
  description: string;
}

export interface UpdateSecuritySettingsRequest {
  encryption_enabled?: boolean;
  encryption_algorithm?: string;
  two_factor_enabled?: boolean;
  session_timeout_minutes?: number;
}

// ============================================================================
// API Error Types
// ============================================================================

export interface ComplianceAPIError {
  error: string;
  message?: string;
  details?: Record<string, any>;
}

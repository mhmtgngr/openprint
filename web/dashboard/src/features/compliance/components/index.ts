/**
 * Compliance Components Index
 * Exports all compliance UI components
 */

export { ComplianceStatusBadge } from './ComplianceStatusBadge';
export type { ComplianceStatusBadgeProps } from './ComplianceStatusBadge';

// Export as ComplianceOverviewComponent to avoid name conflict with ComplianceOverview type
export { ComplianceOverviewComponent as ComplianceOverview } from './ComplianceOverview';
export type { ComplianceOverviewProps } from './ComplianceOverview';

export { AuditLogsTable } from './AuditLogsTable';
export type { AuditLogsTableProps } from './AuditLogsTable';

export { ComplianceReports } from './ComplianceReports';
export type { ComplianceReportsProps, GenerateReportParams } from './ComplianceReports';

export { DataRetentionPolicy } from './DataRetentionPolicy';
export type { DataRetentionPolicyProps } from './DataRetentionPolicy';

export { SecuritySettings } from './SecuritySettings';
export type { SecuritySettingsProps } from './SecuritySettings';

export { ComplianceChecklist } from './ComplianceChecklist';
export type { ComplianceChecklistProps } from './ComplianceChecklist';

export { RiskAssessment } from './RiskAssessment';
export type { RiskAssessmentProps } from './RiskAssessment';

export { StatCard } from './StatCard';
export type { StatCardProps } from './StatCard';

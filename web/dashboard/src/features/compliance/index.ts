/**
 * Compliance Feature Module
 * Exports all types, API functions, and hooks for compliance functionality
 */

// Types
export type {
  AuditCategory,
  AuditEvent,
  AuditEventsResponse,
  AuditFilter,
  AuditOutcome,
  ChecklistItem,
  ChecklistStatus,
  ComplianceAPIError,
  ComplianceControl,
  ComplianceControlsListResponse,
  ComplianceFinding,
  ComplianceFramework,
  ComplianceReport,
  ComplianceStatus,
  ComplianceSummary,
  ContainmentStatus,
  CreateControlRequest,
  CreateDataBreachRequest,
  DataBreach,
  FrameworkStatus,
  GenerateReportRequest,
  IPWhitelistEntry,
  PendingReviewsResponse,
  ReportStatus,
  RiskLevel,
  UpdateControlStatusRequest,
} from './types';

// Types that conflict with component names - export with Type suffix
export type { ComplianceOverview as ComplianceOverviewType } from './types';
export type { DataRetentionPolicy as DataRetentionPolicyType } from './types';
export type { SecuritySettings as SecuritySettingsType } from './types';
export type { RiskAssessment as RiskAssessmentType } from './types';

// Components - import from components/index.ts where they are already properly aliased
export {
  ComplianceOverview,
  ComplianceStatusBadge,
  AuditLogsTable,
  ComplianceReports,
  DataRetentionPolicy,
  SecuritySettings,
  ComplianceChecklist,
  RiskAssessment,
  StatCard,
} from './components';

// Component Props
export type {
  ComplianceOverviewProps,
  ComplianceStatusBadgeProps,
  AuditLogsTableProps,
  ComplianceReportsProps,
  DataRetentionPolicyProps,
  SecuritySettingsProps,
  ComplianceChecklistProps,
  RiskAssessmentProps,
  StatCardProps,
} from './components';

// API Functions
export {
  addIPToWhitelist,
  clearAuditLogs,
  createAuditEvent,
  createBreach,
  createControl,
  deleteControl,
  deleteReport,
  downloadReport,
  exportAuditLogs,
  generateReport,
  getBreach,
  getComplianceOverview,
  getComplianceSummary,
  getControl,
  listBreaches,
  listControls,
  listReports,
  queryAuditLogs,
  removeIPFromWhitelist,
  runComplianceChecklist,
  runRiskAssessment,
  updateControlStatus,
  updateBreachStatus,
  getPendingReviews,
  getRetentionPolicy,
  updateRetentionPolicy,
  getSecuritySettings,
  updateSecuritySettings,
} from './api';

// Hooks
export {
  complianceKeys,
  useAddIPToWhitelist,
  useAuditLogs,
  useClearAuditLogs,
  useComplianceControl,
  useComplianceControls,
  useComplianceOverview,
  useComplianceReports,
  useComplianceSummary,
  useCreateAuditEvent,
  useCreateBreach,
  useCreateControl,
  useDataBreach,
  useDataBreaches,
  useDeleteControl,
  useDeleteReport,
  useDownloadReport,
  useExportAuditLogs,
  useGenerateReport,
  usePendingReviews,
  useRemoveIPFromWhitelist,
  useRetentionPolicy,
  useRunComplianceChecklist,
  useRunRiskAssessment,
  useSecuritySettings,
  useUpdateBreachStatus,
  useUpdateControlStatus,
  useUpdateRetentionPolicy,
  useUpdateSecuritySettings,
} from './hooks';

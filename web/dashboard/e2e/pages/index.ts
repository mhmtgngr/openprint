/**
 * Page Objects Index
 * Central export point for all page objects
 */

export { BasePage } from './BasePage';
export { DashboardPage } from './DashboardPage';
export { AuthPage } from './AuthPage';
export { JobsPage } from './JobsPage';
export { PrintersPage } from './PrintersPage';
export { AnalyticsPage } from './AnalyticsPage';
export { SettingsPage } from './SettingsPage';
export { SecurePrintPage } from './SecurePrintPage';
export { QuotasPage } from './QuotasPage';
export { PoliciesPage } from './PoliciesPage';
export { Microsoft365Page } from './Microsoft365Page';
export { CompliancePage } from './CompliancePage';

// Re-export types
export type { PrintPolicy, PolicyCondition, PolicyAction, PolicyScope } from './PoliciesPage';
export type { SecurePrintJob } from './SecurePrintPage';
export type { Quota } from './QuotasPage';
export type { M365Config, M365Document } from './Microsoft365Page';
export type { ComplianceReport, ComplianceCheck } from './CompliancePage';

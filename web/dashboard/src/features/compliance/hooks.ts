/**
 * Compliance Feature React Query Hooks
 * Provides data fetching and mutation hooks for compliance functionality
 */

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
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
import type {
  AuditFilter,
  CreateControlRequest,
  CreateDataBreachRequest,
  GenerateReportRequest,
  UpdateControlStatusRequest,
  DataRetentionPolicy,
  SecuritySettings,
  IPWhitelistEntry,
} from './types';

// Query key factories
export const complianceKeys = {
  all: ['compliance'] as const,
  overview: () => [...complianceKeys.all, 'overview'] as const,
  summary: () => [...complianceKeys.all, 'summary'] as const,
  controls: () => [...complianceKeys.all, 'controls'] as const,
  control: (id: string) => [...complianceKeys.controls(), id] as const,
  auditLogs: () => [...complianceKeys.all, 'auditLogs'] as const,
  reports: () => [...complianceKeys.all, 'reports'] as const,
  report: (id: string) => [...complianceKeys.reports(), id] as const,
  breaches: () => [...complianceKeys.all, 'breaches'] as const,
  breach: (id: string) => [...complianceKeys.breaches(), id] as const,
  pendingReviews: (days: number) => [...complianceKeys.all, 'pendingReviews', days] as const,
  retention: () => [...complianceKeys.all, 'retention'] as const,
  security: () => [...complianceKeys.all, 'security'] as const,
  checklist: () => [...complianceKeys.all, 'checklist'] as const,
  riskAssessment: () => [...complianceKeys.all, 'riskAssessment'] as const,
};

// ============================================================================
// Compliance Overview & Summary Hooks
// ============================================================================

/**
 * Hook to fetch compliance overview for all frameworks
 */
export function useComplianceOverview() {
  return useQuery({
    queryKey: complianceKeys.overview(),
    queryFn: getComplianceOverview,
    staleTime: 5 * 60 * 1000, // 5 minutes
  });
}

/**
 * Hook to fetch compliance summary
 */
export function useComplianceSummary() {
  return useQuery({
    queryKey: complianceKeys.summary(),
    queryFn: getComplianceSummary,
    staleTime: 5 * 60 * 1000,
  });
}

// ============================================================================
// Compliance Controls Hooks
// ============================================================================

/**
 * Hook to fetch compliance controls with optional filtering
 */
export function useComplianceControls(params?: {
  framework?: string;
  status?: string;
  page?: number;
  limit?: number;
}) {
  return useQuery({
    queryKey: [...complianceKeys.controls(), params],
    queryFn: () => listControls(params),
    staleTime: 2 * 60 * 1000,
  });
}

/**
 * Hook to fetch a single compliance control
 */
export function useComplianceControl(id: string, enabled = true) {
  return useQuery({
    queryKey: complianceKeys.control(id),
    queryFn: () => getControl(id),
    enabled: enabled && !!id,
    staleTime: 5 * 60 * 1000,
  });
}

/**
 * Hook to create a new compliance control
 */
export function useCreateControl() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateControlRequest) => createControl(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: complianceKeys.controls() });
    },
  });
}

/**
 * Hook to update compliance control status
 */
export function useUpdateControlStatus() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateControlStatusRequest }) =>
      updateControlStatus(id, data),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: complianceKeys.control(variables.id) });
      queryClient.invalidateQueries({ queryKey: complianceKeys.controls() });
    },
  });
}

/**
 * Hook to delete a compliance control
 */
export function useDeleteControl() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => deleteControl(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: complianceKeys.controls() });
    },
  });
}

// ============================================================================
// Audit Logs Hooks
// ============================================================================

/**
 * Hook to fetch audit logs with filtering
 */
export function useAuditLogs(filter?: AuditFilter) {
  return useQuery({
    queryKey: [...complianceKeys.auditLogs(), filter],
    queryFn: () => queryAuditLogs(filter),
    staleTime: 30 * 1000, // 30 seconds - audit logs change frequently
  });
}

/**
 * Hook to create an audit event
 */
export function useCreateAuditEvent() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (event: Parameters<typeof createAuditEvent>[0]) => createAuditEvent(event),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: complianceKeys.auditLogs() });
    },
  });
}

/**
 * Hook to export audit logs
 */
export function useExportAuditLogs() {
  return useMutation({
    mutationFn: (params: Parameters<typeof exportAuditLogs>[0]) => exportAuditLogs(params),
  });
}

/**
 * Hook to clear old audit logs
 */
export function useClearAuditLogs() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (beforeDate: string) => clearAuditLogs(beforeDate),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: complianceKeys.auditLogs() });
    },
  });
}

// ============================================================================
// Compliance Reports Hooks
// ============================================================================

/**
 * Hook to fetch compliance reports
 */
export function useComplianceReports() {
  return useQuery({
    queryKey: complianceKeys.reports(),
    queryFn: listReports,
    staleTime: 2 * 60 * 1000,
  });
}

/**
 * Hook to generate a compliance report
 */
export function useGenerateReport() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: GenerateReportRequest) => generateReport(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: complianceKeys.reports() });
      queryClient.invalidateQueries({ queryKey: complianceKeys.overview() });
    },
  });
}

/**
 * Hook to download a compliance report
 */
export function useDownloadReport() {
  return useMutation({
    mutationFn: ({ reportId, format }: { reportId: string; format?: 'pdf' | 'json' }) =>
      downloadReport(reportId, format),
  });
}

/**
 * Hook to delete a compliance report
 */
export function useDeleteReport() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (reportId: string) => deleteReport(reportId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: complianceKeys.reports() });
    },
  });
}

// ============================================================================
// Data Breaches Hooks
// ============================================================================

/**
 * Hook to fetch data breaches
 */
export function useDataBreaches() {
  return useQuery({
    queryKey: complianceKeys.breaches(),
    queryFn: listBreaches,
    staleTime: 2 * 60 * 1000,
  });
}

/**
 * Hook to fetch a single data breach
 */
export function useDataBreach(id: string, enabled = true) {
  return useQuery({
    queryKey: complianceKeys.breach(id),
    queryFn: () => getBreach(id),
    enabled: enabled && !!id,
    staleTime: 1 * 60 * 1000,
  });
}

/**
 * Hook to create a data breach record
 */
export function useCreateBreach() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateDataBreachRequest) => createBreach(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: complianceKeys.breaches() });
    },
  });
}

/**
 * Hook to update data breach status
 */
export function useUpdateBreachStatus() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, status, notes }: { id: string; status: string; notes?: string }) =>
      updateBreachStatus(id, status, notes),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: complianceKeys.breach(variables.id) });
      queryClient.invalidateQueries({ queryKey: complianceKeys.breaches() });
    },
  });
}

// ============================================================================
// Pending Reviews Hooks
// ============================================================================

/**
 * Hook to fetch controls pending review
 */
export function usePendingReviews(days: number = 30) {
  return useQuery({
    queryKey: complianceKeys.pendingReviews(days),
    queryFn: () => getPendingReviews(days),
    staleTime: 5 * 60 * 1000,
  });
}

// ============================================================================
// Data Retention Hooks
// ============================================================================

/**
 * Hook to fetch data retention policy
 */
export function useRetentionPolicy() {
  return useQuery({
    queryKey: complianceKeys.retention(),
    queryFn: getRetentionPolicy,
    staleTime: 10 * 60 * 1000,
  });
}

/**
 * Hook to update data retention policy
 */
export function useUpdateRetentionPolicy() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (policy: Partial<DataRetentionPolicy>) => updateRetentionPolicy(policy),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: complianceKeys.retention() });
    },
  });
}

// ============================================================================
// Security Settings Hooks
// ============================================================================

/**
 * Hook to fetch security settings
 */
export function useSecuritySettings() {
  return useQuery({
    queryKey: complianceKeys.security(),
    queryFn: getSecuritySettings,
    staleTime: 10 * 60 * 1000,
  });
}

/**
 * Hook to update security settings
 */
export function useUpdateSecuritySettings() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (settings: Partial<SecuritySettings>) => updateSecuritySettings(settings),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: complianceKeys.security() });
    },
  });
}

/**
 * Hook to add IP to whitelist
 */
export function useAddIPToWhitelist() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (entry: Omit<IPWhitelistEntry, 'id'>) => addIPToWhitelist(entry),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: complianceKeys.security() });
    },
  });
}

/**
 * Hook to remove IP from whitelist
 */
export function useRemoveIPFromWhitelist() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (ip: string) => removeIPFromWhitelist(ip),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: complianceKeys.security() });
    },
  });
}

// ============================================================================
// Compliance Checklist Hooks
// ============================================================================

/**
 * Hook to run compliance checklist
 */
export function useRunComplianceChecklist() {
  return useMutation({
    mutationFn: () => runComplianceChecklist(),
  });
}

// ============================================================================
// Risk Assessment Hooks
// ============================================================================

/**
 * Hook to run risk assessment
 */
export function useRunRiskAssessment() {
  return useMutation({
    mutationFn: () => runRiskAssessment(),
  });
}

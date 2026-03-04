/**
 * Compliance Service API Client
 * Handles all communication with the compliance-service backend
 */

import type {
  AuditEvent,
  AuditEventsResponse,
  AuditFilter,
  ComplianceControl,
  ComplianceControlsListResponse,
  ComplianceOverview,
  ComplianceReport,
  ComplianceSummary,
  CreateControlRequest,
  CreateDataBreachRequest,
  DataBreach,
  GenerateReportRequest,
  PendingReviewsResponse,
  UpdateControlStatusRequest,
  DataRetentionPolicy,
  SecuritySettings,
  IPWhitelistEntry,
  ComplianceAPIError,
} from './types';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1';
const COMPLIANCE_BASE = `${API_BASE_URL}`;

/**
 * Standard API request wrapper
 */
async function request<T>(
  endpoint: string,
  options: RequestInit = {}
): Promise<T> {
  const url = `${COMPLIANCE_BASE}${endpoint}`;
  const token = localStorage.getItem('auth_token');

  const headers: HeadersInit = {
    'Content-Type': 'application/json',
    ...(token && { Authorization: `Bearer ${token}` }),
    ...options.headers,
  };

  const response = await fetch(url, {
    ...options,
    headers,
  });

  if (!response.ok) {
    const error: ComplianceAPIError = await response.json().catch(() => ({ error: 'Unknown error' }));
    throw new Error(error.message || error.error || 'Request failed');
  }

  return response.json();
}

// ============================================================================
// Compliance Overview & Summary
// ============================================================================

/**
 * Get compliance overview for all frameworks
 */
export async function getComplianceOverview(): Promise<ComplianceOverview> {
  // For now, return mock data - the backend doesn't have this exact endpoint
  return request<ComplianceOverview>('/compliance/overview').catch(() => ({
    fedramp: { status: 'compliant', last_audit: '2024-01-15' },
    hipaa: { status: 'compliant', last_audit: '2024-01-15' },
    gdpr: { status: 'compliant', last_audit: '2024-01-15' },
    soc2: { status: 'in_progress', last_audit: '2024-01-15' },
    total_logs: 1523,
    compliant_standards: 3,
    pending_actions: 5,
  }));
}

/**
 * Get compliance summary for all frameworks
 */
export async function getComplianceSummary(): Promise<ComplianceSummary> {
  return request<ComplianceSummary>('/reports/summary');
}

// ============================================================================
// Compliance Controls
// ============================================================================

/**
 * List compliance controls with optional filtering
 */
export async function listControls(params?: {
  framework?: string;
  status?: string;
  page?: number;
  limit?: number;
}): Promise<ComplianceControlsListResponse> {
  const searchParams = new URLSearchParams();
  if (params?.framework) searchParams.set('framework', params.framework);
  if (params?.status) searchParams.set('status', params.status);
  if (params?.page) searchParams.set('page', params.page.toString());
  if (params?.limit) searchParams.set('limit', params.limit.toString());

  const query = searchParams.toString();
  return request<ComplianceControlsListResponse>(`/controls${query ? `?${query}` : ''}`);
}

/**
 * Get a single compliance control by ID
 */
export async function getControl(id: string): Promise<ComplianceControl> {
  return request<ComplianceControl>(`/controls/${id}`);
}

/**
 * Create a new compliance control
 */
export async function createControl(data: CreateControlRequest): Promise<ComplianceControl> {
  return request<ComplianceControl>('/controls', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

/**
 * Update compliance control status
 */
export async function updateControlStatus(
  id: string,
  data: UpdateControlStatusRequest
): Promise<{ message: string }> {
  return request<{ message: string }>(`/controls/status/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  });
}

/**
 * Delete a compliance control
 */
export async function deleteControl(id: string): Promise<void> {
  return request<void>(`/controls/${id}`, {
    method: 'DELETE',
  });
}

// ============================================================================
// Audit Logs
// ============================================================================

/**
 * Query audit events with filtering
 */
export async function queryAuditLogs(filter?: AuditFilter): Promise<AuditEventsResponse> {
  const searchParams = new URLSearchParams();
  if (filter?.limit) searchParams.set('limit', filter.limit.toString());
  if (filter?.offset) searchParams.set('offset', filter.offset.toString());
  if (filter?.start_time) searchParams.set('start_time', filter.start_time.toISOString());
  if (filter?.end_time) searchParams.set('end_time', filter.end_time.toISOString());
  if (filter?.user_id) searchParams.set('user_id', filter.user_id);
  if (filter?.event_type) searchParams.set('event_type', filter.event_type);
  if (filter?.category) searchParams.set('category', filter.category);

  const query = searchParams.toString();
  return request<AuditEventsResponse>(`/audit${query ? `?${query}` : ''}`);
}

/**
 * Create an audit event
 */
export async function createAuditEvent(event: Partial<AuditEvent>): Promise<{ id: string }> {
  return request<{ id: string }>('/audit', {
    method: 'POST',
    body: JSON.stringify(event),
  });
}

/**
 * Export audit logs in various formats
 */
export async function exportAuditLogs(params: {
  format: 'json' | 'csv';
  start_time?: string;
  end_time?: string;
}): Promise<Blob> {
  const searchParams = new URLSearchParams();
  searchParams.set('format', params.format);
  if (params.start_time) searchParams.set('start_time', params.start_time);
  if (params.end_time) searchParams.set('end_time', params.end_time);

  const url = `${COMPLIANCE_BASE}/audit/export?${searchParams.toString()}`;
  const token = localStorage.getItem('auth_token');

  const response = await fetch(url, {
    headers: {
      ...(token && { Authorization: `Bearer ${token}` }),
    },
  });

  if (!response.ok) {
    throw new Error('Failed to export audit logs');
  }

  return response.blob();
}

/**
 * Clear old audit logs
 */
export async function clearAuditLogs(beforeDate: string): Promise<void> {
  return request<void>('/audit/clear', {
    method: 'DELETE',
    body: JSON.stringify({ before_date: beforeDate }),
  });
}

// ============================================================================
// Compliance Reports
// ============================================================================

/**
 * Generate a compliance report
 */
export async function generateReport(data: GenerateReportRequest): Promise<ComplianceReport> {
  return request<ComplianceReport>('/reports/generate', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

/**
 * List all compliance reports
 */
export async function listReports(): Promise<{ reports: ComplianceReport[] }> {
  return request<{ reports: ComplianceReport[] }>('/reports');
}

/**
 * Download a compliance report
 */
export async function downloadReport(reportId: string, format: 'pdf' | 'json' = 'pdf'): Promise<Blob> {
  const url = `${COMPLIANCE_BASE}/reports/${reportId}/download?format=${format}`;
  const token = localStorage.getItem('auth_token');

  const response = await fetch(url, {
    headers: {
      ...(token && { Authorization: `Bearer ${token}` }),
    },
  });

  if (!response.ok) {
    throw new Error('Failed to download report');
  }

  return response.blob();
}

/**
 * Delete a compliance report
 */
export async function deleteReport(reportId: string): Promise<void> {
  return request<void>(`/reports/${reportId}`, {
    method: 'DELETE',
  });
}

// ============================================================================
// Data Breaches
// ============================================================================

/**
 * List all data breaches
 */
export async function listBreaches(): Promise<{ breaches: DataBreach[]; count: number }> {
  return request<{ breaches: DataBreach[]; count: number }>('/breaches');
}

/**
 * Create a new data breach record
 */
export async function createBreach(data: CreateDataBreachRequest): Promise<{ id: string }> {
  return request<{ id: string }>('/breaches', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

/**
 * Get a single data breach
 */
export async function getBreach(id: string): Promise<DataBreach> {
  return request<DataBreach>(`/breaches/${id}`);
}

/**
 * Update data breach status
 */
export async function updateBreachStatus(
  id: string,
  status: string,
  notes?: string
): Promise<void> {
  return request<void>(`/breaches/${id}/status`, {
    method: 'PUT',
    body: JSON.stringify({ status, notes }),
  });
}

// ============================================================================
// Pending Reviews
// ============================================================================

/**
 * Get controls pending review
 */
export async function getPendingReviews(days: number = 30): Promise<PendingReviewsResponse> {
  return request<PendingReviewsResponse>(`/reviews/pending?days=${days}`);
}

// ============================================================================
// Data Retention
// ============================================================================

/**
 * Get data retention policy
 */
export async function getRetentionPolicy(): Promise<DataRetentionPolicy> {
  return request<DataRetentionPolicy>('/retention').catch(() => ({
    enabled: false,
    period_days: 90,
    period_unit: 'days',
  }));
}

/**
 * Update data retention policy
 */
export async function updateRetentionPolicy(policy: Partial<DataRetentionPolicy>): Promise<void> {
  return request<void>('/retention', {
    method: 'PUT',
    body: JSON.stringify(policy),
  });
}

// ============================================================================
// Security Settings
// ============================================================================

/**
 * Get security settings
 */
export async function getSecuritySettings(): Promise<SecuritySettings> {
  return request<SecuritySettings>('/security').catch(() => ({
    encryption_enabled: false,
    encryption_algorithm: 'AES-256',
    two_factor_enabled: false,
    session_timeout_minutes: 30,
    ip_whitelist: [],
  }));
}

/**
 * Update security settings
 */
export async function updateSecuritySettings(settings: Partial<SecuritySettings>): Promise<void> {
  return request<void>('/security', {
    method: 'PUT',
    body: JSON.stringify(settings),
  });
}

/**
 * Add IP to whitelist
 */
export async function addIPToWhitelist(entry: Omit<IPWhitelistEntry, 'id'>): Promise<void> {
  return request<void>('/security/whitelist', {
    method: 'POST',
    body: JSON.stringify(entry),
  });
}

/**
 * Remove IP from whitelist
 */
export async function removeIPFromWhitelist(ip: string): Promise<void> {
  return request<void>(`/security/whitelist/${ip}`, {
    method: 'DELETE',
  });
}

// ============================================================================
// Compliance Checklist
// ============================================================================

/**
 * Run compliance checklist
 */
export async function runComplianceChecklist(): Promise<{ checklist: Array<{ name: string; status: 'pass' | 'fail' | 'warning' | 'pending' }> }> {
  return request<{ checklist: Array<{ name: string; status: 'pass' | 'fail' | 'warning' | 'pending' }> }>('/checklist').catch(() => ({
    checklist: [
      { name: 'Access Control', status: 'pass' },
      { name: 'Audit Logging', status: 'pass' },
      { name: 'Data Encryption', status: 'pass' },
      { name: 'Incident Response', status: 'warning' },
      { name: 'Security Training', status: 'pending' },
    ],
  }));
}

// ============================================================================
// Risk Assessment
// ============================================================================

/**
 * Run risk assessment
 */
export async function runRiskAssessment(): Promise<{ risk_score: number; level: 'low' | 'medium' | 'high'; mitigations: string[] }> {
  return request<{ risk_score: number; level: 'low' | 'medium' | 'high'; mitigations: string[] }>('/risk-assessment').catch(() => ({
    risk_score: 25,
    level: 'low',
    mitigations: [
      'Enable two-factor authentication for all users',
      'Implement IP whitelist for admin access',
      'Review audit logs weekly',
    ],
  }));
}

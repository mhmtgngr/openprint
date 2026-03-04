import type {
  PrintPolicy,
  CreatePolicyRequest,
  PolicyTemplate,
  PolicyEvaluationContext,
  PolicyEvaluationResult,
  PolicyHistoryEntry,
  PolicyTestJob,
} from './types';
import { policiesApi } from '@/services/api';

const API_BASE_URL = import.meta.env.VITE_API_URL || '/api/v1';

/**
 * Fetch with authentication wrapper
 */
async function fetchWithAuth(url: string, options: RequestInit = {}): Promise<Response> {
  const token = localStorage.getItem('auth_tokens')
    ? JSON.parse(localStorage.getItem('auth_tokens')!).accessToken
    : null;

  const headers = {
    'Content-Type': 'application/json',
    ...(token && { Authorization: `Bearer ${token}` }),
    ...options.headers,
  };

  return fetch(url, { ...options, headers });
}

/**
 * Handle API response
 */
async function handleResponse<T>(response: Response): Promise<T> {
  if (!response.ok) {
    const error = await response.json().catch(() => ({
      code: 'unknown_error',
      message: 'An unknown error occurred',
    }));
    throw new Error(error.message || 'API request failed');
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return response.json() as Promise<T>;
}

/**
 * Policy Templates
 */
const POLICY_TEMPLATES: PolicyTemplate[] = [
  {
    id: 'template-color-restriction',
    name: 'Color Restriction',
    description: 'Restrict color printing to specific users or groups',
    category: 'cost_control',
    icon: 'PaletteIcon',
    conditions: [
      { id: 'c1', type: 'userRole', operator: 'equals', value: 'user' },
    ],
    actions: [
      { id: 'a1', type: 'forceGrayscale', value: true },
    ],
  },
  {
    id: 'template-quota-enforcement',
    name: 'Quota Enforcement',
    description: 'Enforce page quotas per user or group',
    category: 'cost_control',
    icon: 'ChartBarIcon',
    conditions: [
      { id: 'c1', type: 'pageCount', operator: 'greaterThan', value: '100' },
    ],
    actions: [
      { id: 'a1', type: 'requireApproval', value: true },
      { id: 'a2', type: 'notify', parameter: 'manager@example.com' },
    ],
  },
  {
    id: 'template-time-based-access',
    name: 'Time-Based Access',
    description: 'Control printer access by time of day',
    category: 'access_control',
    icon: 'ClockIcon',
    conditions: [
      { id: 'c1', type: 'time', operator: 'between', value: '18:00-08:00' },
    ],
    actions: [
      { id: 'a1', type: 'deny' },
    ],
  },
  {
    id: 'template-secure-routing',
    name: 'Secure Document Routing',
    description: 'Route sensitive documents to secure printers only',
    category: 'security',
    icon: 'ShieldCheckIcon',
    conditions: [
      { id: 'c1', type: 'documentType', operator: 'contains', value: 'confidential' },
    ],
    actions: [
      { id: 'a1', type: 'routeToPrinter', parameter: 'secure-printer-1' },
      { id: 'a2', type: 'requireApproval', value: true },
    ],
  },
  {
    id: 'template-default-duplex',
    name: 'Default Duplex',
    description: 'Enable double-sided printing by default',
    category: 'quality',
    icon: 'DocumentDuplicateIcon',
    conditions: [
      { id: 'c1', type: 'always', operator: 'equals', value: 'true' },
    ],
    actions: [
      { id: 'a1', type: 'forceDuplex', value: true },
    ],
  },
  {
    id: 'template-large-job-approval',
    name: 'Large Job Approval',
    description: 'Require approval for print jobs over 50 pages',
    category: 'cost_control',
    icon: 'DocumentTextIcon',
    conditions: [
      { id: 'c1', type: 'pageCount', operator: 'greaterThan', value: '50' },
    ],
    actions: [
      { id: 'a1', type: 'requireApproval', value: true },
    ],
  },
  {
    id: 'template-block-large-files',
    name: 'Block Large Files',
    description: 'Block print jobs larger than 10MB',
    category: 'security',
    icon: 'XCircleIcon',
    conditions: [
      { id: 'c1', type: 'fileSize', operator: 'greaterThan', value: '10485760' },
    ],
    actions: [
      { id: 'a1', type: 'blockJob' },
    ],
  },
  {
    id: 'template-student-restrictions',
    name: 'Student Restrictions',
    description: 'Apply printing restrictions for student users',
    category: 'access_control',
    icon: 'AcademicCapIcon',
    conditions: [
      { id: 'c1', type: 'userRole', operator: 'equals', value: 'student' },
    ],
    actions: [
      { id: 'a1', type: 'forceGrayscale', value: true },
      { id: 'a2', type: 'setCopiesLimit', value: 1 },
    ],
  },
];

/**
 * Extended Policies API with additional methods
 */
export const policyApi = {
  /**
   * List all policies
   */
  async list(): Promise<PrintPolicy[]> {
    return policiesApi.list();
  },

  /**
   * Get policy by ID
   */
  async get(id: string): Promise<PrintPolicy> {
    return policiesApi.get(id);
  },

  /**
   * Create new policy
   */
  async create(data: CreatePolicyRequest): Promise<PrintPolicy> {
    return policiesApi.create(data);
  },

  /**
   * Update policy
   */
  async update(id: string, data: Partial<CreatePolicyRequest>): Promise<PrintPolicy> {
    return policiesApi.update(id, data);
  },

  /**
   * Delete policy
   */
  async delete(id: string): Promise<void> {
    return policiesApi.delete(id);
  },

  /**
   * Toggle policy enabled state
   */
  async toggle(id: string, isEnabled: boolean): Promise<PrintPolicy> {
    return policiesApi.toggle(id, isEnabled);
  },

  /**
   * Reorder policies
   */
  async reorder(policyIds: string[]): Promise<void> {
    return policiesApi.reorder(policyIds);
  },

  /**
   * Get policy templates
   */
  async getTemplates(): Promise<PolicyTemplate[]> {
    // In a real app, this would be an API call
    return Promise.resolve(POLICY_TEMPLATES);
  },

  /**
   * Get policy template by ID
   */
  async getTemplate(id: string): Promise<PolicyTemplate | undefined> {
    const templates = await this.getTemplates();
    return templates.find(t => t.id === id);
  },

  /**
   * Evaluate policy against context
   */
  async evaluate(policyId: string, context: PolicyEvaluationContext): Promise<PolicyEvaluationResult> {
    const response = await fetchWithAuth(`${API_BASE_URL}/policies/${policyId}/evaluate`, {
      method: 'POST',
      body: JSON.stringify(context),
    });
    return handleResponse<PolicyEvaluationResult>(response);
  },

  /**
   * Test multiple policies against a job
   */
  async testJob(job: PolicyTestJob): Promise<{
    results: Array<{ policyId: string; policyName: string; result: PolicyEvaluationResult }>;
  }> {
    const response = await fetchWithAuth(`${API_BASE_URL}/policies/test-job`, {
      method: 'POST',
      body: JSON.stringify(job),
    });
    return handleResponse<{ results: Array<{ policyId: string; policyName: string; result: PolicyEvaluationResult }> }>(response);
  },

  /**
   * Get policy history
   */
  async getHistory(policyId: string): Promise<PolicyHistoryEntry[]> {
    const response = await fetchWithAuth(`${API_BASE_URL}/policies/${policyId}/history`);
    return handleResponse<PolicyHistoryEntry[]>(response);
  },

  /**
   * Restore policy from history
   */
  async restoreFromHistory(policyId: string, versionId: string): Promise<PrintPolicy> {
    const response = await fetchWithAuth(`${API_BASE_URL}/policies/${policyId}/restore`, {
      method: 'POST',
      body: JSON.stringify({ versionId }),
    });
    return handleResponse<PrintPolicy>(response);
  },

  /**
   * Duplicate policy
   */
  async duplicate(id: string, name?: string): Promise<PrintPolicy> {
    const policy = await this.get(id);
    const newPolicy: CreatePolicyRequest = {
      name: name || `${policy.name} (Copy)`,
      description: policy.description,
      conditions: policy.conditions,
      actions: policy.actions,
      appliesTo: policy.appliesTo || 'all',
      targetIds: [],
    };
    return this.create(newPolicy);
  },

  /**
   * Export policy as JSON
   */
  async export(id: string): Promise<Blob> {
    const policy = await this.get(id);
    return new Blob([JSON.stringify(policy, null, 2)], { type: 'application/json' });
  },

  /**
   * Import policy from JSON
   */
  async import(file: File): Promise<PrintPolicy> {
    const text = await file.text();
    const data = JSON.parse(text);
    // Remove id to create a new policy
    delete data.id;
    delete data.createdAt;
    delete data.updatedAt;
    return this.create(data as CreatePolicyRequest);
  },
};

export default policyApi;

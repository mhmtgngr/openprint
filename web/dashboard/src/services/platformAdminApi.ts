/**
 * Platform Admin API - API calls for platform administrator operations
 *
 * Handles multi-tenant management including:
 * - Organization CRUD operations
 * - Quota management
 * - User management across organizations
 * - Usage reporting
 */

import type {
  Organization,
  PlatformAdminOrganizationView,
  CreateOrganizationRequest,
  UpdateOrganizationRequest,
  UpdateQuotaRequest,
  ResourceQuota,
  UsageReport,
  OrganizationUser,
  OrgRole,
  OrganizationInvitation,
  PaginatedOrganizationsResponse,
  OrganizationsListFilters,
  OrganizationAlert,
} from '@/types';

const API_BASE_URL = import.meta.env.VITE_API_URL || '/api/v1';

// ============================================================================
// Auth helpers
// ============================================================================

const getAccessToken = (): string | null => {
  const stored = localStorage.getItem('auth_tokens');
  if (stored) {
    const tokens = JSON.parse(stored) as { accessToken: string; refreshToken: string };
    return tokens.accessToken;
  }
  return null;
};

const fetchWithAuth = async (url: string, options: RequestInit = {}): Promise<Response> => {
  const token = getAccessToken();

  if (token) {
    options.headers = {
      ...options.headers,
      Authorization: `Bearer ${token}`,
    };
  }

  return fetch(url, options);
};

const handleResponse = async <T>(response: Response): Promise<T> => {
  if (!response.ok) {
    const error = await response.json().catch(() => ({
      code: 'unknown_error',
      message: 'An unknown error occurred',
    }));
    throw new Error(error.message || 'Request failed');
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return response.json() as Promise<T>;
};

// ============================================================================
// Organizations API (Platform Admin)
// ============================================================================

export const platformAdminApi = {
  /**
   * List all organizations with filtering and pagination
   */
  async listOrganizations(
    filters?: OrganizationsListFilters,
    limit = 50,
    offset = 0
  ): Promise<PaginatedOrganizationsResponse> {
    const params = new URLSearchParams();
    params.set('limit', limit.toString());
    params.set('offset', offset.toString());

    if (filters?.status) params.set('status', filters.status);
    if (filters?.plan) params.set('plan', filters.plan);
    if (filters?.search) params.set('search', filters.search);
    if (filters?.sortBy) {
      params.set('sortBy', filters.sortBy);
      params.set('sortOrder', filters.sortOrder || 'asc');
    }

    const response = await fetchWithAuth(
      `${API_BASE_URL}/platform/organizations?${params.toString()}`
    );
    return handleResponse<PaginatedOrganizationsResponse>(response);
  },

  /**
   * Get a single organization by ID
   */
  async getOrganization(orgId: string): Promise<PlatformAdminOrganizationView> {
    const response = await fetchWithAuth(
      `${API_BASE_URL}/platform/organizations/${orgId}`
    );
    return handleResponse<PlatformAdminOrganizationView>(response);
  },

  /**
   * Create a new organization
   */
  async createOrganization(data: CreateOrganizationRequest): Promise<Organization> {
    const response = await fetchWithAuth(`${API_BASE_URL}/platform/organizations`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    return handleResponse<Organization>(response);
  },

  /**
   * Update an organization
   */
  async updateOrganization(
    orgId: string,
    data: UpdateOrganizationRequest
  ): Promise<Organization> {
    const response = await fetchWithAuth(
      `${API_BASE_URL}/platform/organizations/${orgId}`,
      {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
      }
    );
    return handleResponse<Organization>(response);
  },

  /**
   * Delete an organization
   */
  async deleteOrganization(orgId: string): Promise<void> {
    const response = await fetchWithAuth(
      `${API_BASE_URL}/platform/organizations/${orgId}`,
      { method: 'DELETE' }
    );
    return handleResponse<void>(response);
  },

  /**
   * Suspend an organization
   */
  async suspendOrganization(orgId: string, reason: string): Promise<Organization> {
    const response = await fetchWithAuth(
      `${API_BASE_URL}/platform/organizations/${orgId}/suspend`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ reason }),
      }
    );
    return handleResponse<Organization>(response);
  },

  /**
   * Reactivate a suspended organization
   */
  async reactivateOrganization(orgId: string): Promise<Organization> {
    const response = await fetchWithAuth(
      `${API_BASE_URL}/platform/organizations/${orgId}/reactivate`,
      { method: 'POST' }
    );
    return handleResponse<Organization>(response);
  },
};

// ============================================================================
// Quota Management API
// ============================================================================

export const quotaApi = {
  /**
   * Get organization quota details
   */
  async getOrganizationQuota(orgId: string): Promise<ResourceQuota> {
    const response = await fetchWithAuth(
      `${API_BASE_URL}/platform/organizations/${orgId}/quota`
    );
    return handleResponse<ResourceQuota>(response);
  },

  /**
   * Update organization quota limits
   */
  async updateQuota(orgId: string, data: UpdateQuotaRequest): Promise<ResourceQuota> {
    const response = await fetchWithAuth(
      `${API_BASE_URL}/platform/organizations/${orgId}/quota`,
      {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
      }
    );
    return handleResponse<ResourceQuota>(response);
  },

  /**
   * Get quota usage across all organizations
   */
  async getAllQuotas(): Promise<Array<{ orgId: string; orgName: string; quota: ResourceQuota }>> {
    const response = await fetchWithAuth(
      `${API_BASE_URL}/platform/quotas`
    );
    return handleResponse<Array<{ orgId: string; orgName: string; quota: ResourceQuota }>>(response);
  },
};

// ============================================================================
// Organization Users API
// ============================================================================

export const orgUsersApi = {
  /**
   * Get all users in an organization
   */
  async getOrganizationUsers(orgId: string): Promise<OrganizationUser[]> {
    const response = await fetchWithAuth(
      `${API_BASE_URL}/platform/organizations/${orgId}/users`
    );
    return handleResponse<OrganizationUser[]>(response);
  },

  /**
   * Add a user to an organization
   */
  async addOrganizationUser(
    orgId: string,
    userId: string,
    role: OrgRole
  ): Promise<OrganizationUser> {
    const response = await fetchWithAuth(
      `${API_BASE_URL}/platform/organizations/${orgId}/users`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ userId, role }),
      }
    );
    return handleResponse<OrganizationUser>(response);
  },

  /**
   * Update user role in organization
   */
  async updateUserRole(
    orgId: string,
    userId: string,
    role: OrgRole
  ): Promise<OrganizationUser> {
    const response = await fetchWithAuth(
      `${API_BASE_URL}/platform/organizations/${orgId}/users/${userId}/role`,
      {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ role }),
      }
    );
    return handleResponse<OrganizationUser>(response);
  },

  /**
   * Remove user from organization
   */
  async removeOrganizationUser(orgId: string, userId: string): Promise<void> {
    const response = await fetchWithAuth(
      `${API_BASE_URL}/platform/organizations/${orgId}/users/${userId}`,
      { method: 'DELETE' }
    );
    return handleResponse<void>(response);
  },

  /**
   * Invite user to organization
   */
  async inviteUser(
    orgId: string,
    email: string,
    role: OrgRole
  ): Promise<OrganizationInvitation> {
    const response = await fetchWithAuth(
      `${API_BASE_URL}/platform/organizations/${orgId}/invitations`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, role }),
      }
    );
    return handleResponse<OrganizationInvitation>(response);
  },

  /**
   * Get pending invitations for organization
   */
  async getInvitations(orgId: string): Promise<OrganizationInvitation[]> {
    const response = await fetchWithAuth(
      `${API_BASE_URL}/platform/organizations/${orgId}/invitations`
    );
    return handleResponse<OrganizationInvitation[]>(response);
  },

  /**
   * Cancel invitation
   */
  async cancelInvitation(orgId: string, invitationId: string): Promise<void> {
    const response = await fetchWithAuth(
      `${API_BASE_URL}/platform/organizations/${orgId}/invitations/${invitationId}`,
      { method: 'DELETE' }
    );
    return handleResponse<void>(response);
  },
};

// ============================================================================
// Usage Reporting API
// ============================================================================

export const usageReportApi = {
  /**
   * Get usage report for an organization
   */
  async getOrganizationUsage(
    orgId: string,
    period: 'daily' | 'weekly' | 'monthly' | 'yearly' = 'monthly',
    startDate?: string,
    endDate?: string
  ): Promise<UsageReport> {
    const params = new URLSearchParams();
    params.set('period', period);
    if (startDate) params.set('startDate', startDate);
    if (endDate) params.set('endDate', endDate);

    const response = await fetchWithAuth(
      `${API_BASE_URL}/platform/organizations/${orgId}/usage?${params.toString()}`
    );
    return handleResponse<UsageReport>(response);
  },

  /**
   * Get aggregated usage across all organizations
   */
  async getPlatformUsage(period: 'daily' | 'weekly' | 'monthly' = 'monthly'): Promise<{
    totalOrganizations: number;
    totalUsers: number;
    totalPrinters: number;
    totalJobs: number;
    totalPages: number;
    totalStorageGB: number;
    topOrganizations: Array<{
      orgId: string;
      orgName: string;
      jobsCount: number;
      pagesCount: number;
    }>;
  }> {
    const response = await fetchWithAuth(
      `${API_BASE_URL}/platform/usage?period=${period}`
    );
    return handleResponse(response);
  },

  /**
   * Get usage trends for an organization
   */
  async getUsageTrends(
    orgId: string,
    period: 'daily' | 'weekly' | 'monthly' = 'monthly',
    lastN = 12
  ): Promise<Array<{
    date: string;
    jobs: number;
    pages: number;
    users: number;
    storage: number;
  }>> {
    const response = await fetchWithAuth(
      `${API_BASE_URL}/platform/organizations/${orgId}/trends?period=${period}&lastN=${lastN}`
    );
    return handleResponse(response);
  },
};

// ============================================================================
// Alerts API
// ============================================================================

export const alertsApi = {
  /**
   * Get alerts for an organization
   */
  async getOrganizationAlerts(orgId: string): Promise<OrganizationAlert[]> {
    const response = await fetchWithAuth(
      `${API_BASE_URL}/platform/organizations/${orgId}/alerts`
    );
    return handleResponse<OrganizationAlert[]>(response);
  },

  /**
   * Get all active platform alerts
   */
  async getAllAlerts(): Promise<OrganizationAlert[]> {
    const response = await fetchWithAuth(
      `${API_BASE_URL}/platform/alerts`
    );
    return handleResponse<OrganizationAlert[]>(response);
  },

  /**
   * Resolve an alert
   */
  async resolveAlert(alertId: string): Promise<void> {
    const response = await fetchWithAuth(
      `${API_BASE_URL}/platform/alerts/${alertId}/resolve`,
      { method: 'POST' }
    );
    return handleResponse<void>(response);
  },
};

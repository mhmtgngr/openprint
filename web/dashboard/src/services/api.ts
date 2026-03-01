import type {
  AuthTokens,
  AuthResponse,
  LoginRequest,
  RegisterRequest,
  User,
  Organization,
  Printer,
  PrintJob,
  CreateJobRequest,
  UsageStats,
  EnvironmentReport,
  UsageAnalyticsParams,
  PaginatedResponse,
  Agent,
  PrinterPermission,
  PermissionType,
  AuditLog,
  Invitation,
  InviteUserRequest,
  UpdateOrganizationRequest,
  UpdateUserRequest,
  Webhook,
  CreateWebhookRequest,
  APIError,
  JobHistoryEntry,
  UserQuota,
  QuotaPeriod,
  UpdateQuotaRequest,
  PrintPolicy,
  CreatePolicyRequest,
  PrintRelease,
  ReleaseJobRequest,
  CreateSecureJobRequest,
  EmailToPrintConfig,
  EmailPrintJob,
  UpdateEmailConfigRequest,
} from '@/types';

const API_BASE_URL = import.meta.env.VITE_API_URL || '/api/v1';

class APIErrorClass extends Error {
  code: string;
  details?: Record<string, unknown>;

  constructor(error: APIError) {
    super(error.message);
    this.name = 'APIError';
    this.code = error.code;
    this.details = error.details;
  }
}

let accessToken: string | null = null;
let refreshToken: string | null = null;

const getAccessToken = (): string | null => {
  if (accessToken) return accessToken;

  const stored = localStorage.getItem('auth_tokens');
  if (stored) {
    const tokens = JSON.parse(stored) as AuthTokens;
    accessToken = tokens.accessToken;
    refreshToken = tokens.refreshToken;
    return accessToken;
  }
  return null;
};

const setTokens = (tokens: AuthTokens): void => {
  accessToken = tokens.accessToken;
  refreshToken = tokens.refreshToken;
  localStorage.setItem('auth_tokens', JSON.stringify(tokens));
};

const clearTokens = (): void => {
  accessToken = null;
  refreshToken = null;
  localStorage.removeItem('auth_tokens');
};

const refreshAccessToken = async (): Promise<boolean> => {
  if (!refreshToken) {
    clearTokens();
    return false;
  }

  try {
    const response = await fetch(`${API_BASE_URL}/auth/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: refreshToken }),
    });

    if (!response.ok) {
      clearTokens();
      return false;
    }

    const data = (await response.json()) as AuthTokens;
    setTokens(data);
    return true;
  } catch {
    clearTokens();
    return false;
  }
};

const fetchWithAuth = async (
  url: string,
  options: RequestInit = {}
): Promise<Response> => {
  let token = getAccessToken();

  if (token) {
    options.headers = {
      ...options.headers,
      Authorization: `Bearer ${token}`,
    };
  }

  let response = await fetch(url, options);

  if (response.status === 401 && token) {
    const refreshed = await refreshAccessToken();
    if (refreshed) {
      token = getAccessToken();
      options.headers = {
        ...options.headers,
        Authorization: `Bearer ${token}`,
      };
      response = await fetch(url, options);
    }
  }

  return response;
};

const handleResponse = async <T>(response: Response): Promise<T> => {
  if (!response.ok) {
    const error = (await response.json().catch(() => ({
      code: 'unknown_error',
      message: 'An unknown error occurred',
    }))) as APIError;
    throw new APIErrorClass(error);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return response.json() as Promise<T>;
};

// Auth API
export const authApi = {
  async login(credentials: LoginRequest): Promise<AuthResponse> {
    const response = await fetch(`${API_BASE_URL}/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(credentials),
    });
    const data = await handleResponse<AuthResponse>(response);
    setTokens({
      accessToken: data.access_token,
      refreshToken: data.refresh_token,
    });
    return data;
  },

  async register(data: RegisterRequest): Promise<AuthResponse> {
    const response = await fetch(`${API_BASE_URL}/auth/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    const result = await handleResponse<AuthResponse>(response);
    setTokens({
      accessToken: result.access_token,
      refreshToken: result.refresh_token,
    });
    return result;
  },

  async logout(): Promise<void> {
    await fetchWithAuth(`${API_BASE_URL}/auth/logout`, { method: 'POST' });
    clearTokens();
  },

  async getCurrentUser(): Promise<User> {
    const response = await fetchWithAuth(`${API_BASE_URL}/auth/me`);
    return handleResponse<User>(response);
  },

  async initiateSSO(provider: string, redirectUri: string): Promise<{ sso_url: string }> {
    const response = await fetch(`${API_BASE_URL}/auth/sso/initiate`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ provider, redirect_uri: redirectUri }),
    });
    return handleResponse<{ sso_url: string }>(response);
  },
};

// Printers API
export const printersApi = {
  async list(): Promise<Printer[]> {
    const response = await fetchWithAuth(`${API_BASE_URL}/printers`);
    const result = await handleResponse<PaginatedResponse<Printer>>(response);
    return result.data || [];
  },

  async get(id: string): Promise<Printer> {
    const response = await fetchWithAuth(`${API_BASE_URL}/printers/${id}`);
    return handleResponse<Printer>(response);
  },

  async update(id: string, data: Partial<Printer>): Promise<Printer> {
    const response = await fetchWithAuth(`${API_BASE_URL}/printers/${id}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    return handleResponse<Printer>(response);
  },

  async delete(id: string): Promise<void> {
    const response = await fetchWithAuth(`${API_BASE_URL}/printers/${id}`, {
      method: 'DELETE',
    });
    return handleResponse<void>(response);
  },

  async getPermissions(printerId: string): Promise<PrinterPermission[]> {
    const response = await fetchWithAuth(
      `${API_BASE_URL}/printers/${printerId}/permissions`
    );
    return handleResponse<PrinterPermission[]>(response);
  },

  async grantPermission(
    printerId: string,
    userId: string,
    permissionType: PermissionType
  ): Promise<PrinterPermission> {
    const response = await fetchWithAuth(
      `${API_BASE_URL}/printers/${printerId}/permissions`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ userId, permissionType }),
      }
    );
    return handleResponse<PrinterPermission>(response);
  },

  async revokePermission(printerId: string, userId: string): Promise<void> {
    const response = await fetchWithAuth(
      `${API_BASE_URL}/printers/${printerId}/permissions/${userId}`,
      { method: 'DELETE' }
    );
    return handleResponse<void>(response);
  },
};

// Jobs API
export const jobsApi = {
  async list(params?: {
    status?: string;
    limit?: number;
    offset?: number;
  }): Promise<PaginatedResponse<PrintJob>> {
    const searchParams = new URLSearchParams();
    if (params?.status) searchParams.set('status', params.status);
    if (params?.limit) searchParams.set('limit', params.limit.toString());
    if (params?.offset) searchParams.set('offset', params.offset.toString());

    const response = await fetchWithAuth(
      `${API_BASE_URL}/jobs?${searchParams.toString()}`
    );
    return handleResponse<PaginatedResponse<PrintJob>>(response);
  },

  async get(id: string): Promise<PrintJob> {
    const response = await fetchWithAuth(`${API_BASE_URL}/jobs/${id}`);
    return handleResponse<PrintJob>(response);
  },

  async create(data: CreateJobRequest): Promise<PrintJob> {
    const response = await fetchWithAuth(`${API_BASE_URL}/jobs`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    return handleResponse<PrintJob>(response);
  },

  async cancel(id: string): Promise<{ success: boolean }> {
    const response = await fetchWithAuth(`${API_BASE_URL}/jobs/${id}/cancel`, {
      method: 'POST',
    });
    return handleResponse<{ success: boolean }>(response);
  },

  async retry(id: string): Promise<PrintJob> {
    const response = await fetchWithAuth(`${API_BASE_URL}/jobs/${id}/retry`, {
      method: 'POST',
    });
    return handleResponse<PrintJob>(response);
  },

  async getHistory(id: string): Promise<JobHistoryEntry[]> {
    const response = await fetchWithAuth(`${API_BASE_URL}/jobs/${id}/history`);
    return handleResponse<JobHistoryEntry[]>(response);
  },
};

// Agents API
export const agentsApi = {
  async list(): Promise<Agent[]> {
    const response = await fetchWithAuth(`${API_BASE_URL}/agents`);
    return handleResponse<Agent[]>(response);
  },

  async get(id: string): Promise<Agent> {
    const response = await fetchWithAuth(`${API_BASE_URL}/agents/${id}`);
    return handleResponse<Agent>(response);
  },

  async update(id: string, data: Partial<Agent>): Promise<Agent> {
    const response = await fetchWithAuth(`${API_BASE_URL}/agents/${id}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    return handleResponse<Agent>(response);
  },

  async delete(id: string): Promise<void> {
    const response = await fetchWithAuth(`${API_BASE_URL}/agents/${id}`, {
      method: 'DELETE',
    });
    return handleResponse<void>(response);
  },
};

// Organization API
export const organizationApi = {
  async get(): Promise<Organization> {
    const response = await fetchWithAuth(`${API_BASE_URL}/organizations`);
    return handleResponse<Organization>(response);
  },

  async update(data: UpdateOrganizationRequest): Promise<Organization> {
    const response = await fetchWithAuth(`${API_BASE_URL}/organizations`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    return handleResponse<Organization>(response);
  },

  async getUsers(): Promise<User[]> {
    const response = await fetchWithAuth(`${API_BASE_URL}/organizations/users`);
    return handleResponse<User[]>(response);
  },

  async inviteUser(data: InviteUserRequest): Promise<Invitation> {
    const response = await fetchWithAuth(`${API_BASE_URL}/organizations/invitations`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    return handleResponse<Invitation>(response);
  },

  async removeUser(userId: string): Promise<void> {
    const response = await fetchWithAuth(
      `${API_BASE_URL}/organizations/users/${userId}`,
      { method: 'DELETE' }
    );
    return handleResponse<void>(response);
  },

  async updateUserRole(userId: string, role: string): Promise<User> {
    const response = await fetchWithAuth(
      `${API_BASE_URL}/organizations/users/${userId}/role`,
      {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ role }),
      }
    );
    return handleResponse<User>(response);
  },

  async getInvitations(): Promise<Invitation[]> {
    const response = await fetchWithAuth(
      `${API_BASE_URL}/organizations/invitations`
    );
    return handleResponse<Invitation[]>(response);
  },

  async cancelInvitation(invitationId: string): Promise<void> {
    const response = await fetchWithAuth(
      `${API_BASE_URL}/organizations/invitations/${invitationId}`,
      { method: 'DELETE' }
    );
    return handleResponse<void>(response);
  },
};

// Analytics API
export const analyticsApi = {
  async getUsage(params?: UsageAnalyticsParams): Promise<UsageStats[]> {
    const searchParams = new URLSearchParams();
    if (params?.startDate) searchParams.set('start_date', params.startDate);
    if (params?.endDate) searchParams.set('end_date', params.endDate);
    if (params?.groupBy) searchParams.set('group_by', params.groupBy);

    const response = await fetchWithAuth(
      `${API_BASE_URL}/analytics/usage?${searchParams.toString()}`
    );
    return handleResponse<UsageStats[]>(response);
  },

  async getEnvironment(period: string = '30d'): Promise<EnvironmentReport> {
    const response = await fetchWithAuth(
      `${API_BASE_URL}/analytics/environment?period=${period}`
    );
    return handleResponse<EnvironmentReport>(response);
  },

  async getAuditLogs(params?: {
    limit?: number;
    offset?: number;
  }): Promise<PaginatedResponse<AuditLog>> {
    const searchParams = new URLSearchParams();
    if (params?.limit) searchParams.set('limit', params.limit.toString());
    if (params?.offset) searchParams.set('offset', params.offset.toString());

    const response = await fetchWithAuth(
      `${API_BASE_URL}/analytics/audit-logs?${searchParams.toString()}`
    );
    return handleResponse<PaginatedResponse<AuditLog>>(response);
  },
};

// Webhooks API
export const webhooksApi = {
  async list(): Promise<Webhook[]> {
    const response = await fetchWithAuth(`${API_BASE_URL}/webhooks`);
    return handleResponse<Webhook[]>(response);
  },

  async create(data: CreateWebhookRequest): Promise<Webhook> {
    const response = await fetchWithAuth(`${API_BASE_URL}/webhooks`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    return handleResponse<Webhook>(response);
  },

  async update(id: string, data: Partial<Webhook>): Promise<Webhook> {
    const response = await fetchWithAuth(`${API_BASE_URL}/webhooks/${id}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    return handleResponse<Webhook>(response);
  },

  async delete(id: string): Promise<void> {
    const response = await fetchWithAuth(`${API_BASE_URL}/webhooks/${id}`, {
      method: 'DELETE',
    });
    return handleResponse<void>(response);
  },

  async test(id: string): Promise<{ success: boolean }> {
    const response = await fetchWithAuth(`${API_BASE_URL}/webhooks/${id}/test`, {
      method: 'POST',
    });
    return handleResponse<{ success: boolean }>(response);
  },
};

// User API
export const userApi = {
  async update(data: UpdateUserRequest): Promise<User> {
    const response = await fetchWithAuth(`${API_BASE_URL}/users/me`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    return handleResponse<User>(response);
  },

  async changePassword(currentPassword: string, newPassword: string): Promise<void> {
    const response = await fetchWithAuth(`${API_BASE_URL}/users/me/password`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ currentPassword, newPassword }),
    });
    return handleResponse<void>(response);
  },
};

export { setTokens, clearTokens, getAccessToken };
export type { APIErrorClass as APIError };

// Quotas API
export const quotasApi = {
  async getUserQuota(userId?: string): Promise<UserQuota> {
    const url = userId
      ? `${API_BASE_URL}/quotas/users/${userId}`
      : `${API_BASE_URL}/quotas/me`;
    const response = await fetchWithAuth(url);
    return handleResponse<UserQuota>(response);
  },

  async getOrgQuotas(): Promise<UserQuota[]> {
    const response = await fetchWithAuth(`${API_BASE_URL}/quotas/organization`);
    return handleResponse<UserQuota[]>(response);
  },

  async updateQuota(data: UpdateQuotaRequest): Promise<UserQuota> {
    const response = await fetchWithAuth(`${API_BASE_URL}/quotas`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    return handleResponse<UserQuota>(response);
  },

  async getPeriods(params?: { limit?: number; offset?: number }): Promise<PaginatedResponse<QuotaPeriod>> {
    const searchParams = new URLSearchParams();
    if (params?.limit) searchParams.set('limit', params.limit.toString());
    if (params?.offset) searchParams.set('offset', params.offset.toString());

    const response = await fetchWithAuth(
      `${API_BASE_URL}/quotas/periods?${searchParams.toString()}`
    );
    return handleResponse<PaginatedResponse<QuotaPeriod>>(response);
  },
};

// Policies API
export const policiesApi = {
  async list(): Promise<PrintPolicy[]> {
    const response = await fetchWithAuth(`${API_BASE_URL}/policies`);
    return handleResponse<PrintPolicy[]>(response);
  },

  async get(id: string): Promise<PrintPolicy> {
    const response = await fetchWithAuth(`${API_BASE_URL}/policies/${id}`);
    return handleResponse<PrintPolicy>(response);
  },

  async create(data: CreatePolicyRequest): Promise<PrintPolicy> {
    const response = await fetchWithAuth(`${API_BASE_URL}/policies`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    return handleResponse<PrintPolicy>(response);
  },

  async update(id: string, data: Partial<CreatePolicyRequest>): Promise<PrintPolicy> {
    const response = await fetchWithAuth(`${API_BASE_URL}/policies/${id}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    return handleResponse<PrintPolicy>(response);
  },

  async delete(id: string): Promise<void> {
    const response = await fetchWithAuth(`${API_BASE_URL}/policies/${id}`, {
      method: 'DELETE',
    });
    return handleResponse<void>(response);
  },

  async toggle(id: string, isEnabled: boolean): Promise<PrintPolicy> {
    const response = await fetchWithAuth(`${API_BASE_URL}/policies/${id}/toggle`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ isEnabled }),
    });
    return handleResponse<PrintPolicy>(response);
  },

  async reorder(policyIds: string[]): Promise<void> {
    const response = await fetchWithAuth(`${API_BASE_URL}/policies/reorder`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ policyIds }),
    });
    return handleResponse<void>(response);
  },
};

// Print Release API
export const printReleaseApi = {
  async createSecureJob(data: CreateSecureJobRequest): Promise<PrintJob> {
    const response = await fetchWithAuth(`${API_BASE_URL}/jobs/secure`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    return handleResponse<PrintJob>(response);
  },

  async getPendingReleases(): Promise<PrintRelease[]> {
    const response = await fetchWithAuth(`${API_BASE_URL}/releases/pending`);
    return handleResponse<PrintRelease[]>(response);
  },

  async releaseJob(data: ReleaseJobRequest): Promise<PrintJob> {
    const response = await fetchWithAuth(`${API_BASE_URL}/releases/release`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    return handleResponse<PrintJob>(response);
  },

  async cancelRelease(jobId: string): Promise<void> {
    const response = await fetchWithAuth(`${API_BASE_URL}/releases/${jobId}`, {
      method: 'DELETE',
    });
    return handleResponse<void>(response);
  },
};

// Email-to-Print API
export const emailToPrintApi = {
  async getConfig(): Promise<EmailToPrintConfig> {
    const response = await fetchWithAuth(`${API_BASE_URL}/email-to-print/config`);
    return handleResponse<EmailToPrintConfig>(response);
  },

  async updateConfig(data: UpdateEmailConfigRequest): Promise<EmailToPrintConfig> {
    const response = await fetchWithAuth(`${API_BASE_URL}/email-to-print/config`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    return handleResponse<EmailToPrintConfig>(response);
  },

  async getJobs(params?: {
    status?: string;
    limit?: number;
    offset?: number;
  }): Promise<PaginatedResponse<EmailPrintJob>> {
    const searchParams = new URLSearchParams();
    if (params?.status) searchParams.set('status', params.status);
    if (params?.limit) searchParams.set('limit', params.limit.toString());
    if (params?.offset) searchParams.set('offset', params.offset.toString());

    const response = await fetchWithAuth(
      `${API_BASE_URL}/email-to-print/jobs?${searchParams.toString()}`
    );
    return handleResponse<PaginatedResponse<EmailPrintJob>>(response);
  },

  async testEmail(): Promise<{ success: boolean; email: string }> {
    const response = await fetchWithAuth(`${API_BASE_URL}/email-to-print/test`, {
      method: 'POST',
    });
    return handleResponse<{ success: boolean; email: string }>(response);
  },
};

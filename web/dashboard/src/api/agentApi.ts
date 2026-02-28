/**
 * API client for agent-related operations
 * Communicates with registry-service and job-service endpoints
 */

import type {
  Agent,
  AgentDetail,
  AgentListParams,
  DiscoveredPrinter,
  JobAssignment,
  JobAssignmentRequest,
} from '@/types/agents';
import type { PaginatedResponse } from '@/types';

const API_BASE_URL = import.meta.env.VITE_API_URL || '/api/v1';

const getAccessToken = (): string | null => {
  const stored = localStorage.getItem('auth_tokens');
  if (stored) {
    const tokens = JSON.parse(stored) as { accessToken: string };
    return tokens.accessToken;
  }
  return null;
};

const fetchWithAuth = async (
  url: string,
  options: RequestInit = {}
): Promise<Response> => {
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
    const error = (await response.json().catch(() => ({
      code: 'unknown_error',
      message: 'An unknown error occurred',
    }))) as { code: string; message: string; details?: Record<string, unknown> };
    throw new Error(error.message || 'API request failed');
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return response.json() as Promise<T>;
};

// Agents API
export const agentApi = {
  /**
   * List all agents with optional filtering
   * GET /api/v1/agents
   */
  async list(params?: AgentListParams): Promise<Agent[]> {
    const searchParams = new URLSearchParams();
    if (params?.status && params.status !== 'all') {
      searchParams.set('status', params.status);
    }
    if (params?.userId) {
      searchParams.set('user_id', params.userId);
    }
    if (params?.search) {
      searchParams.set('search', params.search);
    }
    if (params?.limit) {
      searchParams.set('limit', params.limit.toString());
    }
    if (params?.offset) {
      searchParams.set('offset', params.offset.toString());
    }

    const queryString = searchParams.toString();
    const response = await fetchWithAuth(
      `${API_BASE_URL}/agents${queryString ? `?${queryString}` : ''}`
    );
    return handleResponse<Agent[]>(response);
  },

  /**
   * Get detailed information about a specific agent
   * GET /api/v1/agents/:id
   */
  async getDetail(id: string): Promise<AgentDetail> {
    const response = await fetchWithAuth(`${API_BASE_URL}/agents/${id}/detail`);
    return handleResponse<AgentDetail>(response);
  },

  /**
   * Get a single agent by ID
   * GET /api/v1/agents/:id
   */
  async get(id: string): Promise<Agent> {
    const response = await fetchWithAuth(`${API_BASE_URL}/agents/${id}`);
    return handleResponse<Agent>(response);
  },

  /**
   * Update agent information
   * PATCH /api/v1/agents/:id
   */
  async update(id: string, data: Partial<Agent>): Promise<Agent> {
    const response = await fetchWithAuth(`${API_BASE_URL}/agents/${id}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    return handleResponse<Agent>(response);
  },

  /**
   * Delete/unregister an agent
   * DELETE /api/v1/agents/:id
   */
  async delete(id: string): Promise<void> {
    const response = await fetchWithAuth(`${API_BASE_URL}/agents/${id}`, {
      method: 'DELETE',
    });
    return handleResponse<void>(response);
  },

  /**
   * Get printers discovered by a specific agent
   * GET /api/v1/agents/:id/printers
   */
  async getPrinters(id: string): Promise<DiscoveredPrinter[]> {
    const response = await fetchWithAuth(`${API_BASE_URL}/agents/${id}/printers`);
    return handleResponse<DiscoveredPrinter[]>(response);
  },

  /**
   * Get agent health metrics
   * GET /api/v1/agents/:id/health
   */
  async getHealth(id: string): Promise<AgentDetail['healthMetrics']> {
    const response = await fetchWithAuth(`${API_BASE_URL}/agents/${id}/health`);
    return handleResponse<AgentDetail['healthMetrics']>(response);
  },

  /**
   * Get agent job history
   * GET /api/v1/agents/:id/jobs
   */
  async getJobHistory(
    id: string,
    params?: { limit?: number; offset?: number }
  ): Promise<AgentDetail['jobHistory']> {
    const searchParams = new URLSearchParams();
    if (params?.limit) {
      searchParams.set('limit', params.limit.toString());
    }
    if (params?.offset) {
      searchParams.set('offset', params.offset.toString());
    }

    const queryString = searchParams.toString();
    const response = await fetchWithAuth(
      `${API_BASE_URL}/agents/${id}/jobs${queryString ? `?${queryString}` : ''}`
    );
    return handleResponse<AgentDetail['jobHistory']>(response);
  },

  /**
   * Trigger printer discovery on an agent
   * POST /api/v1/agents/:id/discover
   */
  async triggerDiscovery(id: string): Promise<{ printers: DiscoveredPrinter[] }> {
    const response = await fetchWithAuth(`${API_BASE_URL}/agents/${id}/discover`, {
      method: 'POST',
    });
    return handleResponse<{ printers: DiscoveredPrinter[] }>(response);
  },

  /**
   * Restart an agent
   * POST /api/v1/agents/:id/restart
   */
  async restart(id: string): Promise<{ success: boolean }> {
    const response = await fetchWithAuth(`${API_BASE_URL}/agents/${id}/restart`, {
      method: 'POST',
    });
    return handleResponse<{ success: boolean }>(response);
  },
};

// Discovered Printers API
export const discoveredPrintersApi = {
  /**
   * List all discovered printers across all agents
   * GET /api/v1/discovered-printers
   */
  async list(params?: {
    agentId?: string;
    status?: string;
    search?: string;
    limit?: number;
    offset?: number;
  }): Promise<{ printers: DiscoveredPrinter[]; total: number }> {
    const searchParams = new URLSearchParams();
    if (params?.agentId) {
      searchParams.set('agent_id', params.agentId);
    }
    if (params?.status) {
      searchParams.set('status', params.status);
    }
    if (params?.search) {
      searchParams.set('search', params.search);
    }
    if (params?.limit) {
      searchParams.set('limit', params.limit.toString());
    }
    if (params?.offset) {
      searchParams.set('offset', params.offset.toString());
    }

    const queryString = searchParams.toString();
    const response = await fetchWithAuth(
      `${API_BASE_URL}/discovered-printers${queryString ? `?${queryString}` : ''}`
    );
    return handleResponse<{ printers: DiscoveredPrinter[]; total: number }>(response);
  },

  /**
   * Get a single discovered printer by ID
   * GET /api/v1/discovered-printers/:id
   */
  async get(id: string): Promise<DiscoveredPrinter> {
    const response = await fetchWithAuth(`${API_BASE_URL}/discovered-printers/${id}`);
    return handleResponse<DiscoveredPrinter>(response);
  },

  /**
   * Update discovered printer information
   * PATCH /api/v1/discovered-printers/:id
   */
  async update(
    id: string,
    data: Partial<DiscoveredPrinter>
  ): Promise<DiscoveredPrinter> {
    const response = await fetchWithAuth(`${API_BASE_URL}/discovered-printers/${id}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    return handleResponse<DiscoveredPrinter>(response);
  },

  /**
   * Delete a discovered printer
   * DELETE /api/v1/discovered-printers/:id
   */
  async delete(id: string): Promise<void> {
    const response = await fetchWithAuth(`${API_BASE_URL}/discovered-printers/${id}`, {
      method: 'DELETE',
    });
    return handleResponse<void>(response);
  },

  /**
   * Set a printer as default for its agent
   * POST /api/v1/discovered-printers/:id/set-default
   */
  async setDefault(id: string): Promise<DiscoveredPrinter> {
    const response = await fetchWithAuth(
      `${API_BASE_URL}/discovered-printers/${id}/set-default`,
      { method: 'POST' }
    );
    return handleResponse<DiscoveredPrinter>(response);
  },
};

// Job Assignments API
export const jobAssignmentsApi = {
  /**
   * List all job assignments
   * GET /api/v1/job-assignments
   */
  async list(params?: {
    status?: string;
    agentId?: string;
    userId?: string;
    limit?: number;
    offset?: number;
  }): Promise<PaginatedResponse<JobAssignment>> {
    const searchParams = new URLSearchParams();
    if (params?.status) {
      searchParams.set('status', params.status);
    }
    if (params?.agentId) {
      searchParams.set('agent_id', params.agentId);
    }
    if (params?.userId) {
      searchParams.set('user_id', params.userId);
    }
    if (params?.limit) {
      searchParams.set('limit', params.limit.toString());
    }
    if (params?.offset) {
      searchParams.set('offset', params.offset.toString());
    }

    const queryString = searchParams.toString();
    const response = await fetchWithAuth(
      `${API_BASE_URL}/job-assignments${queryString ? `?${queryString}` : ''}`
    );
    return handleResponse<PaginatedResponse<JobAssignment>>(response);
  },

  /**
   * Create a new job assignment
   * POST /api/v1/job-assignments
   */
  async create(data: JobAssignmentRequest): Promise<JobAssignment> {
    const response = await fetchWithAuth(`${API_BASE_URL}/job-assignments`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    return handleResponse<JobAssignment>(response);
  },

  /**
   * Get a single job assignment by ID
   * GET /api/v1/job-assignments/:id
   */
  async get(id: string): Promise<JobAssignment> {
    const response = await fetchWithAuth(`${API_BASE_URL}/job-assignments/${id}`);
    return handleResponse<JobAssignment>(response);
  },

  /**
   * Update a job assignment
   * PATCH /api/v1/job-assignments/:id
   */
  async update(
    id: string,
    data: Partial<JobAssignmentRequest>
  ): Promise<JobAssignment> {
    const response = await fetchWithAuth(`${API_BASE_URL}/job-assignments/${id}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    return handleResponse<JobAssignment>(response);
  },

  /**
   * Cancel a job assignment
   * DELETE /api/v1/job-assignments/:id
   */
  async cancel(id: string): Promise<void> {
    const response = await fetchWithAuth(`${API_BASE_URL}/job-assignments/${id}`, {
      method: 'DELETE',
    });
    return handleResponse<void>(response);
  },

  /**
   * Reassign a job to a different agent/user
   * POST /api/v1/job-assignments/:id/reassign
   */
  async reassign(
    id: string,
    data: { agentId?: string; userId?: string; printerId?: string }
  ): Promise<JobAssignment> {
    const response = await fetchWithAuth(
      `${API_BASE_URL}/job-assignments/${id}/reassign`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
      }
    );
    return handleResponse<JobAssignment>(response);
  },
};

// Utility function to format agent uptime
export function formatUptime(seconds: number): string {
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);

  if (days > 0) {
    return `${days}d ${hours}h`;
  } else if (hours > 0) {
    return `${hours}h ${minutes}m`;
  } else {
    return `${minutes}m`;
  }
}

// Utility function to get status color for charts
export function getAgentStatusColor(status: string): string {
  switch (status) {
    case 'online':
      return '#22c55e';
    case 'offline':
      return '#9ca3af';
    case 'error':
      return '#ef4444';
    default:
      return '#6b7280';
  }
}

// Utility function to get printer type color
export function getPrinterTypeColor(type: string): string {
  switch (type) {
    case 'local':
      return '#3b82f6';
    case 'network':
      return '#8b5cf6';
    case 'shared':
      return '#f59e0b';
    default:
      return '#6b7280';
  }
}

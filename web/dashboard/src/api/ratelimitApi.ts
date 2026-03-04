/**
 * API client for Rate Limiting and Throttling System
 *
 * Communicates with auth-service endpoints for:
 * - Rate limit policy management
 * - Violation log viewing
 * - Circuit breaker control
 * - Trusted client management
 * - Rate limit metrics
 */

import type {
  RateLimitPolicy,
  CreateRateLimitPolicyRequest,
  UpdateRateLimitPolicyRequest,
  RateLimitPolicyFilters,
  RateLimitPoliciesResponse,
  RateLimitViolation,
  ViolationFilters,
  RateLimitViolationsResponse,
  ViolationStats,
  CircuitBreaker,
  CreateCircuitBreakerRequest,
  UpdateCircuitBreakerRequest,
  CircuitBreakersResponse,
  CircuitBreakerTransition,
  CircuitBreakerState,
  TrustedClient,
  CreateTrustedClientRequest,
  UpdateTrustedClientRequest,
  TrustedClientsResponse,
  RateLimitMetrics,
  RateLimitSummary,
  RateLimitHistoryPoint,
  QueuedRequest,
  QueuedRequestsResponse,
  QueueStats,
} from '@/types/ratelimit';

const API_BASE_URL = import.meta.env.VITE_API_URL || '/api/v1';

// ============================================================================
// Authentication Helpers
// ============================================================================

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

// ============================================================================
// Rate Limit Policy API
// ============================================================================

export const rateLimitPoliciesApi = {
  /**
   * List all rate limit policies with filtering
   * GET /api/v1/rate-limits/policies
   */
  async list(
    filters?: RateLimitPolicyFilters,
    limit = 50,
    offset = 0
  ): Promise<RateLimitPoliciesResponse> {
    const searchParams = new URLSearchParams();
    if (filters?.scope) searchParams.set('scope', filters.scope);
    if (filters?.dimension) searchParams.set('dimension', filters.dimension);
    if (filters?.status) searchParams.set('status', filters.status);
    if (filters?.endpoint) searchParams.set('endpoint', filters.endpoint);
    if (filters?.search) searchParams.set('search', filters.search);
    if (filters?.sortBy) searchParams.set('sort_by', filters.sortBy);
    if (filters?.sortOrder) searchParams.set('sort_order', filters.sortOrder);
    searchParams.set('limit', limit.toString());
    searchParams.set('offset', offset.toString());

    const queryString = searchParams.toString();
    const response = await fetchWithAuth(
      `${API_BASE_URL}/rate-limits/policies${queryString ? `?${queryString}` : ''}`
    );
    return handleResponse<RateLimitPoliciesResponse>(response);
  },

  /**
   * Get a single rate limit policy by ID
   * GET /api/v1/rate-limits/policies/:id
   */
  async get(id: string): Promise<RateLimitPolicy> {
    const response = await fetchWithAuth(`${API_BASE_URL}/rate-limits/policies/${id}`);
    return handleResponse<RateLimitPolicy>(response);
  },

  /**
   * Create a new rate limit policy
   * POST /api/v1/rate-limits/policies
   */
  async create(data: CreateRateLimitPolicyRequest): Promise<RateLimitPolicy> {
    const response = await fetchWithAuth(`${API_BASE_URL}/rate-limits/policies`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    return handleResponse<RateLimitPolicy>(response);
  },

  /**
   * Update a rate limit policy
   * PATCH /api/v1/rate-limits/policies/:id
   */
  async update(id: string, data: UpdateRateLimitPolicyRequest): Promise<RateLimitPolicy> {
    const response = await fetchWithAuth(`${API_BASE_URL}/rate-limits/policies/${id}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    return handleResponse<RateLimitPolicy>(response);
  },

  /**
   * Delete a rate limit policy
   * DELETE /api/v1/rate-limits/policies/:id
   */
  async delete(id: string): Promise<void> {
    const response = await fetchWithAuth(`${API_BASE_URL}/rate-limits/policies/${id}`, {
      method: 'DELETE',
    });
    return handleResponse<void>(response);
  },

  /**
   * Toggle policy status (active/disabled)
   * POST /api/v1/rate-limits/policies/:id/toggle
   */
  async toggle(id: string, enabled: boolean): Promise<RateLimitPolicy> {
    const response = await fetchWithAuth(`${API_BASE_URL}/rate-limits/policies/${id}/toggle`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ enabled }),
    });
    return handleResponse<RateLimitPolicy>(response);
  },

  /**
   * Clone a policy
   * POST /api/v1/rate-limits/policies/:id/clone
   */
  async clone(id: string, name: string): Promise<RateLimitPolicy> {
    const response = await fetchWithAuth(`${API_BASE_URL}/rate-limits/policies/${id}/clone`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name }),
    });
    return handleResponse<RateLimitPolicy>(response);
  },

  /**
   * Test a policy against a request
   * POST /api/v1/rate-limits/policies/:id/test
   */
  async test(
    id: string,
    request: { identifier: string; endpoint: string; method?: string }
  ): Promise<{ allowed: boolean; reason?: string; retryAfter?: number }> {
    const response = await fetchWithAuth(`${API_BASE_URL}/rate-limits/policies/${id}/test`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(request),
    });
    return handleResponse<{ allowed: boolean; reason?: string; retryAfter?: number }>(response);
  },
};

// ============================================================================
// Rate Limit Violations API
// ============================================================================

export const rateLimitViolationsApi = {
  /**
   * List rate limit violations with filtering
   * GET /api/v1/rate-limits/violations
   */
  async list(
    filters?: ViolationFilters,
    limit = 50,
    offset = 0
  ): Promise<RateLimitViolationsResponse> {
    const searchParams = new URLSearchParams();
    if (filters?.policyId) searchParams.set('policy_id', filters.policyId);
    if (filters?.identifier) searchParams.set('identifier', filters.identifier);
    if (filters?.identifierType) searchParams.set('identifier_type', filters.identifierType);
    if (filters?.severity) searchParams.set('severity', filters.severity);
    if (filters?.startDate) searchParams.set('start_date', filters.startDate);
    if (filters?.endDate) searchParams.set('end_date', filters.endDate);
    if (filters?.resolved !== undefined) searchParams.set('resolved', filters.resolved.toString());
    if (filters?.search) searchParams.set('search', filters.search);
    if (filters?.sortBy) searchParams.set('sort_by', filters.sortBy);
    if (filters?.sortOrder) searchParams.set('sort_order', filters.sortOrder);
    searchParams.set('limit', limit.toString());
    searchParams.set('offset', offset.toString());

    const queryString = searchParams.toString();
    const response = await fetchWithAuth(
      `${API_BASE_URL}/rate-limits/violations${queryString ? `?${queryString}` : ''}`
    );
    return handleResponse<RateLimitViolationsResponse>(response);
  },

  /**
   * Get a single violation by ID
   * GET /api/v1/rate-limits/violations/:id
   */
  async get(id: string): Promise<RateLimitViolation> {
    const response = await fetchWithAuth(`${API_BASE_URL}/rate-limits/violations/${id}`);
    return handleResponse<RateLimitViolation>(response);
  },

  /**
   * Get violation statistics
   * GET /api/v1/rate-limits/violations/stats
   */
  async getStats(params?: {
    startDate?: string;
    endDate?: string;
    groupBy?: 'severity' | 'policy' | 'identifier' | 'day';
  }): Promise<ViolationStats> {
    const searchParams = new URLSearchParams();
    if (params?.startDate) searchParams.set('start_date', params.startDate);
    if (params?.endDate) searchParams.set('end_date', params.endDate);
    if (params?.groupBy) searchParams.set('group_by', params.groupBy);

    const queryString = searchParams.toString();
    const response = await fetchWithAuth(
      `${API_BASE_URL}/rate-limits/violations/stats${queryString ? `?${queryString}` : ''}`
    );
    return handleResponse<ViolationStats>(response);
  },

  /**
   * Resolve a violation
   * POST /api/v1/rate-limits/violations/:id/resolve
   */
  async resolve(id: string, notes?: string): Promise<RateLimitViolation> {
    const response = await fetchWithAuth(`${API_BASE_URL}/rate-limits/violations/${id}/resolve`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ notes }),
    });
    return handleResponse<RateLimitViolation>(response);
  },

  /**
   * Bulk resolve violations
   * POST /api/v1/rate-limits/violations/bulk-resolve
   */
  async bulkResolve(
    ids: string[],
    notes?: string
  ): Promise<{ resolved: number; failed: number }> {
    const response = await fetchWithAuth(`${API_BASE_URL}/rate-limits/violations/bulk-resolve`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ids, notes }),
    });
    return handleResponse<{ resolved: number; failed: number }>(response);
  },

  /**
   * Export violations as CSV
   * GET /api/v1/rate-limits/violations/export
   */
  async export(filters?: ViolationFilters): Promise<Blob> {
    const searchParams = new URLSearchParams();
    if (filters?.policyId) searchParams.set('policy_id', filters.policyId);
    if (filters?.identifier) searchParams.set('identifier', filters.identifier);
    if (filters?.severity) searchParams.set('severity', filters.severity);
    if (filters?.startDate) searchParams.set('start_date', filters.startDate);
    if (filters?.endDate) searchParams.set('end_date', filters.endDate);

    const queryString = searchParams.toString();
    const response = await fetchWithAuth(
      `${API_BASE_URL}/rate-limits/violations/export${queryString ? `?${queryString}` : ''}`
    );

    if (!response.ok) {
      throw new Error('Failed to export violations');
    }

    return response.blob();
  },
};

// ============================================================================
// Circuit Breaker API
// ============================================================================

export const circuitBreakersApi = {
  /**
   * List all circuit breakers
   * GET /api/v1/rate-limits/circuit-breakers
   */
  async list(limit = 50, offset = 0): Promise<CircuitBreakersResponse> {
    const searchParams = new URLSearchParams();
    searchParams.set('limit', limit.toString());
    searchParams.set('offset', offset.toString());

    const queryString = searchParams.toString();
    const response = await fetchWithAuth(
      `${API_BASE_URL}/rate-limits/circuit-breakers${queryString ? `?${queryString}` : ''}`
    );
    return handleResponse<CircuitBreakersResponse>(response);
  },

  /**
   * Get a single circuit breaker by ID
   * GET /api/v1/rate-limits/circuit-breakers/:id
   */
  async get(id: string): Promise<CircuitBreaker> {
    const response = await fetchWithAuth(`${API_BASE_URL}/rate-limits/circuit-breakers/${id}`);
    return handleResponse<CircuitBreaker>(response);
  },

  /**
   * Create a new circuit breaker
   * POST /api/v1/rate-limits/circuit-breakers
   */
  async create(data: CreateCircuitBreakerRequest): Promise<CircuitBreaker> {
    const response = await fetchWithAuth(`${API_BASE_URL}/rate-limits/circuit-breakers`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    return handleResponse<CircuitBreaker>(response);
  },

  /**
   * Update a circuit breaker
   * PATCH /api/v1/rate-limits/circuit-breakers/:id
   */
  async update(id: string, data: UpdateCircuitBreakerRequest): Promise<CircuitBreaker> {
    const response = await fetchWithAuth(`${API_BASE_URL}/rate-limits/circuit-breakers/${id}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    return handleResponse<CircuitBreaker>(response);
  },

  /**
   * Delete a circuit breaker
   * DELETE /api/v1/rate-limits/circuit-breakers/:id
   */
  async delete(id: string): Promise<void> {
    const response = await fetchWithAuth(`${API_BASE_URL}/rate-limits/circuit-breakers/${id}`, {
      method: 'DELETE',
    });
    return handleResponse<void>(response);
  },

  /**
   * Manually change circuit breaker state
   * POST /api/v1/rate-limits/circuit-breakers/:id/state
   */
  async setState(id: string, state: CircuitBreakerState, reason?: string): Promise<CircuitBreaker> {
    const response = await fetchWithAuth(`${API_BASE_URL}/rate-limits/circuit-breakers/${id}/state`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ state, reason }),
    });
    return handleResponse<CircuitBreaker>(response);
  },

  /**
   * Reset a circuit breaker (close it)
   * POST /api/v1/rate-limits/circuit-breakers/:id/reset
   */
  async reset(id: string): Promise<CircuitBreaker> {
    const response = await fetchWithAuth(
      `${API_BASE_URL}/rate-limits/circuit-breakers/${id}/reset`,
      {
        method: 'POST',
      }
    );
    return handleResponse<CircuitBreaker>(response);
  },

  /**
   * Get circuit breaker state transitions
   * GET /api/v1/rate-limits/circuit-breakers/:id/transitions
   */
  async getTransitions(id: string, limit = 50): Promise<CircuitBreakerTransition[]> {
    const searchParams = new URLSearchParams();
    searchParams.set('limit', limit.toString());

    const queryString = searchParams.toString();
    const response = await fetchWithAuth(
      `${API_BASE_URL}/rate-limits/circuit-breakers/${id}/transitions${queryString ? `?${queryString}` : ''}`
    );
    return handleResponse<CircuitBreakerTransition[]>(response);
  },

  /**
   * Enable/disable a circuit breaker
   * POST /api/v1/rate-limits/circuit-breakers/:id/toggle
   */
  async toggle(id: string, enabled: boolean): Promise<CircuitBreaker> {
    const response = await fetchWithAuth(`${API_BASE_URL}/rate-limits/circuit-breakers/${id}/toggle`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ enabled }),
    });
    return handleResponse<CircuitBreaker>(response);
  },
};

// ============================================================================
// Trusted Clients API
// ============================================================================

export const trustedClientsApi = {
  /**
   * List all trusted clients
   * GET /api/v1/rate-limits/trusted-clients
   */
  async list(limit = 50, offset = 0): Promise<TrustedClientsResponse> {
    const searchParams = new URLSearchParams();
    searchParams.set('limit', limit.toString());
    searchParams.set('offset', offset.toString());

    const queryString = searchParams.toString();
    const response = await fetchWithAuth(
      `${API_BASE_URL}/rate-limits/trusted-clients${queryString ? `?${queryString}` : ''}`
    );
    return handleResponse<TrustedClientsResponse>(response);
  },

  /**
   * Get a single trusted client by ID
   * GET /api/v1/rate-limits/trusted-clients/:id
   */
  async get(id: string): Promise<TrustedClient> {
    const response = await fetchWithAuth(`${API_BASE_URL}/rate-limits/trusted-clients/${id}`);
    return handleResponse<TrustedClient>(response);
  },

  /**
   * Create a new trusted client
   * POST /api/v1/rate-limits/trusted-clients
   */
  async create(data: CreateTrustedClientRequest): Promise<TrustedClient> {
    const response = await fetchWithAuth(`${API_BASE_URL}/rate-limits/trusted-clients`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    return handleResponse<TrustedClient>(response);
  },

  /**
   * Update a trusted client
   * PATCH /api/v1/rate-limits/trusted-clients/:id
   */
  async update(id: string, data: UpdateTrustedClientRequest): Promise<TrustedClient> {
    const response = await fetchWithAuth(`${API_BASE_URL}/rate-limits/trusted-clients/${id}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    return handleResponse<TrustedClient>(response);
  },

  /**
   * Delete a trusted client
   * DELETE /api/v1/rate-limits/trusted-clients/:id
   */
  async delete(id: string): Promise<void> {
    const response = await fetchWithAuth(`${API_BASE_URL}/rate-limits/trusted-clients/${id}`, {
      method: 'DELETE',
    });
    return handleResponse<void>(response);
  },

  /**
   * Toggle trusted client active status
   * POST /api/v1/rate-limits/trusted-clients/:id/toggle
   */
  async toggle(id: string, active: boolean): Promise<TrustedClient> {
    const response = await fetchWithAuth(`${API_BASE_URL}/rate-limits/trusted-clients/${id}/toggle`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ active }),
    });
    return handleResponse<TrustedClient>(response);
  },
};

// ============================================================================
// Rate Limit Metrics API
// ============================================================================

export const rateLimitMetricsApi = {
  /**
   * Get real-time rate limit metrics
   * GET /api/v1/rate-limits/metrics
   */
  async getMetrics(period: string = '5m'): Promise<RateLimitMetrics> {
    const response = await fetchWithAuth(
      `${API_BASE_URL}/rate-limits/metrics?period=${period}`
    );
    return handleResponse<RateLimitMetrics>(response);
  },

  /**
   * Get rate limit summary statistics
   * GET /api/v1/rate-limits/metrics/summary
   */
  async getSummary(): Promise<RateLimitSummary> {
    const response = await fetchWithAuth(`${API_BASE_URL}/rate-limits/metrics/summary`);
    return handleResponse<RateLimitSummary>(response);
  },

  /**
   * Get historical rate limit data
   * GET /api/v1/rate-limits/metrics/history
   */
  async getHistory(params?: {
    startDate?: string;
    endDate?: string;
    granularity?: 'hour' | 'day' | 'week';
  }): Promise<RateLimitHistoryPoint[]> {
    const searchParams = new URLSearchParams();
    if (params?.startDate) searchParams.set('start_date', params.startDate);
    if (params?.endDate) searchParams.set('end_date', params.endDate);
    if (params?.granularity) searchParams.set('granularity', params.granularity);

    const queryString = searchParams.toString();
    const response = await fetchWithAuth(
      `${API_BASE_URL}/rate-limits/metrics/history${queryString ? `?${queryString}` : ''}`
    );
    return handleResponse<RateLimitHistoryPoint[]>(response);
  },

  /**
   * Get metrics for a specific policy
   * GET /api/v1/rate-limits/policies/:id/metrics
   */
  async getPolicyMetrics(
    policyId: string,
    params?: { startDate?: string; endDate?: string }
  ): Promise<RateLimitHistoryPoint[]> {
    const searchParams = new URLSearchParams();
    if (params?.startDate) searchParams.set('start_date', params.startDate);
    if (params?.endDate) searchParams.set('end_date', params.endDate);

    const queryString = searchParams.toString();
    const response = await fetchWithAuth(
      `${API_BASE_URL}/rate-limits/policies/${policyId}/metrics${queryString ? `?${queryString}` : ''}`
    );
    return handleResponse<RateLimitHistoryPoint[]>(response);
  },
};

// ============================================================================
// Queue Management API
// ============================================================================

export const rateLimitQueueApi = {
  /**
   * List queued requests
   * GET /api/v1/rate-limits/queue
   */
  async list(limit = 50, offset = 0): Promise<QueuedRequestsResponse> {
    const searchParams = new URLSearchParams();
    searchParams.set('limit', limit.toString());
    searchParams.set('offset', offset.toString());

    const queryString = searchParams.toString();
    const response = await fetchWithAuth(
      `${API_BASE_URL}/rate-limits/queue${queryString ? `?${queryString}` : ''}`
    );
    return handleResponse<QueuedRequestsResponse>(response);
  },

  /**
   * Get queue statistics
   * GET /api/v1/rate-limits/queue/stats
   */
  async getStats(): Promise<QueueStats> {
    const response = await fetchWithAuth(`${API_BASE_URL}/rate-limits/queue/stats`);
    return handleResponse<QueueStats>(response);
  },

  /**
   * Process a queued request immediately
   * POST /api/v1/rate-limits/queue/:id/process
   */
  async processRequest(id: string): Promise<{ success: boolean; message?: string }> {
    const response = await fetchWithAuth(`${API_BASE_URL}/rate-limits/queue/${id}/process`, {
      method: 'POST',
    });
    return handleResponse<{ success: boolean; message?: string }>(response);
  },

  /**
   * Cancel a queued request
   * DELETE /api/v1/rate-limits/queue/:id
   */
  async cancelRequest(id: string): Promise<void> {
    const response = await fetchWithAuth(`${API_BASE_URL}/rate-limits/queue/${id}`, {
      method: 'DELETE',
    });
    return handleResponse<void>(response);
  },

  /**
   * Clear all queued requests
   * DELETE /api/v1/rate-limits/queue/clear
   */
  async clearQueue(): Promise<{ cleared: number }> {
    const response = await fetchWithAuth(`${API_BASE_URL}/rate-limits/queue/clear`, {
      method: 'DELETE',
    });
    return handleResponse<{ cleared: number }>(response);
  },

  /**
   * Update queue priority
   * PATCH /api/v1/rate-limits/queue/:id/priority
   */
  async updatePriority(
    id: string,
    priority: 'low' | 'normal' | 'high' | 'urgent'
  ): Promise<QueuedRequest> {
    const response = await fetchWithAuth(`${API_BASE_URL}/rate-limits/queue/${id}/priority`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ priority }),
    });
    return handleResponse<QueuedRequest>(response);
  },
};

// ============================================================================
// Utility Functions
// ============================================================================

/**
 * Format a rate limit window for display
 */
export function formatRateLimitWindow(window: string, windowSize?: number): string {
  const pluralize = (n: number, unit: string) => `${n} ${unit}${n !== 1 ? 's' : ''}`;

  if (windowSize) {
    switch (window) {
      case 'second':
        return pluralize(windowSize, 'second');
      case 'minute':
        return pluralize(windowSize, 'minute');
      case 'hour':
        return pluralize(windowSize, 'hour');
      case 'day':
        return pluralize(windowSize, 'day');
    }
  }

  switch (window) {
    case 'second':
      return 'Per second';
    case 'minute':
      return 'Per minute';
    case 'hour':
      return 'Per hour';
    case 'day':
      return 'Per day';
    default:
      return window;
  }
}

/**
 * Get color class for circuit breaker state
 */
export function getCircuitBreakerStateColor(state: CircuitBreakerState): string {
  switch (state) {
    case 'closed':
      return 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300';
    case 'open':
      return 'bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-300';
    case 'half_open':
      return 'bg-yellow-100 dark:bg-yellow-900/30 text-yellow-700 dark:text-yellow-300';
    default:
      return 'bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300';
  }
}

/**
 * Get color class for violation severity
 */
export function getViolationSeverityColor(severity: string): string {
  switch (severity) {
    case 'low':
      return 'bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300';
    case 'medium':
      return 'bg-yellow-100 dark:bg-yellow-900/30 text-yellow-700 dark:text-yellow-300';
    case 'high':
      return 'bg-orange-100 dark:bg-orange-900/30 text-orange-700 dark:text-orange-300';
    case 'critical':
      return 'bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-300';
    default:
      return 'bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300';
  }
}

/**
 * Calculate percentage for rate limit usage
 */
export function calculateUsagePercentage(used: number, limit: number): number {
  if (limit === 0) return 0;
  return Math.min(100, Math.round((used / limit) * 100));
}

/**
 * Get color class for usage percentage
 */
export function getUsageColor(percentage: number): string {
  if (percentage >= 90) return 'text-red-600 dark:text-red-400';
  if (percentage >= 70) return 'text-yellow-600 dark:text-yellow-400';
  return 'text-green-600 dark:text-green-400';
}

/**
 * Get bar color class for usage percentage
 */
export function getUsageBarColor(percentage: number): string {
  if (percentage >= 90) return 'bg-red-500';
  if (percentage >= 70) return 'bg-yellow-500';
  return 'bg-green-500';
}

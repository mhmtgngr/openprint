/**
 * Rate Limiting Types for OpenPrint Dashboard
 *
 * This file contains all TypeScript types related to the API rate limiting
 * and throttling system including policies, violations, circuit breakers,
 * and trusted clients.
 */

// ============================================================================
// Rate Limit Policy Types
// ============================================================================

/**
 * Rate limit dimension types - what the rate limit applies to
 */
export type RateLimitDimension = 'global' | 'per_ip' | 'per_user' | 'per_api_key' | 'per_endpoint';

/**
 * Time window types for rate limiting
 */
export type RateLimitWindow = 'second' | 'minute' | 'hour' | 'day';

/**
 * Algorithm types for rate limiting
 */
export type RateLimitAlgorithm = 'sliding_window' | 'fixed_window' | 'token_bucket' | 'leaky_bucket';

/**
 * Scope of a rate limit policy
 */
export type RateLimitScope = 'organization' | 'endpoint' | 'user' | 'api_key';

/**
 * Status of a rate limit policy
 */
export type RateLimitPolicyStatus = 'active' | 'disabled' | 'draft';

/**
 * Priority level for requests in the queue
 */
export type QueuePriority = 'low' | 'normal' | 'high' | 'urgent';

/**
 * Circuit breaker states
 */
export type CircuitBreakerState = 'closed' | 'open' | 'half_open';

/**
 * Violation severity levels
 */
export type ViolationSeverity = 'low' | 'medium' | 'high' | 'critical';

/**
 * Action to take when rate limit is exceeded
 */
export type RateLimitAction = 'reject' | 'queue' | 'throttle' | 'allow_with_warning';

/**
 * Main rate limit policy configuration
 */
export interface RateLimitPolicy {
  id: string;
  orgId?: string;
  name: string;
  description?: string;
  scope: RateLimitScope;
  dimension: RateLimitDimension;
  algorithm: RateLimitAlgorithm;
  status: RateLimitPolicyStatus;
  priority: number;

  // Rate limit configuration
  limit: number;
  window: RateLimitWindow;
  windowSize?: number; // Custom window size in seconds

  // Burst configuration
  burstLimit?: number;
  burstWindow?: RateLimitWindow;

  // Endpoint-specific configuration
  endpoint?: string;
  method?: string;
  pathPattern?: string; // Regex pattern for matching paths

  // Actions
  actionOnLimit: RateLimitAction;
  retryAfter?: number; // Seconds to wait before retry
  queueMaxSize?: number;
  queueTimeout?: number; // Seconds

  // Advanced options
  allowBypass?: boolean;
  bypassKeys?: string[]; // API keys or IPs that bypass the limit

  // Metadata
  createdAt: string;
  updatedAt: string;
  createdBy?: string;
  lastModifiedBy?: string;

  // Statistics
  currentUsage?: number;
  violationCount?: number;
  avgRequestsPerWindow?: number;
}

/**
 * Request for creating or updating a rate limit policy
 */
export interface CreateRateLimitPolicyRequest {
  name: string;
  description?: string;
  scope: RateLimitScope;
  dimension: RateLimitDimension;
  algorithm?: RateLimitAlgorithm;
  limit: number;
  window: RateLimitWindow;
  windowSize?: number;
  burstLimit?: number;
  burstWindow?: RateLimitWindow;
  endpoint?: string;
  method?: string;
  pathPattern?: string;
  actionOnLimit: RateLimitAction;
  retryAfter?: number;
  queueMaxSize?: number;
  queueTimeout?: number;
  allowBypass?: boolean;
  bypassKeys?: string[];
  priority?: number;
  targetIds?: string[]; // User IDs, API key IDs, or endpoint IDs
}

/**
 * Request for updating a rate limit policy
 */
export interface UpdateRateLimitPolicyRequest extends Partial<CreateRateLimitPolicyRequest> {
  status?: RateLimitPolicyStatus;
}

/**
 * Filters for listing rate limit policies
 */
export interface RateLimitPolicyFilters {
  scope?: RateLimitScope;
  dimension?: RateLimitDimension;
  status?: RateLimitPolicyStatus;
  endpoint?: string;
  search?: string;
  sortBy?: 'name' | 'createdAt' | 'limit' | 'priority' | 'violationCount';
  sortOrder?: 'asc' | 'desc';
}

// ============================================================================
// Rate Limit Violation Types
// ============================================================================

/**
 * Rate limit violation log entry
 */
export interface RateLimitViolation {
  id: string;
  orgId?: string;
  policyId: string;
  policyName: string;

  // Who violated the limit
  identifier: string; // IP address, user ID, or API key
  identifierType: 'ip' | 'user_id' | 'api_key' | 'endpoint';

  // What was requested
  endpoint: string;
  method: string;
  path: string;

  // Violation details
  requestedAt: string;
  limitExceeded: number;
  actualCount: number;
  severity: ViolationSeverity;

  // Response
  actionTaken: RateLimitAction;
  httpStatus?: number;
  responseMessage?: string;

  // Metadata
  userAgent?: string;
  ipAddress?: string;
  requestId?: string;

  // Resolution
  resolved?: boolean;
  resolvedAt?: string;
  resolvedBy?: string;
  notes?: string;

  createdAt: string;
}

/**
 * Aggregated violation statistics
 */
export interface ViolationStats {
  total: number;
  bySeverity: Record<ViolationSeverity, number>;
  byPolicy: Record<string, number>;
  byIdentifier: Record<string, number>;
  topViolators: Array<{
    identifier: string;
    count: number;
    lastViolation: string;
  }>;
  recentTrend: Array<{
    date: string;
    count: number;
  }>;
}

/**
 * Filters for listing violations
 */
export interface ViolationFilters {
  policyId?: string;
  identifier?: string;
  identifierType?: string;
  severity?: ViolationSeverity;
  startDate?: string;
  endDate?: string;
  resolved?: boolean;
  search?: string;
  sortBy?: 'createdAt' | 'severity' | 'limitExceeded' | 'actualCount';
  sortOrder?: 'asc' | 'desc';
}

// ============================================================================
// Circuit Breaker Types
// ============================================================================

/**
 * Circuit breaker configuration
 */
export interface CircuitBreaker {
  id: string;
  orgId?: string;
  name: string;
  description?: string;
  state: CircuitBreakerState;

  // Target configuration
  endpoint: string;
  method?: string;

  // Thresholds
  failureThreshold: number; // Number of failures before opening
  successThreshold: number; // Number of successes to close
  timeoutThreshold: number; // Request timeout in ms

  // Window configuration
  slidingWindowSize: number; // In seconds
  minimumRequests: number; // Minimum requests before tripping

  // Recovery configuration
  halfOpenMaxCalls: number;
  openDuration: number; // How long to stay open in seconds

  // Statistics
  currentFailureCount: number;
  currentSuccessCount: number;
  lastStateChange: string;
  lastFailureAt?: string;
  totalRequests: number;
  totalFailures: number;
  totalSuccesses: number;

  // Metadata
  createdAt: string;
  updatedAt: string;
  isEnabled: boolean;
}

/**
 * Request for creating a circuit breaker
 */
export interface CreateCircuitBreakerRequest {
  name: string;
  description?: string;
  endpoint: string;
  method?: string;
  failureThreshold: number;
  successThreshold: number;
  timeoutThreshold: number;
  slidingWindowSize: number;
  minimumRequests: number;
  halfOpenMaxCalls: number;
  openDuration: number;
  isEnabled?: boolean;
}

/**
 * Request for updating a circuit breaker
 */
export interface UpdateCircuitBreakerRequest extends Partial<CreateCircuitBreakerRequest> {
  state?: CircuitBreakerState;
}

/**
 * Circuit breaker state transition
 */
export interface CircuitBreakerTransition {
  id: string;
  circuitBreakerId: string;
  fromState: CircuitBreakerState;
  toState: CircuitBreakerState;
  reason: string;
  triggeredBy: string; // 'automatic' or user ID
  metadata?: Record<string, unknown>;
  createdAt: string;
}

// ============================================================================
// Trusted Client Types
// ============================================================================

/**
 * Trusted client that bypasses rate limits
 */
export interface TrustedClient {
  id: string;
  orgId?: string;
  name: string;
  type: 'ip' | 'api_key' | 'user_agent' | 'service';
  identifier: string; // IP address, API key prefix, or user agent pattern
  description?: string;

  // Bypass configuration
  bypassAll: boolean;
  bypassPolicies?: string[]; // Policy IDs to bypass

  // Rate limits for trusted client (if not bypassing all)
  customLimit?: number;
  customWindow?: RateLimitWindow;

  // Metadata
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
  createdBy?: string;
  lastUsedAt?: string;

  // Statistics
  requestCount?: number;
  lastRequestAt?: string;
}

/**
 * Request for creating a trusted client
 */
export interface CreateTrustedClientRequest {
  name: string;
  type: 'ip' | 'api_key' | 'user_agent' | 'service';
  identifier: string;
  description?: string;
  bypassAll: boolean;
  bypassPolicies?: string[];
  customLimit?: number;
  customWindow?: RateLimitWindow;
  isActive?: boolean;
}

/**
 * Request for updating a trusted client
 */
export interface UpdateTrustedClientRequest extends Partial<CreateTrustedClientRequest> {}

// ============================================================================
// Rate Limit Metrics Types
// ============================================================================

/**
 * Real-time rate limit metrics
 */
export interface RateLimitMetrics {
  timestamp: string;
  period: string; // '1m', '5m', '15m', '1h'

  // Request metrics
  totalRequests: number;
  allowedRequests: number;
  throttledRequests: number;
  rejectedRequests: number;
  queuedRequests: number;

  // Policy metrics
  policyHits: Record<string, number>;
  topPolicies: Array<{
    policyId: string;
    policyName: string;
    hits: number;
    violations: number;
  }>;

  // Identifier metrics
  topIdentifiers: Array<{
    identifier: string;
    identifierType: string;
    requests: number;
    violations: number;
  }>;

  // Endpoint metrics
  endpointMetrics: Array<{
    endpoint: string;
    method: string;
    requests: number;
    avgResponseTime: number;
    violationRate: number;
  }>;

  // Queue metrics
  averageQueueLength: number;
  maxQueueLength: number;
  averageQueueWaitTime: number; // In milliseconds
}

/**
 * Rate limit summary statistics
 */
export interface RateLimitSummary {
  totalPolicies: number;
  activePolicies: number;
  totalViolations: number;
  violationsToday: number;
  violationsThisWeek: number;
  activeCircuitBreakers: number;
  totalTrustedClients: number;
  avgResponseTime: number;
  p95ResponseTime: number;
  p99ResponseTime: number;
}

/**
 * Historical rate limit data point
 */
export interface RateLimitHistoryPoint {
  timestamp: string;
  requests: number;
  allowed: number;
  throttled: number;
  rejected: number;
  violations: number;
}

// ============================================================================
// Queue Management Types
// ============================================================================

/**
 * Queued request information
 */
export interface QueuedRequest {
  id: string;
  policyId: string;
  identifier: string;
  identifierType: string;
  endpoint: string;
  method: string;
  path: string;
  priority: QueuePriority;
  queuedAt: string;
  estimatedWait?: number; // In seconds
  retryAfter?: number;
  metadata?: Record<string, unknown>;
}

/**
 * Queue statistics
 */
export interface QueueStats {
  totalQueued: number;
  byPriority: Record<QueuePriority, number>;
  averageWaitTime: number;
  maxWaitTime: number;
  processedCount: number;
  expiredCount: number;
}

// ============================================================================
// Print Quota Types (Print-specific extensions)
// ============================================================================

/**
 * Print quota policy extending base rate limit
 */
export interface PrintQuotaPolicy extends RateLimitPolicy {
  quotaType: 'pages' | 'jobs' | 'color_pages' | 'cost';
  period: 'daily' | 'weekly' | 'monthly';
  resetDate?: string;

  // Print-specific configuration
  allowOverage?: boolean;
  overageAction?: 'block' | 'charge' | 'warn' | 'allow';
  overageCostPerPage?: number;

  // User/group assignment
  appliesToAll?: boolean;
  userIds?: string[];
  groupIds?: string[];
  printerIds?: string[];
}

/**
 * User print quota status
 */
export interface UserPrintQuota {
  userId: string;
  userName: string;
  policyId: string;
  policyName: string;

  // Quota limits
  quotaLimit: number;
  quotaUsed: number;
  quotaRemaining: number;
  quotaPercentage: number;

  // Period info
  periodStart: string;
  periodEnd: string;
  resetDate: string;

  // Overage
  overageAmount?: number;
  overageCost?: number;

  // Status
  isBlocked?: boolean;
  warningsSent: number;
}

/**
 * Cost allocation for print jobs
 */
export interface PrintCostAllocation {
  id: string;
  jobId: string;
  userId: string;
  quotaId: string;

  // Cost breakdown
  totalPages: number;
  colorPages: number;
  costPerPage: number;
  costPerColorPage: number;
  totalCost: number;

  // Allocation
  allocatedFrom: 'user_quota' | 'group_quota' | 'organization_pool';
  quotaRemainingAfter: number;

  createdAt: string;
}

// ============================================================================
// API Response Wrappers
// ============================================================================

export interface RateLimitPoliciesResponse {
  data: RateLimitPolicy[];
  total: number;
  limit: number;
  offset: number;
}

export interface RateLimitViolationsResponse {
  data: RateLimitViolation[];
  total: number;
  limit: number;
  offset: number;
  stats: ViolationStats;
}

export interface CircuitBreakersResponse {
  data: CircuitBreaker[];
  total: number;
  limit: number;
  offset: number;
}

export interface TrustedClientsResponse {
  data: TrustedClient[];
  total: number;
  limit: number;
  offset: number;
}

export interface QueuedRequestsResponse {
  data: QueuedRequest[];
  total: number;
  stats: QueueStats;
}

// ============================================================================
// UI State Types
// ============================================================================

export interface RateLimitUIState {
  selectedPolicy?: RateLimitPolicy;
  isCreating: boolean;
  isEditing: boolean;
  showDeleteModal: boolean;
  showViolationsPanel: boolean;
  selectedViolation?: RateLimitViolation;
}

export interface CircuitBreakerUIState {
  selectedBreaker?: CircuitBreaker;
  isCreating: boolean;
  isEditing: boolean;
  showTransitions: boolean;
  selectedTransition?: CircuitBreakerTransition;
}

// ============================================================================
// Form Types
// ============================================================================

export interface RateLimitPolicyFormData {
  name: string;
  description?: string;
  scope: RateLimitScope;
  dimension: RateLimitDimension;
  algorithm: RateLimitAlgorithm;
  limit: string;
  window: RateLimitWindow;
  windowSize?: string;
  burstLimit?: string;
  burstWindow?: RateLimitWindow;
  endpoint?: string;
  method?: string;
  pathPattern?: string;
  actionOnLimit: RateLimitAction;
  retryAfter?: string;
  queueMaxSize?: string;
  queueTimeout?: string;
  allowBypass?: boolean;
  bypassKeys?: string[];
  priority?: string;
}

export interface CircuitBreakerFormData {
  name: string;
  description?: string;
  endpoint: string;
  method?: string;
  failureThreshold: string;
  successThreshold: string;
  timeoutThreshold: string;
  slidingWindowSize: string;
  minimumRequests: string;
  halfOpenMaxCalls: string;
  openDuration: string;
  isEnabled: boolean;
}

export interface TrustedClientFormData {
  name: string;
  type: 'ip' | 'api_key' | 'user_agent' | 'service';
  identifier: string;
  description?: string;
  bypassAll: boolean;
  bypassPolicies?: string[];
  customLimit?: string;
  customWindow?: RateLimitWindow;
  isActive: boolean;
}

/**
 * Monitoring API Service
 * Provides integration with Prometheus, AlertManager, and Jaeger
 * for metrics, alerts, and distributed tracing
 */

import type {
  Alert,
  AlertSummary,
  AlertGroup,
  AlertRule,
  Silence,
  Matcher,
  ServiceHealth,
  ServiceMetrics,
  TimeRange,
  Trace,
  TraceQuery,
  TraceSearchResult,
  TraceSummary,
  QueryResult,
  MetricData,
  GrafanaDashboard,
  GrafanaFolder,
} from '@/types';

const MONITORING_BASE_URL = import.meta.env.VITE_MONITORING_API_URL || '/monitoring';
const PROMETHEUS_URL = import.meta.env.VITE_PROMETHEUS_URL || 'http://localhost:9090';
const GRAFANA_URL = import.meta.env.VITE_GRAFANA_URL || 'http://localhost:3000';
const JAEGER_URL = import.meta.env.VITE_JAEGER_URL || 'http://localhost:16686';

// Helper to calculate timestamp from time range
function getStartTime(range: TimeRange): number {
  const now = Date.now();
  const multipliers: Record<TimeRange, number> = {
    '5m': 5 * 60 * 1000,
    '15m': 15 * 60 * 1000,
    '30m': 30 * 60 * 1000,
    '1h': 60 * 60 * 1000,
    '3h': 3 * 60 * 60 * 1000,
    '6h': 6 * 60 * 60 * 1000,
    '12h': 12 * 60 * 60 * 1000,
    '24h': 24 * 60 * 60 * 1000,
    '7d': 7 * 24 * 60 * 60 * 1000,
    '30d': 30 * 24 * 60 * 60 * 1000,
  };
  return now - multipliers[range];
}

// ============== Service Health ==============
export const serviceHealthApi = {
  /**
   * Get health status for all services
   */
  async getAllServices(): Promise<ServiceHealth[]> {
    const response = await fetch(`${MONITORING_BASE_URL}/health/services`);
    if (!response.ok) {
      throw new Error(`Failed to fetch service health: ${response.statusText}`);
    }
    return response.json();
  },

  /**
   * Get health for a specific service
   */
  async getService(serviceName: string): Promise<ServiceHealth> {
    const response = await fetch(`${MONITORING_BASE_URL}/health/services/${serviceName}`);
    if (!response.ok) {
      throw new Error(`Failed to fetch service health: ${response.statusText}`);
    }
    return response.json();
  },

  /**
   * Get service metrics summary
   */
  async getServiceMetrics(serviceName: string): Promise<ServiceMetrics> {
    const response = await fetch(`${MONITORING_BASE_URL}/metrics/services/${serviceName}`);
    if (!response.ok) {
      throw new Error(`Failed to fetch service metrics: ${response.statusText}`);
    }
    return response.json();
  },
};

// ============== Prometheus Metrics ==============
export const prometheusApi = {
  /**
   * Execute a PromQL query
   */
  async query(query: string, time?: number): Promise<QueryResult> {
    const params = new URLSearchParams({ query });
    if (time) {
      params.set('time', time.toString());
    }
    const response = await fetch(`${PROMETHEUS_URL}/api/v1/query?${params.toString()}`);
    if (!response.ok) {
      throw new Error(`Prometheus query failed: ${response.statusText}`);
    }
    return response.json();
  },

  /**
   * Execute a PromQL range query
   */
  async queryRange(query: string, range: TimeRange, step: string = '15s'): Promise<QueryResult> {
    const params = new URLSearchParams({
      query,
      start: (getStartTime(range) / 1000).toString(),
      end: (Date.now() / 1000).toString(),
      step,
    });
    const response = await fetch(`${PROMETHEUS_URL}/api/v1/query_range?${params.toString()}`);
    if (!response.ok) {
      throw new Error(`Prometheus range query failed: ${response.statusText}`);
    }
    return response.json();
  },

  /**
   * Get current metric values for all services
   */
  async getServiceMetrics(serviceName: string): Promise<MetricData[]> {
    const queries = [
      `rate(http_requests_total{service="${serviceName}"}[5m])`,
      `rate(http_request_duration_ms_sum{service="${serviceName}"}[5m]) / rate(http_request_duration_ms_count{service="${serviceName}"}[5m])`,
      `http_requests_in_progress{service="${serviceName}"}`,
      `pg_stat_activity_count{datname="openprint",service="${serviceName}"}`,
      `redis_commands_total{service="${serviceName}"}`,
    ];

    const results = await Promise.allSettled(
      queries.map((q) => this.query(q))
    );

    return results
      .filter((r): r is PromiseFulfilledResult<QueryResult> => r.status === 'fulfilled')
      .map((r) => r.value.data.result)
      .flat();
  },

  /**
   * Get HTTP request rate over time
   */
  async getRequestRate(serviceName?: string, range: TimeRange = '1h'): Promise<QueryResult> {
    const query = serviceName
      ? `sum(rate(http_requests_total{service="${serviceName}"}[5m])) by (instance)`
      : 'sum(rate(http_requests_total[5m])) by (service, instance)';
    return this.queryRange(query, range);
  },

  /**
   * Get error rate over time
   */
  async getErrorRate(serviceName?: string, range: TimeRange = '1h'): Promise<QueryResult> {
    const query = serviceName
      ? `sum(rate(http_requests_total{status=~"5..",service="${serviceName}"}[5m])) by (instance)`
      : 'sum(rate(http_requests_total{status=~"5.."}[5m])) by (service, instance)';
    return this.queryRange(query, range);
  },

  /**
   * Get latency percentiles
   */
  async getLatency(serviceName?: string, range: TimeRange = '1h'): Promise<QueryResult> {
    const query = serviceName
      ? `histogram_quantile(0.95, sum(rate(http_request_duration_ms_bucket{service="${serviceName}"}[5m])) by (le, instance))`
      : 'histogram_quantile(0.95, sum(rate(http_request_duration_ms_bucket[5m])) by (le, service, instance))';
    return this.queryRange(query, range);
  },

  /**
   * List all available metrics
   */
  async listMetrics(): Promise<string[]> {
    const response = await fetch(`${PROMETHEUS_URL}/api/v1/label/__name__/values`);
    if (!response.ok) {
      throw new Error(`Failed to list metrics: ${response.statusText}`);
    }
    const data = await response.json();
    return data.data;
  },

  /**
   * Get metric metadata
   */
  async getMetricMetadata(metric: string): Promise<Record<string, unknown>> {
    const response = await fetch(`${PROMETHEUS_URL}/api/v1/metadata?metric=${metric}`);
    if (!response.ok) {
      throw new Error(`Failed to get metric metadata: ${response.statusText}`);
    }
    const data = await response.json();
    return data.data;
  },
};

// ============== Alerts ==============
export const alertsApi = {
  /**
   * Get all current alerts
   */
  async getAlerts(): Promise<Alert[]> {
    const response = await fetch(`${MONITORING_BASE_URL}/alerts`);
    if (!response.ok) {
      throw new Error(`Failed to fetch alerts: ${response.statusText}`);
    }
    return response.json();
  },

  /**
   * Get alert summary
   */
  async getAlertSummary(): Promise<AlertSummary> {
    const response = await fetch(`${MONITORING_BASE_URL}/alerts/summary`);
    if (!response.ok) {
      throw new Error(`Failed to fetch alert summary: ${response.statusText}`);
    }
    return response.json();
  },

  /**
   * Get alerts for a specific service
   */
  async getAlertsByService(serviceName: string): Promise<Alert[]> {
    const response = await fetch(`${MONITORING_BASE_URL}/alerts/service/${serviceName}`);
    if (!response.ok) {
      throw new Error(`Failed to fetch service alerts: ${response.statusText}`);
    }
    return response.json();
  },

  /**
   * Get alert groups
   */
  async getAlertGroups(): Promise<AlertGroup[]> {
    const response = await fetch(`${MONITORING_BASE_URL}/alerts/groups`);
    if (!response.ok) {
      throw new Error(`Failed to fetch alert groups: ${response.statusText}`);
    }
    return response.json();
  },

  /**
   * Get alert rules
   */
  async getAlertRules(): Promise<AlertRule[]> {
    const response = await fetch(`${MONITORING_BASE_URL}/alerts/rules`);
    if (!response.ok) {
      throw new Error(`Failed to fetch alert rules: ${response.statusText}`);
    }
    return response.json();
  },

  /**
   * Create a new silence
   */
  async createSilence(matchers: Matcher[], duration: string, comment: string, createdBy: string): Promise<Silence> {
    const response = await fetch(`${MONITORING_BASE_URL}/silences`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        matchers,
        startsAt: new Date().toISOString(),
        endsAt: new Date(Date.now() + parseDuration(duration)).toISOString(),
        createdBy,
        comment,
      }),
    });
    if (!response.ok) {
      throw new Error(`Failed to create silence: ${response.statusText}`);
    }
    return response.json();
  },

  /**
   * Get active silences
   */
  async getSilences(): Promise<Silence[]> {
    const response = await fetch(`${MONITORING_BASE_URL}/silences`);
    if (!response.ok) {
      throw new Error(`Failed to fetch silences: ${response.statusText}`);
    }
    return response.json();
  },

  /**
   * Delete a silence
   */
  async deleteSilence(silenceId: string): Promise<void> {
    const response = await fetch(`${MONITORING_BASE_URL}/silences/${silenceId}`, {
      method: 'DELETE',
    });
    if (!response.ok) {
      throw new Error(`Failed to delete silence: ${response.statusText}`);
    }
  },
};

// Helper to parse duration string to milliseconds
function parseDuration(duration: string): number {
  const match = duration.match(/^(\d+)([smhd])$/);
  if (!match) return 3600000; // Default 1 hour

  const value = parseInt(match[1], 10);
  const unit = match[2];

  const multipliers: Record<string, number> = {
    s: 1000,
    m: 60000,
    h: 3600000,
    d: 86400000,
  };

  return value * multipliers[unit];
}

// ============== Distributed Tracing ==============
export const tracingApi = {
  /**
   * Search for traces
   */
  async searchTraces(query: TraceQuery): Promise<TraceSearchResult[]> {
    const params = new URLSearchParams();
    if (query.service) params.set('service', query.service);
    if (query.operation) params.set('operation', query.operation);
    if (query.tags) {
      Object.entries(query.tags).forEach(([key, value]) => {
        params.set(`tag:${key}`, value);
      });
    }
    if (query.startTimeMin) params.set('start', query.startTimeMin);
    if (query.startTimeMax) params.set('end', query.startTimeMax);
    if (query.durationMin) params.set('minDuration', query.durationMin);
    if (query.durationMax) params.set('maxDuration', query.durationMax);
    if (query.limit) params.set('limit', query.limit.toString());

    const response = await fetch(`${MONITORING_BASE_URL}/traces/search?${params.toString()}`);
    if (!response.ok) {
      throw new Error(`Failed to search traces: ${response.statusText}`);
    }
    return response.json();
  },

  /**
   * Get a specific trace by ID
   */
  async getTrace(traceId: string): Promise<Trace> {
    const response = await fetch(`${MONITORING_BASE_URL}/traces/${traceId}`);
    if (!response.ok) {
      throw new Error(`Failed to get trace: ${response.statusText}`);
    }
    return response.json();
  },

  /**
   * Get trace summary statistics
   */
  async getTraceSummary(service?: string, timeRange: TimeRange = '1h'): Promise<TraceSummary> {
    const params = new URLSearchParams({
      start: (getStartTime(timeRange) / 1000).toString(),
      end: (Date.now() / 1000).toString(),
    });
    if (service) params.set('service', service);

    const response = await fetch(`${MONITORING_BASE_URL}/traces/summary?${params.toString()}`);
    if (!response.ok) {
      throw new Error(`Failed to get trace summary: ${response.statusText}`);
    }
    return response.json();
  },

  /**
   * Get all services with traces
   */
  async getServices(): Promise<string[]> {
    const response = await fetch(`${MONITORING_BASE_URL}/traces/services`);
    if (!response.ok) {
      throw new Error(`Failed to get services: ${response.statusText}`);
    }
    return response.json();
  },

  /**
   * Get operations for a service
   */
  async getOperations(service: string): Promise<string[]> {
    const response = await fetch(`${MONITORING_BASE_URL}/traces/services/${service}/operations`);
    if (!response.ok) {
      throw new Error(`Failed to get operations: ${response.statusText}`);
    }
    return response.json();
  },
};

// ============== Grafana ==============
export const grafanaApi = {
  /**
   * Get Grafana dashboard URL
   */
  getDashboardUrl(dashboardUid: string, params?: Record<string, string>): string {
    const searchParams = new URLSearchParams(params);
    return `${GRAFANA_URL}/d/${dashboardUid}?${searchParams.toString()}`;
  },

  /**
   * List available dashboards
   */
  async getDashboards(folderId?: number): Promise<GrafanaDashboard[]> {
    const params = folderId ? `?folderIds=${folderId}` : '';
    const response = await fetch(`${GRAFANA_URL}/api/search${params}`, {
      headers: {
        'Content-Type': 'application/json',
      },
    });
    if (!response.ok) {
      throw new Error(`Failed to get dashboards: ${response.statusText}`);
    }
    return response.json();
  },

  /**
   * Get dashboard by UID
   */
  async getDashboard(uid: string): Promise<GrafanaDashboard> {
    const response = await fetch(`${GRAFANA_URL}/api/dashboards/uid/${uid}`);
    if (!response.ok) {
      throw new Error(`Failed to get dashboard: ${response.statusText}`);
    }
    return response.json();
  },

  /**
   * Get Grafana folders
   */
  async getFolders(): Promise<GrafanaFolder[]> {
    const response = await fetch(`${GRAFANA_URL}/api/folders`);
    if (!response.ok) {
      throw new Error(`Failed to get folders: ${response.statusText}`);
    }
    return response.json();
  },
};

// ============== Jaeger ==============
export const jaegerApi = {
  /**
   * Get Jaeger UI URL for a trace
   */
  getTraceUrl(traceId: string): string {
    return `${JAEGER_URL}/trace/${traceId}`;
  },

  /**
   * Get Jaeger search URL
   */
  getSearchUrl(params?: {
    service?: string;
    start?: string;
    end?: string;
    minDuration?: string;
    maxDuration?: string;
  }): string {
    const searchParams = new URLSearchParams();
    if (params?.service) searchParams.set('service', params.service);
    if (params?.start) searchParams.set('start', params.start);
    if (params?.end) searchParams.set('end', params.end);
    if (params?.minDuration) searchParams.set('minDuration', params.minDuration);
    if (params?.maxDuration) searchParams.set('maxDuration', params.maxDuration);

    return `${JAEGER_URL}/search?${searchParams.toString()}`;
  },
};

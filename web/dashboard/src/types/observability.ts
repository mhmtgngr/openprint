// Observability types for metrics, alerts, and distributed tracing

// ============== Metrics ==============
export interface PrometheusMetric {
  name: string;
  help: string;
  type: 'counter' | 'gauge' | 'histogram' | 'summary';
  value: number;
  labels?: Record<string, string>;
}

export interface MetricQuery {
  query: string;
  label?: string;
  color?: string;
}

export interface MetricSeries {
  name: string;
  labels: Record<string, string>;
  values: [number, string][];
}

export interface MetricDataPoint {
  timestamp: number;
  value: number;
}

export interface ServiceMetrics {
  serviceName: string;
  instance: string;
  metrics: {
    httpRequestsTotal: number;
    httpRequestDurationMs: number;
    httpRequestsInProgress: number;
    dbConnectionsActive: number;
    dbConnectionsIdle: number;
    redisCommandsTotal: number;
    redisDurationSeconds: number;
  };
  uptime: number;
  version: string;
}

export interface MetricsDashboardConfig {
  refreshInterval: number;
  timeRange: TimeRange;
  queries: MetricQuery[];
}

export type TimeRange = '5m' | '15m' | '30m' | '1h' | '3h' | '6h' | '12h' | '24h' | '7d' | '30d';

// ============== Alerts ==============
export interface Alert {
  id: string;
  name: string;
  state: AlertState;
  severity: AlertSeverity;
  service: string;
  message: string;
  startsAt: string;
  endsAt?: string;
  labels: Record<string, string>;
  annotations: Record<string, string>;
  generatorURL: string;
  silenced?: boolean;
}

export type AlertState = 'firing' | 'pending' | 'resolved' | 'inactive';

export type AlertSeverity = 'critical' | 'warning' | 'info' | 'none';

export interface AlertRule {
  id: string;
  name: string;
  query: string;
  duration: string;
  labels: Record<string, string>;
  annotations: Record<string, string>;
  isEnabled: boolean;
}

export interface AlertGroup {
  name: string;
  rules: AlertRule[];
  interval: string;
}

export interface Silence {
  id: string;
  matchers: Matcher[];
  startsAt: string;
  endsAt: string;
  createdBy: string;
  comment: string;
}

export interface Matcher {
  name: string;
  value: string;
  isRegex: boolean;
}

export interface AlertSummary {
  total: number;
  firing: number;
  pending: number;
  resolved: number;
  bySeverity: Record<AlertSeverity, number>;
  byService: Record<string, number>;
}

// ============== Service Health ==============
export interface ServiceHealth {
  serviceName: string;
  status: HealthStatus;
  instance: string;
  version: string;
  uptime: number;
  lastCheck: string;
  metrics: HealthMetrics;
  dependencies: DependencyHealth[];
}

export type HealthStatus = 'healthy' | 'degraded' | 'unhealthy' | 'unknown';

export interface HealthMetrics {
  cpuPercent: number;
  memoryPercent: number;
  diskPercent: number;
  requestRate: number;
  errorRate: number;
  latency: {
    p50: number;
    p95: number;
    p99: number;
  };
}

export interface DependencyHealth {
  name: string;
  type: 'database' | 'redis' | 'http' | 'grpc' | 'message_queue';
  status: HealthStatus;
  latency?: number;
}

// ============== Distributed Tracing ==============
export interface Trace {
  traceID: string;
  rootSpanName: string;
  rootServiceName: string;
  duration: number;
  startTime: number;
  spans: Span[];
  processes: Record<string, Process>;
}

export interface Span {
  traceID: string;
  spanID: string;
  operationName: string;
  processID: string;
  parentSpanID?: string;
  startTime: number;
  duration: number;
  tags: Tag[];
  logs: Log[];
  warnings?: string[];
}

export interface Process {
  serviceName: string;
  tags: Tag[];
}

export interface Tag {
  key: string;
  value: string;
  type?: 'string' | 'bool' | 'number' | 'binary';
}

export interface Log {
  timestamp: number;
  fields: Tag[];
}

export interface TraceQuery {
  service?: string;
  operation?: string;
  tags?: Record<string, string>;
  startTimeMin?: string;
  startTimeMax?: string;
  durationMin?: string;
  durationMax?: string;
  limit?: number;
}

export interface TraceSearchResult {
  traceID: string;
  rootSpanName: string;
  rootServiceName: string;
  startTime: number;
  duration: number;
  spanCount: number;
  match?: string;
}

export interface TraceSummary {
  totalTraces: number;
  errorTraces: number;
  slowTraces: number;
  avgDuration: number;
  p95Duration: number;
  p99Duration: number;
  byService: Record<string, TraceStats>;
}

export interface TraceStats {
  count: number;
  errorCount: number;
  avgDuration: number;
  maxDuration: number;
}

// ============== Grafana ==============
export interface GrafanaDashboard {
  id: number;
  uid: string;
  title: string;
  tags: string[];
  url: string;
  folderId: number;
  folderTitle: string;
  type: 'dash-db' | 'dash-folder';
}

export interface GrafanaFolder {
  id: number;
  uid: string;
  title: string;
  url: string;
}

// ============== Query Results ==============
export interface QueryResult {
  status: 'success' | 'error';
  data: ResultData;
  errorType?: string;
  error?: string;
}

export interface ResultData {
  resultType: 'matrix' | 'vector' | 'scalar' | 'string';
  result: MetricData[];
}

export interface MetricData {
  metric: Record<string, string>;
  values?: [number, string][];
  value?: [number, string];
}

/**
 * Custom k6 metrics definitions
 *
 * This module defines all custom metrics used across load tests.
 * Metrics are exported to various outputs (InfluxDB, Prometheus, Cloud, etc.)
 */

import { Trend, Rate, Counter, Gauge } from 'k6/metrics';

// Authentication metrics
export const authLoginDuration = new Trend('auth_login_duration', true);
export const authRegisterDuration = new Trend('auth_register_duration', true);
export const authRefreshDuration = new Trend('auth_refresh_duration', true);
export const authLogoutDuration = new Trend('auth_logout_duration', true);
export const authFailures = new Rate('auth_failures');

// Registry metrics
export const registryHeartbeatDuration = new Trend('registry_heartbeat_duration', true);
export const registryDiscoveryDuration = new Trend('registry_discovery_duration', true);
export const registryAgentRegistrationDuration = new Trend('registry_agent_registration_duration', true);
export const registryPrinterListDuration = new Trend('registry_printer_list_duration', true);
export const registryHeartbeatRate = new Rate('registry_heartbeat_success');

// Job service metrics
export const jobSubmitDuration = new Trend('job_submit_duration', true);
export const jobStatusQueryDuration = new Trend('job_status_query_duration', true);
export const jobAssignmentDuration = new Trend('job_assignment_duration', true);
export const jobQueueProcessingDuration = new Trend('job_queue_processing_duration', true);
export const jobSubmissionRate = new Rate('job_submission_success');
export const activeJobs = new Gauge('active_jobs');
export const completedJobs = new Counter('completed_jobs');
export const failedJobs = new Counter('failed_jobs');

// Storage metrics
export const storageUploadDuration = new Trend('storage_upload_duration', true);
export const storageDownloadDuration = new Trend('storage_download_duration', true);
export const storageDeleteDuration = new Trend('storage_delete_duration', true);
export const storageBytesUploaded = new Trend('storage_bytes_uploaded', true);
export const storageBytesDownloaded = new Trend('storage_bytes_downloaded', true);
export const storageUploadRate = new Rate('storage_upload_success');

// Notification metrics
export const notificationConnectDuration = new Trend('notification_connect_duration', true);
export const notificationMessageLatency = new Trend('notification_message_latency', true);
export const notificationMessagesReceived = new Counter('notification_messages_received');
export const notificationMessagesSent = new Counter('notification_messages_sent');
export const notificationConnections = new Gauge('notification_connections');
export const notificationReconnects = new Counter('notification_reconnects');

// WebSocket metrics
export const wsConnectionErrors = new Rate('ws_connection_errors');
export const wsMessagesReceived = new Counter('ws_messages_received');
export const wsMessagesSent = new Counter('ws_messages_sent');
export const wsPingPongLatency = new Trend('ws_ping_pong_latency', true);

// Business metrics
export const userSessions = new Gauge('user_sessions');
export const printJobsCreated = new Counter('print_jobs_created');
export const pagesPrinted = new Counter('pages_printed');
export const documentsStored = new Counter('documents_stored');

// Performance baselines
export const baselineResponseTime = new Trend('baseline_response_time', true);
export const baselineThroughput = new Trend('baseline_throughput', true);
export const baselineErrorRate = new Rate('baseline_error_rate');

// Custom timing metrics (generic)
export const customTimings = new Trend('custom_timings', true);

// System metrics under test
export const systemCpuUsage = new Gauge('system_cpu_usage');
export const systemMemoryUsage = new Gauge('system_memory_usage');
export const systemDiskIO = new Trend('system_disk_io', true);

// Test execution metrics
export const testIterations = new Counter('test_iterations');
export const testErrors = new Counter('test_errors');
export const testWarnings = new Counter('test_warnings');

/**
 * Metrics collector class
 * Provides a convenient way to record metrics
 */
export class MetricsCollector {
  constructor(options = {}) {
    this.prefix = options.prefix || '';
    this.tags = options.tags || {};
  }

  /**
   * Record a timing metric
   */
  recordTiming(name, duration, tags = {}) {
    const metricName = this.prefix + name;
    if (typeof metrics !== 'undefined' && metrics[metricName]) {
      metrics[metricName].add(duration, { ...this.tags, ...tags });
    }
  }

  /**
   * Record a counter increment
   */
  increment(name, value = 1, tags = {}) {
    const metricName = this.prefix + name;
    if (typeof metrics !== 'undefined' && metrics[metricName]) {
      metrics[metricName].add(value, { ...this.tags, ...tags });
    }
  }

  /**
   * Record a gauge value
   */
  setGauge(name, value, tags = {}) {
    const metricName = this.prefix + name;
    if (typeof metrics !== 'undefined' && metrics[metricName]) {
      metrics[metricName].set(value, { ...this.tags, ...tags });
    }
  }

  /**
   * Record a rate (success/failure)
   */
  recordRate(name, success, tags = {}) {
    const metricName = this.prefix + name;
    if (typeof metrics !== 'undefined' && metrics[metricName]) {
      metrics[metricName].add(success ? 1 : 0, { ...this.tags, ...tags });
    }
  }

  /**
   * Measure execution time of a function
   */
  measure(name, fn) {
    const start = Date.now();
    try {
      const result = fn();
      const duration = Date.now() - start;
      this.recordTiming(name, duration);
      return result;
    } catch (error) {
      const duration = Date.now() - start;
      this.recordTiming(name, duration, { error: true });
      throw error;
    }
  }
}

/**
 * Service-specific metric collectors
 */
export class AuthMetrics extends MetricsCollector {
  constructor() {
    super({ prefix: 'auth_' });
  }

  recordLogin(duration, success) {
    authLoginDuration.add(duration);
    authFailures.add(success ? 0 : 1);
  }

  recordRefresh(duration) {
    authRefreshDuration.add(duration);
  }
}

export class RegistryMetrics extends MetricsCollector {
  constructor() {
    super({ prefix: 'registry_' });
  }

  recordHeartbeat(duration, success) {
    registryHeartbeatDuration.add(duration);
    registryHeartbeatRate.add(success ? 1 : 0);
  }

  recordDiscovery(duration) {
    registryDiscoveryDuration.add(duration);
  }
}

export class JobMetrics extends MetricsCollector {
  constructor() {
    super({ prefix: 'job_' });
  }

  recordSubmit(duration, success) {
    jobSubmitDuration.add(duration);
    jobSubmissionRate.add(success ? 1 : 0);
    if (success) {
      printJobsCreated.add(1);
    }
  }

  recordStatusQuery(duration) {
    jobStatusQueryDuration.add(duration);
  }

  setActiveJobs(count) {
    activeJobs.set(count);
  }

  incrementCompleted() {
    completedJobs.add(1);
  }

  incrementFailed() {
    failedJobs.add(1);
  }
}

export class StorageMetrics extends MetricsCollector {
  constructor() {
    super({ prefix: 'storage_' });
  }

  recordUpload(duration, bytes, success) {
    storageUploadDuration.add(duration);
    storageBytesUploaded.add(bytes);
    storageUploadRate.add(success ? 1 : 0);
    if (success) {
      documentsStored.add(1);
    }
  }

  recordDownload(duration, bytes) {
    storageDownloadDuration.add(duration);
    storageBytesDownloaded.add(bytes);
  }
}

export class NotificationMetrics extends MetricsCollector {
  constructor() {
    super({ prefix: 'notification_' });
  }

  recordConnect(duration) {
    notificationConnectDuration.add(duration);
    notificationConnections.add(1);
  }

  recordMessageSent() {
    notificationMessagesSent.add(1);
    wsMessagesSent.add(1);
  }

  recordMessageReceived(latency) {
    notificationMessagesReceived.add(1);
    notificationMessageLatency.add(latency);
    wsMessagesReceived.add(1);
  }

  recordReconnect() {
    notificationReconnects.add(1);
  }
}

/**
 * Baseline metrics for regression detection
 */
export const BaselineMetrics = {
  auth: {
    p95: 300,
    p99: 500,
    maxErrorRate: 0.01,
  },
  registry: {
    heartbeat: {
      p95: 100,
      p99: 200,
      maxErrorRate: 0.01,
    },
    discovery: {
      p95: 500,
      p99: 1000,
      maxErrorRate: 0.01,
    },
  },
  job: {
    submit: {
      p95: 500,
      p99: 1000,
      maxErrorRate: 0.01,
    },
    status: {
      p95: 200,
      p99: 400,
      maxErrorRate: 0.01,
    },
  },
  storage: {
    upload: {
      p95: 2000,
      p99: 5000,
      maxErrorRate: 0.01,
    },
    download: {
      p95: 1000,
      p99: 2000,
      maxErrorRate: 0.01,
    },
  },
  notification: {
    connect: {
      p95: 500,
      p99: 1000,
      maxErrorRate: 0.05,
    },
  },
};

/**
 * Compare current metrics against baseline
 * Returns warnings for any metrics that exceed baseline thresholds
 */
export function compareWithBaseline(currentMetrics, baseline = BaselineMetrics) {
  const warnings = [];

  for (const [service, metrics] of Object.entries(currentMetrics)) {
    const serviceBaseline = baseline[service];
    if (!serviceBaseline) continue;

    for (const [metricName, value] of Object.entries(metrics)) {
      const metricBaseline = serviceBaseline[metricName];
      if (!metricBaseline) continue;

      if (value.p95 > metricBaseline.p95) {
        warnings.push(
          `${service}.${metricName}: p95 (${value.p95}ms) exceeds baseline (${metricBaseline.p95}ms)`
        );
      }

      if (value.errorRate > metricBaseline.maxErrorRate) {
        warnings.push(
          `${service}.${metricName}: error rate (${(value.errorRate * 100).toFixed(2)}%) exceeds baseline (${(metricBaseline.maxErrorRate * 100).toFixed(2)}%)`
        );
      }
    }
  }

  return warnings;
}

/**
 * Export metrics as JSON for CI/CD integration
 */
export function exportMetrics() {
  return {
    auth: {
      login: {
        p95: authLoginDuration.p('95'),
        p99: authLoginDuration.p('99'),
        avg: authLoginDuration.avg,
        min: authLoginDuration.min,
        max: authLoginDuration.max,
      },
    },
    registry: {
      heartbeat: {
        p95: registryHeartbeatDuration.p('95'),
        p99: registryHeartbeatDuration.p('99'),
        avg: registryHeartbeatDuration.avg,
      },
    },
    job: {
      submit: {
        p95: jobSubmitDuration.p('95'),
        p99: jobSubmitDuration.p('99'),
        avg: jobSubmitDuration.avg,
      },
    },
    storage: {
      upload: {
        p95: storageUploadDuration.p('95'),
        p99: storageUploadDuration.p('99'),
        avg: storageUploadDuration.avg,
      },
    },
  };
}

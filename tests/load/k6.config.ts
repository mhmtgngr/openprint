/**
 * Central configuration for all k6 load tests
 *
 * This file exports configuration objects used across all load test scenarios.
 * It defines test stages, thresholds, and service endpoints.
 */

import { SharedOptions } from 'k6';

// Service base URLs - configurable via environment variables
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8001';
const REGISTRY_URL = __ENV.REGISTRY_URL || 'http://localhost:8002';
const JOB_URL = __ENV.JOB_URL || 'http://localhost:8003';
const STORAGE_URL = __ENV.STORAGE_URL || 'http://localhost:8004';
const NOTIFICATION_URL = __ENV.NOTIFICATION_URL || 'http://localhost:8005';

// Test configuration presets
export const config = {
  // Service endpoints
  endpoints: {
    auth: BASE_URL,
    registry: REGISTRY_URL,
    job: JOB_URL,
    storage: STORAGE_URL,
    notification: NOTIFICATION_URL,
  },

  // Test user credentials (for testing only)
  testUser: {
    email: __ENV.TEST_USER_EMAIL || 'loadtest@example.com',
    password: __ENV.TEST_USER_PASSWORD || 'TestPassword123!',
  },

  // Stage configurations for different test types
  stages: {
    // Quick smoke test
    smoke: [
      { duration: '5s', target: 1 },
      { duration: '10s', target: 5 },
      { duration: '5s', target: 0 },
    ],

    // Standard baseline test
    baseline: [
      { duration: '30s', target: 10 },
      { duration: '1m', target: 50 },
      { duration: '30s', target: 10 },
      { duration: '10s', target: 0 },
    ],

    // Load test
    load: [
      { duration: '1m', target: 50 },
      { duration: '2m', target: 200 },
      { duration: '2m', target: 200 },
      { duration: '1m', target: 50 },
      { duration: '30s', target: 0 },
    ],

    // Stress test
    stress: [
      { duration: '1m', target: 100 },
      { duration: '2m', target: 500 },
      { duration: '2m', target: 1000 },
      { duration: '1m', target: 0 },
    ],

    // Spike test
    spike: [
      { duration: '30s', target: 50 },
      { duration: '10s', target: 500 },
      { duration: '20s', target: 500 },
      { duration: '10s', target: 50 },
      { duration: '30s', target: 0 },
    ],

    // Soak test (endurance)
    soak: [
      { duration: '5m', target: 100 },
      { duration: '30m', target: 100 },
      { duration: '5m', target: 0 },
    ],

    // High-frequency heartbeat simulation
    heartbeat: [
      { duration: '1m', target: 1000 },
      { duration: '5m', target: 5000 },
      { duration: '5m', target: 5000 },
      { duration: '1m', target: 0 },
    ],
  },

  // Performance thresholds
  thresholds: {
    // HTTP request thresholds
    http_req_duration: ['p(95)<500', 'p(99)<1000'], // 95% under 500ms, 99% under 1s
    http_req_failed: ['rate<0.01'], // Less than 1% failure rate

    // Authentication thresholds
    auth_login_duration: ['p(95)<300', 'p(99)<500'],
    auth_token_refresh_duration: ['p(95)<200'],

    // Registry thresholds
    registry_heartbeat_duration: ['p(95)<100'],
    registry_agent_discovery_duration: ['p(95)<500'],

    // Job service thresholds
    job_submit_duration: ['p(95)<500', 'p(99)<1000'],
    job_status_query_duration: ['p(95)<200'],

    // Storage thresholds
    storage_upload_duration: ['p(95)<2000', 'p(99)<5000'],
    storage_download_duration: ['p(95)<1000'],

    // Notification thresholds
    notification_connect_duration: ['p(95)<500'],
    notification_message_latency: ['p(95)<100'],
  },
};

// Common k6 options for all tests
export const commonOptions: SharedOptions = {
  scenarios: {
    default: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: config.stages.baseline,
    },
  },
  thresholds: config.thresholds,
  // No default timeout - let individual scenarios configure
};

// Auth service options
export const authOptions: SharedOptions = {
  ...commonOptions,
  thresholds: {
    ...config.thresholds,
    'http_req_duration{scenario:default}': ['p(95)<300', 'p(99)<500'],
  },
};

// Registry service options
export const registryOptions: SharedOptions = {
  ...commonOptions,
  thresholds: {
    ...config.thresholds,
    'http_req_duration{scenario:default}': ['p(95)<200', 'p(99)<400'],
  },
};

// Job service options
export const jobOptions: SharedOptions = {
  ...commonOptions,
  thresholds: {
    ...config.thresholds,
    'http_req_duration{scenario:default}': ['p(95)<500', 'p(99)<1000'],
  },
};

// Storage service options
export const storageOptions: SharedOptions = {
  ...commonOptions,
  thresholds: {
    ...config.thresholds,
    'http_req_duration{scenario:default}': ['p(95)<2000', 'p(99)<5000'],
  },
};

// Notification service options
export const notificationOptions: SharedOptions = {
  ...commonOptions,
  thresholds: {
    ...config.thresholds,
    'http_req_duration{scenario:default}': ['p(95)<500'],
  },
};

// WebSocket options for notification tests
export const wsOptions: SharedOptions = {
  ...commonOptions,
  thresholds: {
    'ws Connecting': ['rate<0.05'], // Less than 5% connection failures
    'ws Messages Received': ['count>0'],
  },
};

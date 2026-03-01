/**
 * Shared configuration module for k6 tests
 *
 * This module provides centralized configuration for all load tests.
 * Import this in any test file to access service URLs and test parameters.
 */

// Service endpoints
export const BASE_URL = __ENV.BASE_URL || 'http://localhost:8001';
export const REGISTRY_URL = __ENV.REGISTRY_URL || 'http://localhost:8002';
export const JOB_URL = __ENV.JOB_URL || 'http://localhost:8003';
export const STORAGE_URL = __ENV.STORAGE_URL || 'http://localhost:8004';
export const NOTIFICATION_URL = __ENV.NOTIFICATION_URL || 'http://localhost:8005';
export const GATEWAY_URL = __ENV.GATEWAY_URL || 'http://localhost:8080';

// API Paths - aligned with actual service endpoints
export const API_PATHS = {
  // Auth service
  AUTH_LOGIN: '/auth/login',
  AUTH_REGISTER: '/auth/register',
  AUTH_REFRESH: '/auth/refresh',
  AUTH_LOGOUT: '/auth/logout',
  AUTH_ME: '/auth/me',  // Changed from AUTH_PROFILE - actual endpoint is /auth/me

  // Registry service
  AGENTS: '/agents',
  AGENT_HEARTBEAT: (id) => `/agents/${id}/heartbeat`,  // POST to agent heartbeat endpoint
  AGENT_HEARTBEAT_STATUS: (id) => `/agents/${id}/heartbeat/status`,  // GET heartbeat status
  AGENT_REGISTER: '/agents/register',
  PRINTERS: '/printers',
  PRINTER_REGISTER: '/printers/register',
  USER_PRINTER_MAPPINGS: '/user-printer-mappings',

  // Job service
  JOBS: '/jobs',
  JOB_BY_ID: (id) => `/jobs/${id}`,  // Changed from JOB_STATUS
  JOB_STATUS: (id) => `/jobs/status/${id}`,  // Dedicated status endpoint
  HISTORY: '/history',  // Changed from /jobs/history
  QUEUE_STATS: '/queue/stats',  // Changed from /jobs/queue/stats

  // Storage service
  DOCUMENTS: '/documents',
  DOCUMENT: (id) => `/documents/${id}`,
  UPLOAD: '/upload',
  DOWNLOAD: (path) => `/download/${path}`,

  // Notification service (placeholder - to be implemented)
  WS_CONNECT: '/ws',
  BROADCAST: '/broadcast',
  CONNECTIONS: '/connections',
};

// Test credentials
export const TEST_CREDENTIALS = {
  email: __ENV.TEST_USER_EMAIL || 'loadtest@example.com',
  password: __ENV.TEST_USER_PASSWORD || 'TestPassword123!',
  // Pre-created admin user for tests
  adminEmail: __ENV.ADMIN_EMAIL || 'admin@openprint.local',
  adminPassword: __ENV.ADMIN_PASSWORD || 'Admin123!',
};

// Performance thresholds per service
export const THRESHOLDS = {
  auth: {
    http_req_duration: ['p(95)<300', 'p(99)<500'],
    http_req_failed: ['rate<0.01'],
  },
  registry: {
    http_req_duration: ['p(95)<200', 'p(99)<400'],
    http_req_failed: ['rate<0.01'],
  },
  job: {
    http_req_duration: ['p(95)<500', 'p(99)<1000'],
    http_req_failed: ['rate<0.01'],
  },
  storage: {
    http_req_duration: ['p(95)<2000', 'p(99)<5000'],
    http_req_failed: ['rate<0.01'],
  },
  notification: {
    http_req_duration: ['p(95)<500'],
    http_req_failed: ['rate<0.05'], // Allow higher failure rate for WS
  },
};

// Stage configurations
export const STAGES = {
  smoke: [
    { duration: '5s', target: 1 },
    { duration: '10s', target: 5 },
    { duration: '5s', target: 0 },
  ],
  baseline: [
    { duration: '30s', target: 10 },
    { duration: '1m', target: 50 },
    { duration: '30s', target: 10 },
    { duration: '10s', target: 0 },
  ],
  load: [
    { duration: '1m', target: 50 },
    { duration: '2m', target: 200 },
    { duration: '2m', target: 200 },
    { duration: '1m', target: 50 },
    { duration: '30s', target: 0 },
  ],
  stress: [
    { duration: '1m', target: 100 },
    { duration: '2m', target: 500 },
    { duration: '2m', target: 1000 },
    { duration: '1m', target: 0 },
  ],
  spike: [
    { duration: '30s', target: 50 },
    { duration: '10s', target: 500 },
    { duration: '20s', target: 500 },
    { duration: '10s', target: 50 },
    { duration: '30s', target: 0 },
  ],
  soak: [
    { duration: '5m', target: 100 },
    { duration: '30m', target: 100 },
    { duration: '5m', target: 0 },
  ],
  heartbeat: [
    { duration: '1m', target: 1000 },
    { duration: '5m', target: 5000 },
    { duration: '5m', target: 5000 },
    { duration: '1m', target: 0 },
  ],
};

// Test data generators
export const TestData = {
  // Generate random email
  email: () => `user_${Math.random().toString(36).substring(7)}@example.com`,

  // Generate random password
  password: () => `Pass${Math.random().toString(36).substring(7)}!123`,

  // Generate random user name
  userName: () => `Test User ${Math.floor(Math.random() * 10000)}`,

  // Generate random printer name
  printerName: () => `TestPrinter_${Math.floor(Math.random() * 1000)}`,

  // Generate random document title
  documentTitle: () => `Test Document ${Math.random().toString(36).substring(7)}`,

  // Generate random agent ID
  agentId: () => `agent_${Math.random().toString(36).substring(7)}`,

  // Generate random printer ID (UUID format)
  printerId: () => {
    return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
      const r = Math.random() * 16 | 0;
      const v = c === 'x' ? r : (r & 0x3 | 0x8);
      return v.toString(16);
    });
  },

  // Generate random file data
  fileData: (sizeInKB = 100) => {
    const data = [];
    for (let i = 0; i < sizeInKB * 1024; i++) {
      data.push(String.fromCharCode(65 + (i % 26)));
    }
    return data.join('');
  },
};

// Helper to get configuration from environment
export function getEnv(key, defaultValue) {
  return __ENV[key] || defaultValue;
}

// Helper to determine test environment
export function getTestEnvironment() {
  const env = __ENV.TEST_ENV || 'local';
  const environments = {
    local: {
      baseURL: 'http://localhost:8001',
      registryURL: 'http://localhost:8002',
      jobURL: 'http://localhost:8003',
      storageURL: 'http://localhost:8004',
      notificationURL: 'http://localhost:8005',
    },
    staging: {
      baseURL: 'https://api-staging.openprint.local',
      registryURL: 'https://api-staging.openprint.local',
      jobURL: 'https://api-staging.openprint.local',
      storageURL: 'https://api-staging.openprint.local',
      notificationURL: 'https://api-staging.openprint.local',
    },
    production: {
      baseURL: 'https://api.openprint.cloud',
      registryURL: 'https://api.openprint.cloud',
      jobURL: 'https://api.openprint.cloud',
      storageURL: 'https://api.openprint.cloud',
      notificationURL: 'https://api.openprint.cloud',
    },
  };
  return environments[env] || environments.local;
}

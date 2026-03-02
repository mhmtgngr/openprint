/**
 * Test data fixtures for E2E tests
 */

export const testUsers = {
  admin: {
    email: 'admin@openprint.test',
    password: 'TestAdmin123!',
    name: 'Admin User',
    role: 'admin',
  },
  user: {
    email: 'user@openprint.test',
    password: 'TestUser123!',
    name: 'Regular User',
    role: 'user',
  },
  owner: {
    email: 'owner@openprint.test',
    password: 'TestOwner123!',
    name: 'Owner User',
    role: 'owner',
  },
} as const;

export const testPrinters = {
  hpLaserjet: {
    name: 'HP LaserJet Pro M404n',
    type: 'laser',
    ip: '192.168.1.100',
    port: 9100,
    isOnline: true,
    isActive: true,
  },
  canonImageRunner: {
    name: 'Canon imageRUNNER 1435iF',
    type: 'mfp',
    ip: '192.168.1.101',
    port: 9100,
    isOnline: true,
    isActive: true,
  },
  epsonEcoTank: {
    name: 'Epson EcoTank ET-4760',
    type: 'inkjet',
    ip: '192.168.1.102',
    port: 9100,
    isOnline: false,
    isActive: true,
  },
} as const;

export const testDocuments = {
  pdf: {
    name: 'Test Document.pdf',
    pages: 5,
    size: 102400, // 100KB
    type: 'application/pdf',
  },
  docx: {
    name: 'Meeting Notes.docx',
    pages: 2,
    size: 51200, // 50KB
    type: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
  },
} as const;

export const testPolicies = {
  colorRestriction: {
    name: 'Color Printing Restriction',
    description: 'Limit color printing to admin users only',
    priority: 1,
    isEnabled: true,
    conditions: {
      userRole: ['user'],
    },
    actions: {
      forceGrayscale: true,
    },
  },
  duplexDefault: {
    name: 'Duplex Default',
    description: 'Enable double-sided printing by default',
    priority: 2,
    isEnabled: true,
    conditions: {
      always: true,
    },
    actions: {
      forceDuplex: true,
    },
  },
} as const;

export const testQuotas = {
  monthly: {
    name: 'Monthly Page Limit',
    type: 'monthly',
    limit: 1000,
    resetDay: 1,
  },
  weekly: {
    name: 'Weekly Page Limit',
    type: 'weekly',
    limit: 250,
    resetDay: 1, // Monday
  },
} as const;

// Observability test data
export const testAlerts = {
  highErrorRate: {
    name: 'HighErrorRate',
    severity: 'critical',
    state: 'firing',
    service: 'auth-service',
    message: 'Error rate is above 5% for the last 5 minutes',
  },
  highLatency: {
    name: 'HighLatency',
    severity: 'warning',
    state: 'firing',
    service: 'job-service',
    message: 'P95 latency is above 500ms for the last 10 minutes',
  },
  serviceDown: {
    name: 'ServiceDown',
    severity: 'critical',
    state: 'pending',
    service: 'storage-service',
    message: 'Service has been down for more than 1 minute',
  },
  highMemory: {
    name: 'HighMemoryUsage',
    severity: 'warning',
    state: 'resolved',
    service: 'registry-service',
    message: 'Memory usage is above 80%',
  },
} as const;

export const testServices = {
  authService: {
    name: 'auth-service',
    status: 'healthy',
    cpu: 45.2,
    memory: 62.8,
    requestRate: 125.5,
    errorRate: 0.02,
    p95Latency: 45,
  },
  jobService: {
    name: 'job-service',
    status: 'healthy',
    cpu: 32.1,
    memory: 55.4,
    requestRate: 89.3,
    errorRate: 0.01,
    p95Latency: 78,
  },
  registryService: {
    name: 'registry-service',
    status: 'degraded',
    cpu: 78.5,
    memory: 82.3,
    requestRate: 45.2,
    errorRate: 2.1,
    p95Latency: 120,
  },
  storageService: {
    name: 'storage-service',
    status: 'healthy',
    cpu: 28.9,
    memory: 48.7,
    requestRate: 34.1,
    errorRate: 0.0,
    p95Latency: 156,
  },
  notificationService: {
    name: 'notification-service',
    status: 'healthy',
    cpu: 15.4,
    memory: 35.2,
    requestRate: 12.8,
    errorRate: 0.0,
    p95Latency: 23,
  },
} as const;

export const testTraces = {
  authFlow: {
    traceId: 'trace-abc-123',
    rootSpanName: 'POST /api/v1/auth/login',
    rootServiceName: 'auth-service',
    duration: 45000000, // 45ms in nanoseconds
    spanCount: 5,
    hasError: false,
  },
  jobProcessing: {
    traceId: 'trace-def-456',
    rootSpanName: 'ProcessPrintJob',
    rootServiceName: 'job-service',
    duration: 250000000, // 250ms in nanoseconds
    spanCount: 12,
    hasError: false,
  },
  documentRetrieval: {
    traceId: 'trace-ghi-789',
    rootSpanName: 'GET /api/v1/documents/:id',
    rootServiceName: 'storage-service',
    duration: 89000000, // 89ms in nanoseconds
    spanCount: 4,
    hasError: true, // Error trace
  },
} as const;

export const testMetrics = {
  requestRate: {
    query: 'sum(rate(http_requests_total[5m])) by (service)',
    labels: ['auth-service', 'job-service'],
    values: [125.5, 89.3],
  },
  errorRate: {
    query: 'sum(rate(http_requests_total{status=~"5.."}[5m])) by (service)',
    labels: ['registry-service'],
    values: [2.1],
  },
  latency: {
    query: 'histogram_quantile(0.95, sum(rate(http_request_duration_ms_bucket[5m])) by (le, service))',
    labels: ['job-service', 'storage-service'],
    values: [78, 156],
  },
} as const;

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

// Compliance test data
export const testComplianceStandards = {
  fedramp: {
    name: 'FedRAMP',
    status: 'compliant',
    lastAudit: '2024-01-15',
    description: 'Federal Risk and Authorization Management Program',
    level: 'Moderate',
  },
  hipaa: {
    name: 'HIPAA',
    status: 'compliant',
    lastAudit: '2024-01-15',
    description: 'Health Insurance Portability and Accountability Act',
    version: 'Security Rule 2023',
  },
  gdpr: {
    name: 'GDPR',
    status: 'compliant',
    lastAudit: '2024-01-15',
    description: 'General Data Protection Regulation',
    version: 'EU 2016/679',
  },
  soc2: {
    name: 'SOC 2',
    status: 'in_progress',
    lastAudit: '2024-01-15',
    description: 'Service Organization Control 2',
    type: 'Type II',
  },
} as const;

export const testAuditLogs = {
  loginSuccess: {
    id: 'audit-1',
    timestamp: '2024-03-04T10:30:00Z',
    user: 'admin@openprint.test',
    action: 'login',
    category: 'authentication',
    resource: '/login',
    details: 'Successful login',
    outcome: 'success',
    ipAddress: '192.168.1.100',
    userAgent: 'Mozilla/5.0',
  },
  printJobCreated: {
    id: 'audit-2',
    timestamp: '2024-03-04T10:25:00Z',
    user: 'user@openprint.test',
    action: 'print_job_created',
    category: 'print_job',
    resource: '/api/v1/jobs',
    details: 'Created job "Quarterly Report.pdf"',
    outcome: 'success',
    ipAddress: '192.168.1.101',
    userAgent: 'Mozilla/5.0',
  },
  policyUpdated: {
    id: 'audit-3',
    timestamp: '2024-03-04T10:20:00Z',
    user: 'admin@openprint.test',
    action: 'policy_updated',
    category: 'policy',
    resource: '/api/v1/policies/pol-123',
    details: 'Updated policy "Color Restriction"',
    outcome: 'success',
    ipAddress: '192.168.1.100',
    userAgent: 'Mozilla/5.0',
  },
  failedLogin: {
    id: 'audit-4',
    timestamp: '2024-03-04T10:15:00Z',
    user: 'unknown@hacker.test',
    action: 'login',
    category: 'authentication',
    resource: '/login',
    details: 'Failed login attempt - invalid credentials',
    outcome: 'failure',
    ipAddress: '203.0.113.42',
    userAgent: 'Mozilla/5.0',
  },
  documentUploaded: {
    id: 'audit-5',
    timestamp: '2024-03-04T10:10:00Z',
    user: 'user@openprint.test',
    action: 'document_uploaded',
    category: 'document',
    resource: '/api/v1/documents',
    details: 'Uploaded "Confidential Memo.pdf"',
    outcome: 'success',
    ipAddress: '192.168.1.101',
    userAgent: 'Mozilla/5.0',
  },
} as const;

export const testComplianceReports = {
  fedrampAssessment: {
    id: 'report-fedramp-2024',
    name: 'FedRAMP Moderate Assessment 2024',
    type: 'fedramp',
    framework: 'FedRAMP',
    status: 'complete',
    createdAt: '2024-01-15T00:00:00Z',
    generatedBy: 'admin@openprint.test',
    overallStatus: 'compliant',
    compliantCount: 145,
    nonCompliantCount: 0,
    pendingCount: 5,
  },
  hipaaAudit: {
    id: 'report-hipaa-2024',
    name: 'HIPAA Security Rule Audit 2024',
    type: 'hipaa',
    framework: 'HIPAA',
    status: 'complete',
    createdAt: '2024-02-01T00:00:00Z',
    generatedBy: 'compliance@openprint.test',
    overallStatus: 'compliant',
    compliantCount: 78,
    nonCompliantCount: 0,
    pendingCount: 2,
  },
  soc2Audit: {
    id: 'report-soc2-2024',
    name: 'SOC 2 Type II Audit 2024',
    type: 'soc2',
    framework: 'SOC 2',
    status: 'generating',
    createdAt: '2024-03-01T00:00:00Z',
    generatedBy: 'admin@openprint.test',
    overallStatus: 'pending',
    compliantCount: 0,
    nonCompliantCount: 0,
    pendingCount: 64,
  },
} as const;

export const testComplianceChecks = {
  accessControl: {
    id: 'check-ac-1',
    category: 'Access Control',
    control: 'AC-1: Access Control Policy and Procedures',
    status: 'pass',
    description: 'System has documented access control policies',
    evidence: 'Policy document AC-001 exists and is approved',
    lastChecked: '2024-03-04T00:00:00Z',
  },
  auditLogging: {
    id: 'check-au-2',
    category: 'Audit and Accountability',
    control: 'AU-2: Audit Events',
    status: 'pass',
    description: 'Audit records are generated and stored',
    evidence: 'Audit logs configured for all critical services',
    lastChecked: '2024-03-04T00:00:00Z',
  },
  encryption: {
    id: 'check-sc-13',
    category: 'System and Communications Protection',
    control: 'SC-13: Cryptographic Protection',
    status: 'pass',
    description: 'Data is encrypted at rest and in transit',
    evidence: 'AES-256 encryption enabled for all data',
    lastChecked: '2024-03-04T00:00:00Z',
  },
  incidentResponse: {
    id: 'check-ir-4',
    category: 'Incident Response',
    control: 'IR-4: Incident Handling',
    status: 'warning',
    description: 'Incident response procedures need updating',
    evidence: 'Last tabletop exercise was 18 months ago',
    lastChecked: '2024-03-04T00:00:00Z',
  },
  securityTraining: {
    id: 'check-at-2',
    category: 'Awareness and Training',
    control: 'AT-2: Security Awareness Training',
    status: 'pending',
    description: 'Annual security training not completed',
    evidence: '35% of users have not completed 2024 training',
    lastChecked: '2024-03-04T00:00:00Z',
  },
} as const;

export const testDataBreaches = {
  none: {
    id: 'breach-none',
    status: 'no_breaches',
    count: 0,
    lastBreach: null,
  },
  historical: {
    id: 'breach-2023-001',
    status: 'resolved',
    severity: 'low',
    description: 'Minor data exposure during maintenance',
    discoveredAt: '2023-06-15T00:00:00Z',
    resolvedAt: '2023-06-16T00:00:00Z',
    affectedRecords: 12,
    containmentStatus: 'contained',
  },
} as const;

export const testRetentionPolicies = {
  default: {
    id: 'retention-default',
    name: 'Default Retention Policy',
    period: 2555, // 7 years in days
    periodUnit: 'days',
    autoDelete: true,
    appliesTo: ['audit_logs', 'access_logs', 'session_logs'],
  },
  hipaa: {
    id: 'retention-hipaa',
    name: 'HIPAA Retention Policy',
    period: 3650, // 10 years in days
    periodUnit: 'days',
    autoDelete: true,
    appliesTo: ['phi_audit_logs', 'access_logs'],
  },
} as const;

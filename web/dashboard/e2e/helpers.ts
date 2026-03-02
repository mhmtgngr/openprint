/**
 * Test helpers and utilities for E2E tests
 */
import type { Page, Route } from '@playwright/test';

export interface Credentials {
  email: string;
  password: string;
  name?: string;
}

export const testCredentials: Credentials = {
  email: 'test@example.com',
  password: 'TestPassword123!',
  name: 'Test User',
};

export const adminCredentials: Credentials = {
  email: 'admin@example.com',
  password: 'AdminPassword123!',
  name: 'Admin User',
};

/**
 * Sets up an authenticated user by mocking auth endpoints and setting localStorage
 * This is the preferred way to authenticate tests as it doesn't rely on actual login flow
 */
export async function setupAuthenticatedUser(
  page: Page,
  user: typeof mockUsers[number] = mockUsers[0]
) {
  // Set auth tokens in localStorage using addInitScript for new page contexts
  await page.addInitScript((tokens) => {
    localStorage.setItem('auth_tokens', JSON.stringify(tokens));
  }, {
    accessToken: 'mock-access-token',
    refreshToken: 'mock-refresh-token',
  });

  // Also set tokens directly for already-loaded pages
  await setAuthTokens(page);

  // Mock ALL auth-related endpoints BEFORE any navigation
  await page.route('**/api/v1/auth/me', async (route) => {
    await mockApiResponse(route, user);
  });

  await page.route('**/api/v1/auth/login', async (route) => {
    await mockApiResponse(route, {
      userId: user.id,
      access_token: 'mock-access-token',
      refresh_token: 'mock-refresh-token',
      org: { id: 'org-1', name: 'Test Org' },
    });
  });

  await page.route('**/api/v1/auth/refresh', async (route) => {
    await mockApiResponse(route, {
      access_token: 'mock-refreshed-access-token',
      refresh_token: 'mock-refresh-token',
    });
  });
}

/**
 * Sets auth tokens in localStorage for an already-loaded page
 */
async function setAuthTokens(page: Page) {
  await page.evaluate(() => {
    const tokens = {
      accessToken: 'mock-access-token',
      refreshToken: 'mock-refresh-token',
    };
    localStorage.setItem('auth_tokens', JSON.stringify(tokens));
  });
}

/**
 * Login helper - authenticates a user and navigates to dashboard
 * Uses mocked auth endpoints for reliability
 */
export async function login(
  page: Page,
  credentials: Credentials = testCredentials,
  user: typeof mockUsers[number] = mockUsers[0]
) {
  // Set auth tokens in localStorage for the current context
  await setAuthTokens(page);

  // Mock auth endpoints
  await page.route('**/api/v1/auth/me', async (route) => {
    await mockApiResponse(route, user);
  });

  await page.route('**/api/v1/auth/login', async (route) => {
    await mockApiResponse(route, {
      userId: user.id,
      access_token: 'mock-access-token',
      refresh_token: 'mock-refresh-token',
      org: { id: 'org-1', name: 'Test Org' },
    });
  });

  await page.route('**/api/v1/auth/refresh', async (route) => {
    await mockApiResponse(route, {
      access_token: 'mock-refreshed-access-token',
      refresh_token: 'mock-refresh-token',
    });
  });

  // Mock common dashboard APIs that are called during login flow
  await page.route('**/api/v1/printers', async (route) => {
    await mockApiResponse(route, { printers: mockPrinters });
  });

  await page.route('**/api/v1/jobs*', async (route) => {
    await mockApiResponse(route, {
      data: mockJobs,
      total: mockJobs.length,
      limit: 50,
      offset: 0,
    });
  });

  await page.route('**/api/v1/analytics/environment*', async (route) => {
    await mockApiResponse(route, mockEnvironmentReport);
  });

  // Navigate to login page
  await page.goto('/login', { waitUntil: 'networkidle' });

  // Wait for email input to be visible
  await page.waitForSelector('input[type="email"]', { state: 'visible', timeout: 5000 });

  // Fill in login form
  await page.fill('input[type="email"]', credentials.email);
  await page.fill('input[type="password"]', credentials.password);

  // Submit form
  await page.click('button[type="submit"]');

  // Wait for navigation to dashboard
  await page.waitForURL('**/dashboard', { timeout: 10000 });

  // Re-ensure tokens are set after login for subsequent navigations
  await setAuthTokens(page);
}

/**
 * Quick setup for authenticated tests - sets up auth and navigates to a specific page
 * This bypasses the login form for faster, more reliable tests
 */
export async function setupAuthAndNavigate(
  page: Page,
  path: string,
  user: typeof mockUsers[number] = mockUsers[0]
) {
  // Set up authenticated state with all necessary mocks
  await setupAuthenticatedUser(page, user);

  // Also mock common APIs that most pages need
  await page.route('**/api/v1/printers', async (route) => {
    await mockApiResponse(route, { printers: mockPrinters });
  });

  await page.route('**/api/v1/jobs*', async (route) => {
    await mockApiResponse(route, {
      data: mockJobs,
      total: mockJobs.length,
      limit: 50,
      offset: 0,
    });
  });

  await page.route('**/api/v1/analytics/environment*', async (route) => {
    await mockApiResponse(route, mockEnvironmentReport);
  });

  // Navigate to the path
  await page.goto(path, { waitUntil: 'networkidle' });
}

/**
 * Logout helper
 */
export async function logout(page: Page) {
  await page.click('button:has-text("Logout")');
  await page.waitForURL('**/login');
}

/**
 * Mock API response helper
 */
export async function mockApiResponse(route: Route, data: unknown, status = 200) {
  return route.fulfill({
    status,
    contentType: 'application/json',
    body: JSON.stringify(data),
  });
}

/**
 * Common mock data
 */
export const mockUsers = [
  {
    id: '1',
    email: 'test@example.com',
    name: 'Test User',
    role: 'user',
    orgId: 'org-1',
    isActive: true,
    emailVerified: true,
    createdAt: '2024-01-01T00:00:00Z',
  },
  {
    id: '2',
    email: 'admin@example.com',
    name: 'Admin User',
    role: 'admin',
    orgId: 'org-1',
    isActive: true,
    emailVerified: true,
    createdAt: '2024-01-01T00:00:00Z',
  },
  {
    id: '3',
    email: 'platform-admin@openprint.cloud',
    name: 'Platform Admin',
    role: 'platform_admin',
    isPlatformAdmin: true,
    orgId: 'platform',
    isActive: true,
    emailVerified: true,
    createdAt: '2024-01-01T00:00:00Z',
  },
];

export const mockPrinters = [
  {
    id: 'printer-1',
    name: 'HP LaserJet Pro',
    agentId: 'agent-1',
    orgId: 'org-1',
    type: 'network',
    capabilities: {
      supportsColor: true,
      supportsDuplex: true,
      supportedPaperSizes: ['A4', 'Letter'],
      resolution: '600x600',
    },
    isActive: true,
    isOnline: true,
    createdAt: '2024-01-01T00:00:00Z',
  },
  {
    id: 'printer-2',
    name: 'Canon PIXMA',
    agentId: 'agent-1',
    orgId: 'org-1',
    type: 'usb',
    capabilities: {
      supportsColor: true,
      supportsDuplex: false,
      supportedPaperSizes: ['A4', 'Letter', 'A3'],
      resolution: '4800x1200',
    },
    isActive: true,
    isOnline: false,
    createdAt: '2024-01-01T00:00:00Z',
  },
];

export const mockJobs = [
  {
    id: 'job-1',
    userId: '1',
    printerId: 'printer-1',
    orgId: 'org-1',
    status: 'completed',
    documentName: 'Document.pdf',
    documentType: 'application/pdf',
    pageCount: 5,
    colorPages: 2,
    fileSize: 1024000,
    settings: {
      color: true,
      duplex: true,
      paperSize: 'A4',
      copies: 1,
    },
    createdAt: '2024-02-27T10:00:00Z',
    completedAt: '2024-02-27T10:01:00Z',
    printer: mockPrinters[0],
  },
  {
    id: 'job-2',
    userId: '1',
    printerId: 'printer-1',
    orgId: 'org-1',
    status: 'processing',
    documentName: 'Presentation.pptx',
    documentType: 'application/vnd.openxmlformats-officedocument.presentationml.presentation',
    pageCount: 12,
    colorPages: 12,
    fileSize: 5120000,
    settings: {
      color: true,
      duplex: false,
      paperSize: 'A4',
      copies: 1,
    },
    createdAt: '2024-02-27T10:05:00Z',
    printer: mockPrinters[0],
  },
  {
    id: 'job-3',
    userId: '1',
    printerId: 'printer-2',
    orgId: 'org-1',
    status: 'failed',
    documentName: 'Large_File.pdf',
    documentType: 'application/pdf',
    pageCount: 50,
    colorPages: 0,
    fileSize: 10485760,
    settings: {
      color: false,
      duplex: true,
      paperSize: 'A4',
      copies: 1,
    },
    errorMessage: 'Printer offline',
    createdAt: '2024-02-27T09:00:00Z',
    printer: mockPrinters[1],
  },
];

export const mockOrganization = {
  id: 'org-1',
  name: 'Acme Corp',
  slug: 'acme-corp',
  plan: 'pro',
  settings: {},
  maxUsers: 50,
  maxPrinters: 20,
  createdAt: '2024-01-01T00:00:00Z',
};

export const mockEnvironmentReport = {
  pagesPrinted: 1234,
  co2Grams: 245.6,
  treesSaved: 0.12,
  period: '30d',
};

export const mockUsageStats = [
  {
    id: 'stat-1',
    orgId: 'org-1',
    statDate: '2024-02-27',
    pagesPrinted: 45,
    colorPages: 12,
    jobsCount: 8,
    jobsCompleted: 7,
    jobsFailed: 1,
    totalBytes: 5242880,
    estimatedCost: 2.35,
    co2Grams: 8.9,
    treesSaved: 0.004,
  },
  {
    id: 'stat-2',
    orgId: 'org-1',
    statDate: '2024-02-26',
    pagesPrinted: 32,
    colorPages: 8,
    jobsCount: 5,
    jobsCompleted: 5,
    jobsFailed: 0,
    totalBytes: 3145728,
    estimatedCost: 1.67,
    co2Grams: 6.3,
    treesSaved: 0.003,
  },
];

export const mockInvitations = [
  {
    id: 'inv-1',
    orgId: 'org-1',
    email: 'newuser@example.com',
    role: 'user',
    invitedBy: '2',
    expiresAt: '2024-03-27T00:00:00Z',
    createdAt: '2024-02-27T00:00:00Z',
  },
];

export const mockAuditLogs = [
  {
    id: 'audit-1',
    userId: '1',
    orgId: 'org-1',
    action: 'job.created',
    resourceType: 'job',
    resourceId: 'job-1',
    details: {},
    ipAddress: '192.168.1.100',
    userAgent: 'Mozilla/5.0...',
    timestamp: '2024-02-27T10:00:00Z',
  },
  {
    id: 'audit-2',
    userId: '2',
    orgId: 'org-1',
    action: 'user.role_changed',
    resourceType: 'user',
    resourceId: '1',
    details: { oldRole: 'user', newRole: 'admin' },
    ipAddress: '192.168.1.101',
    userAgent: 'Mozilla/5.0...',
    timestamp: '2024-02-27T09:00:00Z',
  },
];

// Agent-related mock data
export const mockAgents = [
  {
    id: 'agent-1',
    name: 'WORKSTATION-001',
    orgId: 'org-1',
    status: 'online',
    platform: 'windows',
    platformVersion: 'Windows 11 Pro',
    agentVersion: '1.0.0',
    ipAddress: '192.168.1.100',
    lastHeartbeat: new Date(Date.now() - 60000).toISOString(),
    capabilities: {
      supportedFormats: ['PDF', 'DOCX', 'XLSX'],
      maxJobSize: 104857600,
      supportsColor: true,
      supportsDuplex: true,
    },
    sessionState: 'active',
    printerCount: 2,
    jobQueueDepth: 0,
    createdAt: '2024-01-01T00:00:00Z',
    associatedUser: {
      id: 'user-1',
      name: 'John Doe',
      email: 'john@example.com',
    },
  },
  {
    id: 'agent-2',
    name: 'WORKSTATION-002',
    orgId: 'org-1',
    status: 'online',
    platform: 'windows',
    platformVersion: 'Windows 10 Pro',
    agentVersion: '1.0.0',
    ipAddress: '192.168.1.101',
    lastHeartbeat: new Date(Date.now() - 120000).toISOString(),
    capabilities: {
      supportedFormats: ['PDF', 'DOCX'],
      maxJobSize: 52428800,
      supportsColor: false,
      supportsDuplex: true,
    },
    sessionState: 'active',
    printerCount: 1,
    jobQueueDepth: 3,
    createdAt: '2024-01-02T00:00:00Z',
  },
  {
    id: 'agent-3',
    name: 'FINANCE-PC',
    orgId: 'org-1',
    status: 'offline',
    platform: 'windows',
    platformVersion: 'Windows 11 Enterprise',
    agentVersion: '0.9.5',
    ipAddress: '192.168.1.150',
    lastHeartbeat: new Date(Date.now() - 86400000).toISOString(),
    capabilities: {
      supportedFormats: ['PDF'],
      maxJobSize: 104857600,
      supportsColor: true,
      supportsDuplex: true,
    },
    sessionState: 'disconnected',
    printerCount: 1,
    createdAt: '2024-01-03T00:00:00Z',
  },
  {
    id: 'agent-4',
    name: 'RECEPTION-DESK',
    orgId: 'org-1',
    status: 'error',
    platform: 'windows',
    platformVersion: 'Windows 10 Pro',
    agentVersion: '1.0.1',
    ipAddress: '192.168.1.200',
    lastHeartbeat: new Date(Date.now() - 300000).toISOString(),
    capabilities: {
      supportedFormats: ['PDF', 'DOCX', 'PNG', 'JPG'],
      maxJobSize: 104857600,
      supportsColor: true,
      supportsDuplex: true,
      supportsLargeFormat: true,
    },
    sessionState: 'error',
    printerCount: 1,
    jobQueueDepth: 0,
    createdAt: '2024-01-15T00:00:00Z',
  },
];

export const mockDiscoveredPrinters = [
  {
    id: 'printer-discovered-1',
    agentId: 'agent-1',
    name: 'HP LaserJet Pro M404n',
    driver: 'HP Universal Printing PCL 6',
    port: '9100',
    type: 'network',
    capabilities: {
      supportsColor: false,
      supportsDuplex: true,
      supportedPaperSizes: ['A4', 'Letter', 'Legal'],
      resolution: '600 x 600 dpi',
      maxSheetCount: 250,
      supportedFormats: ['PDF', 'DOCX', 'XLSX'],
    },
    status: 'available',
    lastSeen: new Date(Date.now() - 300000).toISOString(),
    discoveredAt: '2024-02-01T10:00:00Z',
    isDefault: true,
  },
  {
    id: 'printer-discovered-2',
    agentId: 'agent-1',
    name: 'Canon PIXMA G6020',
    driver: 'Canon G6020 series',
    port: 'USB001',
    type: 'local',
    capabilities: {
      supportsColor: true,
      supportsDuplex: true,
      supportedPaperSizes: ['A4', 'Letter', 'A3', '4x6', '5x7'],
      resolution: '4800 x 1200 dpi',
      maxSheetCount: 100,
      supportedFormats: ['PDF', 'JPG', 'PNG'],
    },
    status: 'available',
    lastSeen: new Date(Date.now() - 600000).toISOString(),
    discoveredAt: '2024-02-01T10:00:00Z',
    isDefault: false,
  },
  {
    id: 'printer-discovered-3',
    agentId: 'agent-2',
    name: 'Brother HL-L5100DN',
    driver: 'Brother HL-L5100DN series',
    port: '192.168.1.50',
    type: 'network',
    capabilities: {
      supportsColor: false,
      supportsDuplex: true,
      supportedPaperSizes: ['A4', 'Letter'],
      resolution: '2400 x 600 dpi',
      maxSheetCount: 250,
      supportedFormats: ['PDF', 'DOCX'],
    },
    status: 'available',
    lastSeen: new Date(Date.now() - 180000).toISOString(),
    discoveredAt: '2024-02-10T14:30:00Z',
    isDefault: true,
  },
  {
    id: 'printer-discovered-4',
    agentId: 'agent-3',
    name: 'Epson WorkForce Pro WF-3730',
    driver: 'Eson WF-3730 Series',
    port: 'WSD',
    type: 'network',
    capabilities: {
      supportsColor: true,
      supportsDuplex: true,
      supportedPaperSizes: ['A4', 'Letter', 'A3'],
      resolution: '4800 x 1200 dpi',
      maxSheetCount: 500,
      supportedFormats: ['PDF', 'DOCX', 'XLSX', 'JPG'],
    },
    status: 'offline',
    lastSeen: new Date(Date.now() - 86400000).toISOString(),
    discoveredAt: '2024-01-20T09:00:00Z',
    isDefault: false,
  },
  {
    id: 'printer-discovered-5',
    agentId: 'agent-4',
    name: 'Xerox WorkCentre 6515',
    driver: 'Xerox Global Print Driver',
    port: '9100',
    type: 'network',
    capabilities: {
      supportsColor: true,
      supportsDuplex: true,
      supportedPaperSizes: ['A4', 'Letter', 'A3', 'Legal', '11x17'],
      resolution: '1200 x 1200 dpi',
      maxSheetCount: 500,
      supportedFormats: ['PDF', 'DOCX', 'XLSX', 'PPTX', 'JPG', 'PNG'],
    },
    status: 'error',
    lastSeen: new Date(Date.now() - 3600000).toISOString(),
    discoveredAt: '2024-02-15T11:00:00Z',
    isDefault: false,
  },
  {
    id: 'printer-discovered-6',
    agentId: 'agent-1',
    name: 'Shared HP on Reception',
    driver: 'HP Universal Printing PCL 6',
    port: 'SHARED',
    type: 'shared',
    capabilities: {
      supportsColor: false,
      supportsDuplex: true,
      supportedPaperSizes: ['A4', 'Letter'],
      resolution: '600 x 600 dpi',
      supportedFormats: ['PDF'],
    },
    status: 'available',
    lastSeen: new Date(Date.now() - 900000).toISOString(),
    discoveredAt: '2024-02-20T08:00:00Z',
    isDefault: false,
  },
];

export const mockJobAssignments = [
  {
    id: 'assignment-1',
    jobId: 'job-1',
    agentId: 'agent-1',
    userId: 'user-1',
    printerId: 'printer-discovered-1',
    status: 'assigned',
    priority: 1,
    assignedAt: '2024-02-27T10:00:00Z',
    job: {
      id: 'job-1',
      documentName: 'Quarterly Report.pdf',
      pageCount: 15,
    },
    agent: mockAgents[0],
    user: {
      id: 'user-1',
      name: 'John Doe',
      email: 'john@example.com',
    },
  },
  {
    id: 'assignment-2',
    jobId: 'job-2',
    agentId: 'agent-2',
    status: 'in_progress',
    priority: 2,
    assignedAt: '2024-02-27T09:30:00Z',
    job: {
      id: 'job-2',
      documentName: 'Sales Presentation.pptx',
      pageCount: 24,
    },
    agent: mockAgents[1],
  },
  {
    id: 'assignment-3',
    jobId: 'job-3',
    agentId: 'agent-1',
    status: 'completed',
    priority: 1,
    assignedAt: '2024-02-27T08:00:00Z',
    completedAt: '2024-02-27T08:05:00Z',
    job: {
      id: 'job-3',
      documentName: 'Invoice Template.docx',
      pageCount: 2,
    },
    agent: mockAgents[0],
  },
  {
    id: 'assignment-4',
    jobId: 'job-4',
    agentId: 'agent-3',
    status: 'failed',
    priority: 1,
    assignedAt: '2024-02-26T16:00:00Z',
    errorMessage: 'Agent offline - unable to process job',
    job: {
      id: 'job-4',
      documentName: 'Large Batch Job.pdf',
      pageCount: 100,
    },
    agent: mockAgents[2],
  },
  {
    id: 'assignment-5',
    jobId: 'job-5',
    status: 'pending',
    priority: 3,
    job: {
      id: 'job-5',
      documentName: 'Meeting Notes.pdf',
      pageCount: 3,
    },
  },
];

export const mockAgentHealthMetrics = {
  uptime: 86400,
  totalJobsProcessed: 150,
  successfulJobs: 145,
  failedJobs: 5,
  averageResponseTime: 2500,
  lastJobTime: new Date(Date.now() - 3600000).toISOString(),
  successRate: 96.7,
  weeklyJobCounts: [
    {
      date: new Date(Date.now() - 6 * 86400000).toISOString().split('T')[0],
      count: 20,
      success: 19,
      failed: 1,
    },
    {
      date: new Date(Date.now() - 5 * 86400000).toISOString().split('T')[0],
      count: 25,
      success: 24,
      failed: 1,
    },
    {
      date: new Date(Date.now() - 4 * 86400000).toISOString().split('T')[0],
      count: 18,
      success: 18,
      failed: 0,
    },
    {
      date: new Date(Date.now() - 3 * 86400000).toISOString().split('T')[0],
      count: 22,
      success: 21,
      failed: 1,
    },
    {
      date: new Date(Date.now() - 2 * 86400000).toISOString().split('T')[0],
      count: 30,
      success: 29,
      failed: 1,
    },
    {
      date: new Date(Date.now() - 86400000).toISOString().split('T')[0],
      count: 20,
      success: 19,
      failed: 1,
    },
    {
      date: new Date().toISOString().split('T')[0],
      count: 15,
      success: 15,
      failed: 0,
    },
  ],
};

/**
 * Setup common API mocks for authenticated pages
 * This sets up mocks for auth and common dashboard APIs
 */
export async function setupCommonApiMocks(page: Page, user: typeof mockUsers[number] = mockUsers[0]) {
  // Mock auth/me endpoint
  await page.route('**/api/v1/auth/me', async (route) => {
    await mockApiResponse(route, user);
  });

  // Mock auth/login endpoint (for login flow)
  await page.route('**/api/v1/auth/login', async (route) => {
    await mockApiResponse(route, {
      userId: user.id,
      access_token: 'mock-access-token',
      refresh_token: 'mock-refresh-token',
      org: { id: 'org-1', name: 'Test Org' },
    });
  });

  // Mock auth/refresh endpoint
  await page.route('**/api/v1/auth/refresh', async (route) => {
    await mockApiResponse(route, {
      access_token: 'mock-refreshed-access-token',
      refresh_token: 'mock-refresh-token',
    });
  });

  // Mock printers API - Dashboard expects { printers: [...] }
  await page.route('**/api/v1/printers', async (route) => {
    await mockApiResponse(route, { printers: mockPrinters });
  });

  // Mock jobs API with pagination wrapper
  await page.route('**/api/v1/jobs*', async (route) => {
    await mockApiResponse(route, {
      data: mockJobs,
      total: mockJobs.length,
      limit: 50,
      offset: 0,
    });
  });

  // Mock environment report API
  await page.route('**/api/v1/analytics/environment*', async (route) => {
    await mockApiResponse(route, mockEnvironmentReport);
  });
}

/**
 * Setup mocks for agents page
 */
export async function setupAgentsApiMocks(page: Page) {
  // Mock agents list endpoint
  await page.route('**/api/v1/agents**', async (route) => {
    const url = route.request().url();
    // Handle detail endpoint differently
    if (url.includes('/detail')) {
      const agentDetail = {
        ...mockAgents[0],
        printers: mockDiscoveredPrinters,
        jobHistory: [],
        healthMetrics: mockAgentHealthMetrics,
      };
      await mockApiResponse(route, agentDetail);
    } else if (url.includes('/health')) {
      await mockApiResponse(route, mockAgentHealthMetrics);
    } else if (url.includes('/printers')) {
      await mockApiResponse(route, mockDiscoveredPrinters);
    } else if (url.includes('/jobs')) {
      await mockApiResponse(route, []);
    } else {
      await mockApiResponse(route, mockAgents);
    }
  });

  // Mock discovered printers endpoint
  await page.route('**/api/v1/discovered-printers*', async (route) => {
    await mockApiResponse(route, {
      printers: mockDiscoveredPrinters,
      total: mockDiscoveredPrinters.length,
    });
  });
}

/**
 * Setup mocks for analytics page
 */
export async function setupAnalyticsApiMocks(page: Page) {
  // Mock usage stats
  await page.route('**/api/v1/analytics/usage*', async (route) => {
    await mockApiResponse(route, mockUsageStats);
  });

  // Mock audit logs
  await page.route('**/api/v1/analytics/audit-logs*', async (route) => {
    await mockApiResponse(route, {
      data: mockAuditLogs,
      total: mockAuditLogs.length,
      limit: 50,
      offset: 0,
    });
  });
}

/**
 * Setup mocks for job assignments
 */
export async function setupJobAssignmentsApiMocks(page: Page) {
  await page.route('**/api/v1/job-assignments*', async (route) => {
    await mockApiResponse(route, {
      data: mockJobAssignments,
      total: mockJobAssignments.length,
      limit: 50,
      offset: 0,
    });
  });
}

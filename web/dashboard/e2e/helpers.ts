/**
 * Test helpers and utilities for E2E tests
 */

export const testCredentials = {
  email: 'test@example.com',
  password: 'TestPassword123!',
  name: 'Test User',
};

export const adminCredentials = {
  email: 'admin@example.com',
  password: 'AdminPassword123!',
  name: 'Admin User',
};

/**
 * Login helper - authenticates a user and navigates to dashboard
 */
export async function login(page, credentials = testCredentials) {
  await page.goto('/login');

  // Fill in login form
  await page.fill('input[type="email"]', credentials.email);
  await page.fill('input[type="password"]', credentials.password);

  // Submit form
  await page.click('button[type="submit"]');

  // Wait for navigation to dashboard
  await page.waitForURL('**/dashboard', { timeout: 5000 });
}

/**
 * Logout helper
 */
export async function logout(page) {
  await page.click('button:has-text("Logout")');
  await page.waitForURL('**/login');
}

/**
 * Mock API response helper
 */
export function mockApiResponse(route, data, status = 200) {
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

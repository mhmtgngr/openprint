/**
 * MSW (Mock Service Worker) API Handlers
 * Provides mock responses for all API endpoints used in the dashboard
 */

import { http, HttpResponse, delay } from 'msw';

// Base URL for API requests
const API_URL = import.meta.env.VITE_API_URL || '/api/v1';

// Mock data generators
export const mockUsers = {
  admin: {
    id: '1',
    email: 'admin@example.com',
    name: 'Admin User',
    role: 'admin',
    organizationId: 'org-1',
  },
  user: {
    id: '2',
    email: 'user@example.com',
    name: 'Regular User',
    role: 'user',
    organizationId: 'org-1',
  },
};

export const mockJobs = [
  {
    id: 'job-1',
    documentName: 'Quarterly Report.pdf',
    status: 'completed',
    pageCount: 15,
    colorPages: 3,
    fileSize: 2048576,
    createdAt: '2025-02-28T10:30:00Z',
    printer: { id: 'printer-1', name: 'Office HP' },
    errorMessage: null,
  },
  {
    id: 'job-2',
    documentName: 'Presentation.pptx',
    status: 'processing',
    pageCount: 24,
    colorPages: 12,
    fileSize: 5242880,
    createdAt: '2025-02-28T11:15:00Z',
    printer: { id: 'printer-2', name: 'Design Canon' },
    errorMessage: null,
  },
  {
    id: 'job-3',
    documentName: 'Invoice_001.pdf',
    status: 'queued',
    pageCount: 2,
    colorPages: 0,
    fileSize: 102400,
    createdAt: '2025-02-28T11:45:00Z',
    printer: null,
    errorMessage: null,
  },
  {
    id: 'job-4',
    documentName: 'Failed_Print.pdf',
    status: 'failed',
    pageCount: 5,
    colorPages: 0,
    fileSize: 512000,
    createdAt: '2025-02-28T09:00:00Z',
    printer: { id: 'printer-1', name: 'Office HP' },
    errorMessage: 'Printer out of paper',
  },
  {
    id: 'job-5',
    documentName: 'Cancelled_Doc.pdf',
    status: 'cancelled',
    pageCount: 10,
    colorPages: 5,
    fileSize: 1048576,
    createdAt: '2025-02-28T08:30:00Z',
    printer: null,
    errorMessage: null,
  },
];

export const mockAgents = [
  {
    id: 'agent-1',
    name: 'Office Agent',
    platform: 'linux',
    agentVersion: '1.2.0',
    status: 'online',
    createdAt: '2025-02-01T10:00:00Z',
    lastSeen: '2025-02-28T12:00:00Z',
    printerCount: 3,
  },
  {
    id: 'agent-2',
    name: 'Warehouse Agent',
    platform: 'windows',
    agentVersion: '1.1.5',
    status: 'offline',
    createdAt: '2025-02-10T14:00:00Z',
    lastSeen: '2025-02-28T08:00:00Z',
    printerCount: 2,
  },
  {
    id: 'agent-3',
    name: 'Reception Agent',
    platform: 'darwin',
    agentVersion: '1.2.0',
    status: 'error',
    createdAt: '2025-02-15T09:00:00Z',
    lastSeen: '2025-02-28T11:30:00Z',
    printerCount: 1,
  },
];

export const mockPrinters = [
  {
    id: 'printer-1',
    name: 'Office HP',
    agentName: 'Office Agent',
    agentId: 'agent-1',
    isOnline: true,
    isActive: true,
    capabilities: { color: true, duplex: true, paperSizes: ['A4', 'Letter'] },
    createdAt: '2025-02-01T10:00:00Z',
    lastSeen: '2025-02-28T12:00:00Z',
  },
  {
    id: 'printer-2',
    name: 'Design Canon',
    agentName: 'Office Agent',
    agentId: 'agent-1',
    isOnline: true,
    isActive: true,
    capabilities: { color: true, duplex: true, paperSizes: ['A4', 'A3', 'Letter'] },
    createdAt: '2025-02-01T10:00:00Z',
    lastSeen: '2025-02-28T12:00:00Z',
  },
  {
    id: 'printer-3',
    name: 'Warehouse Brother',
    agentName: 'Warehouse Agent',
    agentId: 'agent-2',
    isOnline: false,
    isActive: false,
    capabilities: { color: false, duplex: true, paperSizes: ['A4', 'Letter'] },
    createdAt: '2025-02-10T14:00:00Z',
    lastSeen: '2025-02-28T08:00:00Z',
  },
];

export const mockDocuments = [
  {
    id: 'doc-1',
    name: 'Quarterly Report.pdf',
    size: 2048576,
    mimeType: 'application/pdf',
    isEncrypted: true,
    createdAt: '2025-02-20T10:00:00Z',
    updatedAt: '2025-02-20T10:00:00Z',
    ownerEmail: 'admin@example.com',
  },
  {
    id: 'doc-2',
    name: 'Presentation.pptx',
    size: 5242880,
    mimeType: 'application/vnd.openxmlformats-officedocument.presentationml.presentation',
    isEncrypted: false,
    createdAt: '2025-02-25T14:30:00Z',
    updatedAt: '2025-02-25T14:30:00Z',
    ownerEmail: 'user@example.com',
  },
  {
    id: 'doc-3',
    name: 'Contract.pdf',
    size: 102400,
    mimeType: 'application/pdf',
    isEncrypted: true,
    createdAt: '2025-02-26T09:15:00Z',
    updatedAt: '2025-02-26T09:15:00Z',
    ownerEmail: 'admin@example.com',
  },
];

export const mockAnalytics = [
  {
    statDate: '2025-02-28',
    jobsCount: 45,
    jobsCompleted: 42,
    jobsFailed: 3,
    pagesPrinted: 520,
  },
  {
    statDate: '2025-02-27',
    jobsCount: 52,
    jobsCompleted: 50,
    jobsFailed: 2,
    pagesPrinted: 610,
  },
  {
    statDate: '2025-02-26',
    jobsCount: 38,
    jobsCompleted: 36,
    jobsFailed: 2,
    pagesPrinted: 445,
  },
  {
    statDate: '2025-02-25',
    jobsCount: 61,
    jobsCompleted: 58,
    jobsFailed: 3,
    pagesPrinted: 720,
  },
  {
    statDate: '2025-02-24',
    jobsCount: 44,
    jobsCompleted: 43,
    jobsFailed: 1,
    pagesPrinted: 515,
  },
  {
    statDate: '2025-02-23',
    jobsCount: 55,
    jobsCompleted: 53,
    jobsFailed: 2,
    pagesPrinted: 650,
  },
  {
    statDate: '2025-02-22',
    jobsCount: 48,
    jobsCompleted: 46,
    jobsFailed: 2,
    pagesPrinted: 570,
  },
];

// Auth handlers
export const authHandlers = [
  // Login
  http.post(`${API_URL}/auth/login`, async ({ request }) => {
    await delay(300);
    const body = await request.json() as { email: string; password: string };

    if (body.email === 'admin@example.com' && body.password === 'password123') {
      return HttpResponse.json({
        user: mockUsers.admin,
        tokens: {
          accessToken: 'mock-access-token-admin',
          refreshToken: 'mock-refresh-token-admin',
        },
      });
    }

    if (body.email === 'user@example.com' && body.password === 'password123') {
      return HttpResponse.json({
        user: mockUsers.user,
        tokens: {
          accessToken: 'mock-access-token-user',
          refreshToken: 'mock-refresh-token-user',
        },
      });
    }

    return HttpResponse.json(
      { error: 'Invalid credentials' },
      { status: 401 }
    );
  }),

  // Register
  http.post(`${API_URL}/auth/register`, async ({ request }) => {
    await delay(300);
    const body = await request.json() as { email: string; password: string; name: string };

    if (body.password.length < 8) {
      return HttpResponse.json(
        { error: 'Password must be at least 8 characters' },
        { status: 400 }
      );
    }

    return HttpResponse.json({
      user: {
        id: 'new-user-id',
        email: body.email,
        name: body.name || 'New User',
        role: 'user',
        organizationId: 'org-1',
      },
      tokens: {
        accessToken: 'mock-access-token-new',
        refreshToken: 'mock-refresh-token-new',
      },
    });
  }),

  // Logout
  http.post(`${API_URL}/auth/logout`, async () => {
    await delay(200);
    return HttpResponse.json({ success: true });
  }),

  // Refresh token
  http.post(`${API_URL}/auth/refresh`, async () => {
    await delay(200);
    return HttpResponse.json({
      accessToken: 'new-mock-access-token',
      refreshToken: 'new-mock-refresh-token',
    });
  }),

  // Get current user
  http.get(`${API_URL}/auth/me`, async ({ request }) => {
    await delay(200);
    const authHeader = request.headers.get('Authorization');

    if (!authHeader) {
      return HttpResponse.json(
        { error: 'Unauthorized' },
        { status: 401 }
      );
    }

    // Return admin user for any valid token
    return HttpResponse.json({ user: mockUsers.admin });
  }),
];

// Jobs handlers
export const jobsHandlers = [
  // List jobs
  http.get(`${API_URL}/jobs`, async ({ request }) => {
    await delay(300);
    const url = new URL(request.url);
    const status = url.searchParams.get('status');
    const limit = parseInt(url.searchParams.get('limit') || '50');

    let filteredJobs = [...mockJobs];

    if (status && status !== 'all') {
      filteredJobs = filteredJobs.filter((job) => job.status === status);
    }

    return HttpResponse.json({
      data: filteredJobs.slice(0, limit),
      total: filteredJobs.length,
    });
  }),

  // Get single job
  http.get(`${API_URL}/jobs/:id`, async ({ params }) => {
    await delay(200);
    const job = mockJobs.find((j) => j.id === params.id);

    if (!job) {
      return HttpResponse.json(
        { error: 'Job not found' },
        { status: 404 }
      );
    }

    return HttpResponse.json(job);
  }),

  // Create job
  http.post(`${API_URL}/jobs`, async ({ request }) => {
    await delay(500);
    const body = await request.json() as Record<string, unknown>;

    return HttpResponse.json({
      id: `job-${Date.now()}`,
      status: 'queued',
      createdAt: new Date().toISOString(),
      ...body,
    });
  }),

  // Cancel job
  http.post(`${API_URL}/jobs/:id/cancel`, async ({ params }) => {
    await delay(300);
    const job = mockJobs.find((j) => j.id === params.id);

    if (!job) {
      return HttpResponse.json(
        { error: 'Job not found' },
        { status: 404 }
      );
    }

    return HttpResponse.json({
      ...job,
      status: 'cancelled',
    });
  }),

  // Retry job
  http.post(`${API_URL}/jobs/:id/retry`, async ({ params }) => {
    await delay(300);
    const job = mockJobs.find((j) => j.id === params.id);

    if (!job) {
      return HttpResponse.json(
        { error: 'Job not found' },
        { status: 404 }
      );
    }

    return HttpResponse.json({
      ...job,
      status: 'queued',
      errorMessage: null,
    });
  }),

  // Get job history
  http.get(`${API_URL}/jobs/:id/history`, async ({ params }) => {
    await delay(200);

    return HttpResponse.json([
      {
        id: 'hist-1',
        jobId: params.id,
        status: 'queued',
        timestamp: '2025-02-28T11:00:00Z',
        message: 'Job queued',
      },
      {
        id: 'hist-2',
        jobId: params.id,
        status: 'processing',
        timestamp: '2025-02-28T11:01:00Z',
        message: 'Job started processing',
      },
    ]);
  }),
];

// Agents handlers
export const agentsHandlers = [
  // List agents
  http.get(`${API_URL}/agents`, async ({ request }) => {
    await delay(300);
    const url = new URL(request.url);
    const status = url.searchParams.get('status');

    let filteredAgents = [...mockAgents];

    if (status && status !== 'all') {
      filteredAgents = filteredAgents.filter((agent) => agent.status === status);
    }

    return HttpResponse.json(filteredAgents);
  }),

  // Get single agent
  http.get(`${API_URL}/agents/:id`, async ({ params }) => {
    await delay(200);
    const agent = mockAgents.find((a) => a.id === params.id);

    if (!agent) {
      return HttpResponse.json(
        { error: 'Agent not found' },
        { status: 404 }
      );
    }

    return HttpResponse.json(agent);
  }),

  // Delete agent
  http.delete(`${API_URL}/agents/:id`, async ({ params }) => {
    await delay(300);
    const agent = mockAgents.find((a) => a.id === params.id);

    if (!agent) {
      return HttpResponse.json(
        { error: 'Agent not found' },
        { status: 404 }
      );
    }

    return HttpResponse.json({ success: true });
  }),
];

// Printers handlers
export const printersHandlers = [
  // List printers
  http.get(`${API_URL}/printers`, async ({ request }) => {
    await delay(300);
    const url = new URL(request.url);
    const status = url.searchParams.get('status');

    let filteredPrinters = [...mockPrinters];

    if (status === 'online') {
      filteredPrinters = filteredPrinters.filter((p) => p.isOnline);
    } else if (status === 'offline') {
      filteredPrinters = filteredPrinters.filter((p) => !p.isOnline);
    }

    return HttpResponse.json(filteredPrinters);
  }),

  // Get single printer
  http.get(`${API_URL}/printers/:id`, async ({ params }) => {
    await delay(200);
    const printer = mockPrinters.find((p) => p.id === params.id);

    if (!printer) {
      return HttpResponse.json(
        { error: 'Printer not found' },
        { status: 404 }
      );
    }

    return HttpResponse.json(printer);
  }),

  // Toggle printer active status
  http.patch(`${API_URL}/printers/:id`, async ({ params, request }) => {
    await delay(300);
    const printer = mockPrinters.find((p) => p.id === params.id);

    if (!printer) {
      return HttpResponse.json(
        { error: 'Printer not found' },
        { status: 404 }
      );
    }

    const body = await request.json() as { isActive: boolean };

    return HttpResponse.json({
      ...printer,
      isActive: body.isActive,
    });
  }),

  // Delete printer
  http.delete(`${API_URL}/printers/:id`, async ({ params }) => {
    await delay(300);
    const printer = mockPrinters.find((p) => p.id === params.id);

    if (!printer) {
      return HttpResponse.json(
        { error: 'Printer not found' },
        { status: 404 }
      );
    }

    return HttpResponse.json({ success: true });
  }),
];

// Devices handlers (combined agents and printers)
export const devicesHandlers = [
  // List devices (for the Devices component)
  http.get(`${API_URL}/devices`, async ({ request }) => {
    await delay(300);
    const url = new URL(request.url);
    const status = url.searchParams.get('status');
    const type = url.searchParams.get('type');

    let filteredAgents = [...mockAgents];
    let filteredPrinters = [...mockPrinters];

    if (status && status !== 'all') {
      filteredAgents = filteredAgents.filter((a) => a.status === status);
      filteredPrinters = filteredPrinters.filter((p) =>
        status === 'online' ? p.isOnline : !p.isOnline
      );
    }

    if (type === 'agent') {
      filteredPrinters = [];
    } else if (type === 'printer') {
      filteredAgents = [];
    }

    return HttpResponse.json({
      agents: filteredAgents,
      printers: filteredPrinters,
      stats: {
        totalAgents: mockAgents.length,
        onlineAgents: mockAgents.filter((a) => a.status === 'online').length,
        totalPrinters: mockPrinters.length,
        onlinePrinters: mockPrinters.filter((p) => p.isOnline).length,
        offlinePrinters: mockPrinters.filter((p) => !p.isOnline).length,
      },
    });
  }),

  // Delete device
  http.delete(`${API_URL}/devices/:id`, async () => {
    await delay(300);

    return HttpResponse.json({ success: true });
  }),

  // Register device
  http.post(`${API_URL}/devices/register`, async ({ request }) => {
    await delay(500);
    const body = await request.json() as { device_name?: string; location?: string };

    return HttpResponse.json({
      id: `device-${Date.now()}`,
      name: body.device_name,
      location: body.location,
      status: 'offline',
      createdAt: new Date().toISOString(),
    });
  }),
];

// Documents handlers
export const documentsHandlers = [
  // List documents
  http.get(`${API_URL}/documents`, async ({ request }) => {
    await delay(300);
    const url = new URL(request.url);
    const userEmail = url.searchParams.get('userEmail');

    let filteredDocs = [...mockDocuments];

    if (userEmail) {
      filteredDocs = filteredDocs.filter((d) => d.ownerEmail === userEmail);
    }

    return HttpResponse.json({
      documents: filteredDocs,
      count: filteredDocs.length,
    });
  }),

  // Upload document
  http.post(`${API_URL}/documents`, async () => {
    await delay(800);

    return HttpResponse.json({
      id: `doc-${Date.now()}`,
      name: 'uploaded-file.pdf',
      size: 1048576,
      mimeType: 'application/pdf',
      isEncrypted: true,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
      ownerEmail: 'admin@example.com',
    });
  }),

  // Download document
  http.get(`${API_URL}/documents/:id`, async ({ params }) => {
    await delay(500);
    const doc = mockDocuments.find((d) => d.id === params.id);

    if (!doc) {
      return HttpResponse.json(
        { error: 'Document not found' },
        { status: 404 }
      );
    }

    // Return a mock blob
    return new HttpResponse(new ArrayBuffer(1024), {
      headers: {
        'Content-Type': doc.mimeType,
        'Content-Disposition': `attachment; filename="${doc.name}"`,
      },
    });
  }),

  // Delete document
  http.delete(`${API_URL}/documents/:id`, async ({ params }) => {
    await delay(300);
    const doc = mockDocuments.find((d) => d.id === params.id);

    if (!doc) {
      return HttpResponse.json(
        { error: 'Document not found' },
        { status: 404 }
      );
    }

    return HttpResponse.json({ success: true });
  }),
];

// Analytics handlers
export const analyticsHandlers = [
  // Get usage statistics
  http.get(`${API_URL}/analytics/usage`, async () => {
    await delay(300);

    return HttpResponse.json(mockAnalytics);
  }),

  // Get environment report
  http.get(`${API_URL}/analytics/environment`, async ({ request }) => {
    await delay(300);
    const url = new URL(request.url);
    const period = url.searchParams.get('period') || '30d';

    return HttpResponse.json({
      period,
      pagesPrinted: 4030,
      jobsCompleted: 343,
      jobsFailed: 15,
      storageUsed: 52428800,
      activeDevices: 5,
    });
  }),

  // Get quota status (print quota monitoring feature)
  http.get(`${API_URL}/analytics/quota`, async () => {
    await delay(200);

    return HttpResponse.json({
      quotaLimit: 10000,
      quotaUsed: 4030,
      quotaRemaining: 5970,
      quotaResetDate: '2025-03-31T23:59:59Z',
      period: 'monthly',
    });
  }),
];

// Combined handlers
export const handlers = [
  ...authHandlers,
  ...jobsHandlers,
  ...agentsHandlers,
  ...printersHandlers,
  ...devicesHandlers,
  ...documentsHandlers,
  ...analyticsHandlers,
];

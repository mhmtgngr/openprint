/**
 * API helper functions for E2E tests
 * Used to set up test data and make direct API calls
 */

const API_BASE = process.env.API_BASE_URL || 'http://localhost:18001';

interface AuthTokens {
  access_token: string;
  refresh_token: string;
}

let adminTokens: AuthTokens | null = null;

/**
 * Login as admin and get auth tokens
 */
export async function loginAsAdmin(): Promise<AuthTokens> {
  if (adminTokens) return adminTokens;

  const response = await fetch(`${API_BASE}/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      email: 'admin@openprint.test',
      password: 'TestAdmin123!',
    }),
  });

  if (!response.ok) {
    throw new Error(`Admin login failed: ${response.statusText}`);
  }

  adminTokens = await response.json();
  return adminTokens!;
}

/**
 * Create a test printer via API
 */
export async function createTestPrinter(printer: {
  name: string;
  type: string;
  ip: string;
  port: number;
}): Promise<void> {
  const tokens = await loginAsAdmin();

  await fetch(`${API_BASE}/printers`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${tokens.access_token}`,
    },
    body: JSON.stringify(printer),
  });
}

/**
 * Clean up test data
 */
export async function cleanupTestData(): Promise<void> {
  const tokens = await loginAsAdmin();

  // Delete test printers
  const printersResponse = await fetch(`${API_BASE}/printers`, {
    headers: {
      Authorization: `Bearer ${tokens.access_token}`,
    },
  });

  if (printersResponse.ok) {
    const printers = await printersResponse.json();
    for (const printer of (printers.data || printers)) {
      if (printer.name.includes('Test') || printer.name.startsWith('E2E')) {
        await fetch(`${API_BASE}/printers/${printer.id}`, {
          method: 'DELETE',
          headers: {
            Authorization: `Bearer ${tokens.access_token}`,
          },
        });
      }
    }
  }
}

/**
 * Reset user password for testing
 */
export async function resetTestUserPassword(
  email: string,
  newPassword: string
): Promise<void> {
  const tokens = await loginAsAdmin();

  await fetch(`${API_BASE}/users/${email}/password`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${tokens.access_token}`,
    },
    body: JSON.stringify({ password: newPassword }),
  });
}

/**
 * Create a test organization member
 */
export async function createTestUser(user: {
  email: string;
  password: string;
  name: string;
  role: string;
}): Promise<void> {
  const tokens = await loginAsAdmin();

  await fetch(`${API_BASE}/auth/register`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${tokens.access_token}`,
    },
    body: JSON.stringify(user),
  });
}

/**
 * Get auth header for API calls
 */
export async function getAuthHeader(): Promise<string> {
  const tokens = await loginAsAdmin();
  return `Bearer ${tokens.access_token}`;
}

/**
 * Compliance API helpers
 */

/**
 * Get compliance overview status
 */
export async function getComplianceOverview(): Promise<any> {
  const tokens = await loginAsAdmin();
  const response = await fetch(`${API_BASE}/compliance/overview`, {
    headers: {
      Authorization: `Bearer ${tokens.access_token}`,
    },
  });
  if (!response.ok) {
    throw new Error(`Failed to get compliance overview: ${response.statusText}`);
  }
  return await response.json();
}

/**
 * Get audit logs with optional filters
 */
export async function getAuditLogs(filters?: {
  startTime?: string;
  endTime?: string;
  userId?: string;
  eventType?: string;
  limit?: number;
  offset?: number;
}): Promise<any> {
  const tokens = await loginAsAdmin();
  const params = new URLSearchParams();
  if (filters?.startTime) params.append('start_time', filters.startTime);
  if (filters?.endTime) params.append('end_time', filters.endTime);
  if (filters?.userId) params.append('user_id', filters.userId);
  if (filters?.eventType) params.append('event_type', filters.eventType);
  if (filters?.limit) params.append('limit', filters.limit.toString());
  if (filters?.offset) params.append('offset', filters.offset.toString());

  const response = await fetch(`${API_BASE}/compliance/audit?${params.toString()}`, {
    headers: {
      Authorization: `Bearer ${tokens.access_token}`,
    },
  });
  if (!response.ok) {
    throw new Error(`Failed to get audit logs: ${response.statusText}`);
  }
  return await response.json();
}

/**
 * Create an audit event
 */
export async function createAuditEvent(event: {
  eventType: string;
  category: string;
  userId?: string;
  userName?: string;
  resourceId?: string;
  resourceType?: string;
  action: string;
  outcome: string;
  ipAddress?: string;
}): Promise<any> {
  const tokens = await loginAsAdmin();
  const response = await fetch(`${API_BASE}/compliance/audit`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${tokens.access_token}`,
    },
    body: JSON.stringify(event),
  });
  if (!response.ok) {
    throw new Error(`Failed to create audit event: ${response.statusText}`);
  }
  return await response.json();
}

/**
 * Generate compliance report
 */
export async function generateComplianceReport(options: {
  framework: string;
  periodStart: string;
  periodEnd: string;
  generatedBy?: string;
}): Promise<any> {
  const tokens = await loginAsAdmin();
  const response = await fetch(`${API_BASE}/compliance/reports/generate`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${tokens.access_token}`,
    },
    body: JSON.stringify(options),
  });
  if (!response.ok) {
    throw new Error(`Failed to generate report: ${response.statusText}`);
  }
  return await response.json();
}

/**
 * Get compliance reports
 */
export async function getComplianceReports(filters?: {
  framework?: string;
  limit?: number;
  offset?: number;
}): Promise<any> {
  const tokens = await loginAsAdmin();
  const params = new URLSearchParams();
  if (filters?.framework) params.append('framework', filters.framework);
  if (filters?.limit) params.append('limit', filters.limit.toString());
  if (filters?.offset) params.append('offset', filters.offset.toString());

  const response = await fetch(`${API_BASE}/compliance/reports?${params.toString()}`, {
    headers: {
      Authorization: `Bearer ${tokens.access_token}`,
    },
  });
  if (!response.ok) {
    throw new Error(`Failed to get reports: ${response.statusText}`);
  }
  return await response.json();
}

/**
 * Get compliance controls
 */
export async function getComplianceControls(filters?: {
  framework?: string;
  status?: string;
  limit?: number;
  offset?: number;
}): Promise<any> {
  const tokens = await loginAsAdmin();
  const params = new URLSearchParams();
  if (filters?.framework) params.append('framework', filters.framework);
  if (filters?.status) params.append('status', filters.status);
  if (filters?.limit) params.append('limit', filters.limit.toString());
  if (filters?.offset) params.append('offset', filters.offset.toString());

  const response = await fetch(`${API_BASE}/compliance/controls?${params.toString()}`, {
    headers: {
      Authorization: `Bearer ${tokens.access_token}`,
    },
  });
  if (!response.ok) {
    throw new Error(`Failed to get controls: ${response.statusText}`);
  }
  return await response.json();
}

/**
 * Create a compliance control
 */
export async function createComplianceControl(control: {
  framework: string;
  family: string;
  title: string;
  description: string;
  implementation: string;
  status: string;
  nextReview?: string;
  responsibleTeam?: string;
  riskLevel?: string;
}): Promise<any> {
  const tokens = await loginAsAdmin();
  const response = await fetch(`${API_BASE}/compliance/controls`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${tokens.access_token}`,
    },
    body: JSON.stringify(control),
  });
  if (!response.ok) {
    throw new Error(`Failed to create control: ${response.statusText}`);
  }
  return await response.json();
}

/**
 * Update control status
 */
export async function updateControlStatus(controlId: string, options: {
  status: string;
  lastAssessed?: string;
  nextReview?: string;
}): Promise<any> {
  const tokens = await loginAsAdmin();
  const response = await fetch(`${API_BASE}/compliance/controls/status/${controlId}`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${tokens.access_token}`,
    },
    body: JSON.stringify(options),
  });
  if (!response.ok) {
    throw new Error(`Failed to update control status: ${response.statusText}`);
  }
  return await response.json();
}

/**
 * Get pending reviews
 */
export async function getPendingReviews(days: number = 30): Promise<any> {
  const tokens = await loginAsAdmin();
  const response = await fetch(`${API_BASE}/compliance/reviews/pending?days=${days}`, {
    headers: {
      Authorization: `Bearer ${tokens.access_token}`,
    },
  });
  if (!response.ok) {
    throw new Error(`Failed to get pending reviews: ${response.statusText}`);
  }
  return await response.json();
}

/**
 * Record a data breach
 */
export async function recordDataBreach(breach: {
  title: string;
  description: string;
  severity: string;
  discoveredAt?: string;
  affectedRecords?: number;
  containmentStatus?: string;
}): Promise<any> {
  const tokens = await loginAsAdmin();
  const response = await fetch(`${API_BASE}/compliance/breaches`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${tokens.access_token}`,
    },
    body: JSON.stringify(breach),
  });
  if (!response.ok) {
    throw new Error(`Failed to record breach: ${response.statusText}`);
  }
  return await response.json();
}

/**
 * Clean up test compliance data
 */
export async function cleanupTestComplianceData(): Promise<void> {
  const tokens = await loginAsAdmin();

  // Delete test controls
  const controlsResponse = await fetch(`${API_BASE}/compliance/controls?limit=100`, {
    headers: {
      Authorization: `Bearer ${tokens.access_token}`,
    },
  });

  if (controlsResponse.ok) {
    const controls = await controlsResponse.json();
    const testControls = (controls.controls || []).filter((c: any) =>
      c.title?.includes('E2E') || c.title?.includes('Test') || c.description?.includes('E2E')
    );
    for (const control of testControls) {
      await fetch(`${API_BASE}/compliance/controls/${control.id}`, {
        method: 'DELETE',
        headers: {
          Authorization: `Bearer ${tokens.access_token}`,
        },
      });
    }
  }

  // Delete test reports
  const reportsResponse = await fetch(`${API_BASE}/compliance/reports?limit=100`, {
    headers: {
      Authorization: `Bearer ${tokens.access_token}`,
    },
  });

  if (reportsResponse.ok) {
    const reports = await reportsResponse.json();
    const testReports = (reports.reports || []).filter((r: any) =>
      r.name?.includes('E2E') || r.name?.includes('Test')
    );
    for (const report of testReports) {
      await fetch(`${API_BASE}/compliance/reports/${report.id}`, {
        method: 'DELETE',
        headers: {
          Authorization: `Bearer ${tokens.access_token}`,
        },
      });
    }
  }
}

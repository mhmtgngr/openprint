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
  return adminTokens;
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

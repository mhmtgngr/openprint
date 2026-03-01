import { test, expect } from '@playwright/test';
import { testCredentials, mockApiResponse, mockUsers, mockPrinters, mockJobs, mockEnvironmentReport } from '../helpers';

test.describe('Authentication - Login', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/login');
  });

  test('should display login form', async ({ page }) => {
    await expect(page.locator('h1')).toContainText('OpenPrint Cloud');
    await expect(page.locator('h2')).toContainText('Sign in to your account');

    // Check form inputs exist
    await expect(page.locator('input[type="email"]')).toBeVisible();
    await expect(page.locator('input[type="password"]')).toBeVisible();
    await expect(page.locator('button[type="submit"]')).toBeVisible();
  });

  test('should toggle between login and register forms', async ({ page }) => {
    // Initially shows login form
    await expect(page.locator('h2')).toContainText('Sign in to your account');

    // Click toggle button
    await page.click('text=Don\'t have an account? Sign up');

    // Should show register form
    await expect(page.locator('h2')).toContainText('Create a new account');
    await expect(page.locator('input#name')).toBeVisible();

    // Click toggle back
    await page.click('text=Already have an account? Sign in');

    // Should show login form again
    await expect(page.locator('h2')).toContainText('Sign in to your account');
  });

  test('should show validation errors for empty fields', async ({ page }) => {
    // Try to submit with empty fields
    await page.click('button[type="submit"]');

    // Browser validation should prevent submission
    await expect(page.locator('input[type="email"]')).toBeFocused();
  });

  test('should show error for invalid email format', async ({ page }) => {
    await page.fill('input[type="email"]', 'not-an-email');
    await page.fill('input[type="password"]', 'password123');

    // Check for HTML5 validation
    const emailInput = page.locator('input[type="email"]');
    await expect(await emailInput.evaluate((el) => (el as HTMLInputElement).checkValidity())).toBeFalsy();
  });

  test('should enforce minimum password length', async ({ page }) => {
    await page.fill('input[type="email"]', 'test@example.com');
    await page.fill('input[type="password"]', 'short');

    // Check for HTML5 validation
    const passwordInput = page.locator('input[type="password"]');
    await expect(await passwordInput.evaluate((el) => (el as HTMLInputElement).checkValidity())).toBeFalsy();
  });

  test('should show error message on failed login', async ({ page }) => {
    // Mock failed login response
    await page.route('**/api/v1/auth/login', async (route) => {
      await mockApiResponse(route, {
        code: 'INVALID_CREDENTIALS',
        message: 'Invalid email or password',
      }, 401);
    });

    await page.fill('input[type="email"]', 'wrong@example.com');
    await page.fill('input[type="password"]', 'wrongpassword');
    await page.click('button[type="submit"]');

    // Should show error message
    await expect(page.locator('text=Invalid email or password')).toBeVisible();
  });

  test('should successfully login and redirect to dashboard', async ({ page }) => {
    // Mock successful login
    await page.route('**/api/v1/auth/login', async (route) => {
      await mockApiResponse(route, {
        userId: '1',
        access_token: 'mock-access-token',
        refresh_token: 'mock-refresh-token',
        org: { id: 'org-1', name: 'Test Org' },
      });
    });

    // Mock get current user
    await page.route('**/api/v1/auth/me', async (route) => {
      await mockApiResponse(route, mockUsers[0]);
    });

    // Mock dashboard data APIs
    await page.route('**/api/v1/printers', async (route) => {
      await mockApiResponse(route, { printers: [] });
    });

    await page.route('**/api/v1/jobs*', async (route) => {
      await mockApiResponse(route, { data: [], total: 0, limit: 50, offset: 0 });
    });

    await page.route('**/api/v1/analytics/environment*', async (route) => {
      await mockApiResponse(route, { pagesPrinted: 0, co2Grams: 0, treesSaved: 0, period: '30d' });
    });

    await page.fill('input[type="email"]', testCredentials.email);
    await page.fill('input[type="password"]', testCredentials.password);
    await page.click('button[type="submit"]');

    // Should redirect to dashboard
    await page.waitForURL('**/dashboard');

    // Use more specific selector to avoid multiple h1 elements
    await expect(page.getByText('Welcome back').or(page.locator('main h1').first())).toBeVisible();
  });

  test('should successfully register and redirect to dashboard', async ({ page }) => {
    // Switch to register form
    await page.click('text=Don\'t have an account? Sign up');

    // Mock successful registration
    await page.route('**/api/v1/auth/register', async (route) => {
      await mockApiResponse(route, {
        userId: '1',
        access_token: 'mock-access-token',
        refresh_token: 'mock-refresh-token',
      });
    });

    // Mock get current user
    await page.route('**/api/v1/auth/me', async (route) => {
      await mockApiResponse(route, mockUsers[0]);
    });

    // Mock dashboard data APIs
    await page.route('**/api/v1/printers', async (route) => {
      await mockApiResponse(route, { printers: [] });
    });

    await page.route('**/api/v1/jobs*', async (route) => {
      await mockApiResponse(route, { data: [], total: 0, limit: 50, offset: 0 });
    });

    await page.route('**/api/v1/analytics/environment*', async (route) => {
      await mockApiResponse(route, { pagesPrinted: 0, co2Grams: 0, treesSaved: 0, period: '30d' });
    });

    await page.fill('input#name', 'New User');
    await page.fill('input[type="email"]', 'newuser@example.com');
    await page.fill('input[type="password"]', 'SecurePassword123!');
    await page.click('button[type="submit"]');

    // Should redirect to dashboard
    await page.waitForURL('**/dashboard');

    // Use more specific selector to avoid multiple h1 elements
    await expect(page.getByText('Welcome back').or(page.locator('main h1').first())).toBeVisible();
  });

  test('should store auth tokens after successful login', async ({ page }) => {
    // Mock successful login
    await page.route('**/api/v1/auth/login', async (route) => {
      await mockApiResponse(route, {
        userId: '1',
        access_token: 'mock-access-token',
        refresh_token: 'mock-refresh-token',
      });
    });

    // Mock get current user
    await page.route('**/api/v1/auth/me', async (route) => {
      await mockApiResponse(route, mockUsers[0]);
    });

    await page.fill('input[type="email"]', testCredentials.email);
    await page.fill('input[type="password"]', testCredentials.password);
    await page.click('button[type="submit"]');

    await page.waitForURL('**/dashboard');

    // Check localStorage for auth tokens
    const tokens = await page.evaluate(() => {
      const stored = localStorage.getItem('auth_tokens');
      return stored ? JSON.parse(stored) : null;
    });

    expect(tokens).not.toBeNull();
    expect(tokens?.accessToken).toBe('mock-access-token');
    expect(tokens?.refreshToken).toBe('mock-refresh-token');
  });

  test('should have accessible form controls', async ({ page }) => {
    // Check for proper labels
    await expect(page.locator('label[for="email"]')).toBeVisible();
    await expect(page.locator('label[for="password"]')).toBeVisible();

    // Check form inputs have required attributes
    const emailInput = page.locator('input[type="email"]');
    await expect(emailInput).toHaveAttribute('required', '');

    const passwordInput = page.locator('input[type="password"]');
    await expect(passwordInput).toHaveAttribute('required', '');
  });
});

test.describe('Authentication - Protected Routes', () => {
  test('should redirect to login when accessing protected routes unauthenticated', async ({ page }) => {
    const protectedRoutes = ['/dashboard', '/printers', '/jobs', '/documents', '/settings'];

    for (const route of protectedRoutes) {
      await page.goto(route);
      await page.waitForURL('**/login');
    }
  });

  test('should redirect to dashboard when accessing admin routes without admin role', async ({ page }) => {
    // First, login as regular user
    await page.route('**/api/v1/auth/me', async (route) => {
      await mockApiResponse(route, mockUsers[0]); // Regular user
    });

    await page.route('**/api/v1/auth/login', async (route) => {
      await mockApiResponse(route, {
        userId: mockUsers[0].id,
        access_token: 'mock-access-token',
        refresh_token: 'mock-refresh-token',
        org: { id: 'org-1', name: 'Test Org' },
      });
    });

    // Mock common dashboard APIs
    await page.route('**/api/v1/printers', async (route) => {
      await mockApiResponse(route, { printers: mockPrinters });
    });

    await page.route('**/api/v1/jobs*', async (route) => {
      await mockApiResponse(route, { data: mockJobs, total: mockJobs.length, limit: 50, offset: 0 });
    });

    await page.route('**/api/v1/analytics/environment*', async (route) => {
      await mockApiResponse(route, mockEnvironmentReport);
    });

    // Go through login flow first
    await page.goto('/login');
    await page.fill('input[type="email"]', testCredentials.email);
    await page.fill('input[type="password"]', testCredentials.password);
    await page.click('button[type="submit"]');
    await page.waitForURL('**/dashboard');

    const adminRoutes = ['/analytics', '/organization', '/quotas', '/policies', '/audit-logs'];

    for (const route of adminRoutes) {
      await page.goto(route);
      // Should redirect to dashboard (not login since we have auth)
      await page.waitForURL('**/dashboard', { timeout: 5000 });
    }
  });

  test('should allow access to admin routes with admin role', async ({ page }) => {
    // First, login as admin user
    await page.route('**/api/v1/auth/me', async (route) => {
      await mockApiResponse(route, mockUsers[1]); // Admin user
    });

    await page.route('**/api/v1/auth/login', async (route) => {
      await mockApiResponse(route, {
        userId: mockUsers[1].id,
        access_token: 'mock-access-token',
        refresh_token: 'mock-refresh-token',
        org: { id: 'org-1', name: 'Test Org' },
      });
    });

    // Mock analytics data - multiple endpoints
    await page.route('**/api/v1/analytics*', async (route) => {
      await mockApiResponse(route, {
        pagesPrinted: 1000,
        co2Grams: 200,
        treesSaved: 0.1,
        period: '30d',
        usage: [],
        auditLogs: [],
      });
    });

    // Mock common APIs
    await page.route('**/api/v1/printers', async (route) => {
      await mockApiResponse(route, { printers: mockPrinters });
    });

    await page.route('**/api/v1/jobs*', async (route) => {
      await mockApiResponse(route, { data: mockJobs, total: mockJobs.length, limit: 50, offset: 0 });
    });

    // Go through login flow first
    await page.goto('/login');
    await page.fill('input[type="email"]', 'admin@example.com');
    await page.fill('input[type="password"]', 'AdminPassword123!');
    await page.click('button[type="submit"]');
    await page.waitForURL('**/dashboard');

    // Now navigate to analytics
    await page.goto('/analytics');
    await page.waitForURL('**/analytics');

    // Check for Analytics in the page header - use text-based selector
    await expect(page.getByText('Analytics', { exact: true }).or(page.locator('h1').first())).toBeVisible();
  });
});

import { test, expect } from '@playwright/test';
import { LoginPage } from '../helpers/page-objects';
import { testUsers } from '../helpers/test-data';

test.describe('Audit Logs', () => {
  let loginPage: LoginPage;

  test.beforeEach(async ({ page }) => {
    loginPage = new LoginPage(page);

    // Mock audit logs API
    await page.route('**/api/v1/audit-logs/**', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              id: 'audit-1',
              userId: 'user-1',
              userName: 'John Doe',
              action: 'job.created',
              resourceType: 'job',
              resourceId: 'job-1',
              details: { documentName: 'Quarterly Report.pdf', pageCount: 15 },
              ipAddress: '192.168.1.100',
              userAgent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64)',
              timestamp: '2024-02-27T10:30:00Z',
            },
            {
              id: 'audit-2',
              userId: 'user-2',
              userName: 'Jane Smith',
              action: 'printer.registered',
              resourceType: 'printer',
              resourceId: 'printer-1',
              details: { printerName: 'HP LaserJet Pro', type: 'network' },
              ipAddress: '192.168.1.101',
              userAgent: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)',
              timestamp: '2024-02-27T10:15:00Z',
            },
            {
              id: 'audit-3',
              userId: 'user-1',
              userName: 'John Doe',
              action: 'user.role_changed',
              resourceType: 'user',
              resourceId: 'user-2',
              details: { oldRole: 'user', newRole: 'admin' },
              ipAddress: '192.168.1.100',
              userAgent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64)',
              timestamp: '2024-02-27T09:45:00Z',
            },
            {
              id: 'audit-4',
              userId: 'user-3',
              userName: 'Bob Johnson',
              action: 'document.uploaded',
              resourceType: 'document',
              resourceId: 'doc-1',
              details: { fileName: 'Presentation.pptx', fileSize: 5242880 },
              ipAddress: '192.168.1.150',
              userAgent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64)',
              timestamp: '2024-02-27T09:30:00Z',
            },
            {
              id: 'audit-5',
              userId: 'user-2',
              userName: 'Jane Smith',
              action: 'quota.updated',
              resourceType: 'quota',
              resourceId: 'quota-1',
              details: { userId: 'user-3', oldLimit: 500, newLimit: 1000 },
              ipAddress: '192.168.1.101',
              userAgent: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)',
              timestamp: '2024-02-27T09:00:00Z',
            },
            {
              id: 'audit-6',
              userId: 'system',
              userName: 'System',
              action: 'policy.created',
              resourceType: 'policy',
              resourceId: 'policy-1',
              details: { policyName: 'Color Printing Restriction' },
              ipAddress: 'localhost',
              userAgent: 'OpenPrint/1.0',
              timestamp: '2024-02-27T08:30:00Z',
            },
          ],
          total: 6,
          limit: 50,
          offset: 0,
        }),
      });
    });

    await loginPage.login(testUsers.admin.email, testUsers.admin.password);
  });

  test('should display audit logs page', async ({ page }) => {
    await page.goto('/audit-logs');

    await expect(page.getByRole('heading', { name: /audit logs/i })).toBeVisible();
    await expect(page.getByText('Track all administrative and user actions')).toBeVisible();
  });

  test('should display all audit log entries', async ({ page }) => {
    await page.goto('/audit-logs');

    await expect(page.getByText('John Doe')).toBeVisible();
    await expect(page.getByText('Jane Smith')).toBeVisible();
    await expect(page.getByText('Bob Johnson')).toBeVisible();
    await expect(page.getByText('System')).toBeVisible();
  });

  test('should display action types', async ({ page }) => {
    await page.goto('/audit-logs');

    await expect(page.getByText('job.created')).toBeVisible();
    await expect(page.getByText('printer.registered')).toBeVisible();
    await expect(page.getByText('user.role_changed')).toBeVisible();
    await expect(page.getByText('document.uploaded')).toBeVisible();
  });

  test('should display timestamps', async ({ page }) => {
    await page.goto('/audit-logs');

    await expect(page.getByText(/2024-02-27/)).toBeVisible();
  });

  test('should display IP addresses', async ({ page }) => {
    await page.goto('/audit-logs');

    await expect(page.getByText('192.168.1.100')).toBeVisible();
    await expect(page.getByText('192.168.1.101')).toBeVisible();
    await expect(page.getByText('192.168.1.150')).toBeVisible();
  });

  test('should display resource details', async ({ page }) => {
    await page.goto('/audit-logs');

    await expect(page.getByText('Quarterly Report.pdf')).toBeVisible();
    await expect(page.getByText('HP LaserJet Pro')).toBeVisible();
    await expect(page.getByText('Presentation.pptx')).toBeVisible();
  });

  test('should filter by action type', async ({ page }) => {
    await page.goto('/audit-logs');

    const actionFilter = page.getByLabel(/action/i);
    await actionFilter.selectOption('job.created');

    await expect(actionFilter).toHaveValue('job.created');
  });

  test('should filter by user', async ({ page }) => {
    await page.goto('/audit-logs');

    const userFilter = page.getByLabel(/user/i);
    await userFilter.selectOption('user-1');

    await expect(userFilter).toHaveValue('user-1');
  });

  test('should filter by date range', async ({ page }) => {
    await page.goto('/audit-logs');

    const startDateInput = page.getByLabel(/start date/i);
    await startDateInput.fill('2024-02-27');

    const endDateInput = page.getByLabel(/end date/i);
    await endDateInput.fill('2024-02-28');

    await expect(startDateInput).toHaveValue('2024-02-27');
    await expect(endDateInput).toHaveValue('2024-02-28');
  });

  test('should search logs', async ({ page }) => {
    await page.goto('/audit-logs');

    const searchInput = page.getByPlaceholder(/search/i);
    await searchInput.fill('John');

    await expect(page.getByText('John Doe')).toBeVisible();
  });

  test('should paginate results', async ({ page }) => {
    await page.route('**/api/v1/audit-logs/**', (route) => {
      const url = new URL(route.request().url());
      const offset = parseInt(url.searchParams.get('offset') || '0');

      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              id: `audit-${offset + 1}`,
              userId: 'user-1',
              userName: 'Test User',
              action: 'test.action',
              resourceType: 'test',
              resourceId: 'test-1',
              details: {},
              ipAddress: '192.168.1.1',
              userAgent: 'Test',
              timestamp: '2024-02-27T10:00:00Z',
            },
          ],
          total: 55,
          limit: 1,
          offset,
        }),
      });
    });

    await page.goto('/audit-logs');

    await expect(page.getByRole('button', { name: /next/i })).toBeVisible();
  });

  test('should export logs', async ({ page }) => {
    await page.goto('/audit-logs');

    await page.getByRole('button', { name: /export/i }).click();

    await expect(page.getByText(/export audit logs/i)).toBeVisible();
  });

  test('should show export format options', async ({ page }) => {
    await page.goto('/audit-logs');

    await page.getByRole('button', { name: /export/i }).click();

    await expect(page.getByText('CSV')).toBeVisible();
    await expect(page.getByText('JSON')).toBeVisible();
  });
});

test.describe('Audit Log Entry Details', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);

    await page.route('**/api/v1/audit-logs/**', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              id: 'audit-1',
              userId: 'user-1',
              userName: 'John Doe',
              userEmail: 'john@example.com',
              action: 'job.created',
              resourceType: 'job',
              resourceId: 'job-1',
              details: {
                documentName: 'Quarterly Report.pdf',
                pageCount: 15,
                colorPages: 5,
                copies: 1,
                printerName: 'HP LaserJet Pro',
              },
              ipAddress: '192.168.1.100',
              userAgent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36',
              timestamp: '2024-02-27T10:30:00Z',
            },
          ],
          total: 1,
          limit: 50,
          offset: 0,
        }),
      });
    });

    await loginPage.login(testUsers.admin.email, testUsers.admin.password);
  });

  test('should show entry details modal', async ({ page }) => {
    await page.goto('/audit-logs');

    await page.getByText('job.created').click();

    await expect(page.getByText(/audit log details/i)).toBeVisible();
  });

  test('should display full user agent', async ({ page }) => {
    await page.goto('/audit-logs');

    await page.getByText('job.created').click();

    await expect(page.getByText('Mozilla/5.0')).toBeVisible();
  });

  test('should display all details in modal', async ({ page }) => {
    await page.goto('/audit-logs');

    await page.getByText('job.created').click();

    await expect(page.getByText('documentName')).toBeVisible();
    await expect(page.getByText('pageCount')).toBeVisible();
    await expect(page.getByText('colorPages')).toBeVisible();
  });

  test('should close details modal', async ({ page }) => {
    await page.goto('/audit-logs');

    await page.getByText('job.created').click();
    await page.getByRole('button', { name: /close/i }).click();

    await expect(page.getByText(/audit log details/i)).not.toBeVisible();
  });
});

test.describe('Audit Log Filtering', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);

    await page.route('**/api/v1/audit-logs/**', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [],
          total: 0,
          limit: 50,
          offset: 0,
        }),
      });
    });

    await loginPage.login(testUsers.admin.email, testUsers.admin.password);
  });

  test('should show empty state when no logs match filters', async ({ page }) => {
    await page.goto('/audit-logs');

    await page.getByLabel(/action/i).selectOption('nonexistent');

    await expect(page.getByText(/no audit logs found/i)).toBeVisible();
  });

  test('should reset filters', async ({ page }) => {
    await page.goto('/audit-logs');

    await page.getByLabel(/action/i).selectOption('job.created');
    await page.getByRole('button', { name: /reset/i }).click();

    await expect(page.getByLabel(/action/i)).toHaveValue('');
  });
});

test.describe('Audit Log Actions', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);

    await page.route('**/api/v1/audit-logs/**', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              id: 'audit-1',
              userId: 'user-1',
              userName: 'John Doe',
              action: 'job.created',
              resourceType: 'job',
              resourceId: 'job-1',
              details: {},
              ipAddress: '192.168.1.100',
              userAgent: 'Mozilla/5.0',
              timestamp: '2024-02-27T10:30:00Z',
            },
          ],
          total: 1,
          limit: 50,
          offset: 0,
        }),
      });
    });

    await loginPage.login(testUsers.admin.email, testUsers.admin.password);
  });

  test('should filter by job actions', async ({ page }) => {
    await page.goto('/audit-logs');

    await page.getByRole('button', { name: /job actions/i }).click();

    await expect(page.getByText('job.created')).toBeVisible();
    await expect(page.getByText('job.completed')).toBeVisible();
    await expect(page.getByText('job.failed')).toBeVisible();
  });

  test('should filter by user actions', async ({ page }) => {
    await page.goto('/audit-logs');

    await page.getByRole('button', { name: /user actions/i }).click();

    await expect(page.getByText('user.created')).toBeVisible();
    await expect(page.getByText('user.role_changed')).toBeVisible();
    await expect(page.getByText('user.deleted')).toBeVisible();
  });

  test('should filter by admin actions', async ({ page }) => {
    await page.goto('/audit-logs');

    await page.getByRole('button', { name: /admin actions/i }).click();

    await expect(page.getByText('policy.created')).toBeVisible();
    await expect(page.getByText('policy.updated')).toBeVisible();
    await expect(page.getByText('policy.deleted')).toBeVisible();
  });
});

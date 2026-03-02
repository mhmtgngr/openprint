import { test, expect } from '@playwright/test';
import { LoginPage } from '../helpers/page-objects';
import { testUsers } from '../helpers/test-data';

test.describe('Print Release', () => {
  let loginPage: LoginPage;

  test.beforeEach(async ({ page }) => {
    loginPage = new LoginPage(page);

    // Mock print release API
    await page.route('**/api/v1/print-release/**', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              id: 'release-1',
              userId: 'user-1',
              userName: 'John Doe',
              documentName: 'Quarterly Report.pdf',
              pageCount: 15,
              colorPages: 5,
              submittedAt: '2024-02-27T10:30:00Z',
              status: 'pending',
              pinRequired: true,
            },
            {
              id: 'release-2',
              userId: 'user-2',
              userName: 'Jane Smith',
              documentName: 'Presentation.pptx',
              pageCount: 24,
              colorPages: 24,
              submittedAt: '2024-02-27T10:15:00Z',
              status: 'pending',
              pinRequired: false,
            },
            {
              id: 'release-3',
              userId: 'user-1',
              userName: 'John Doe',
              documentName: 'Meeting Notes.docx',
              pageCount: 3,
              colorPages: 0,
              submittedAt: '2024-02-27T09:45:00Z',
              status: 'released',
              releasedAt: '2024-02-27T09:50:00Z',
              pinRequired: false,
            },
            {
              id: 'release-4',
              userId: 'user-3',
              userName: 'Bob Johnson',
              documentName: 'Invoice.pdf',
              pageCount: 2,
              colorPages: 0,
              submittedAt: '2024-02-27T09:30:00Z',
              status: 'expired',
              pinRequired: true,
              expiresAt: '2024-02-27T10:30:00Z',
            },
          ],
          total: 4,
          limit: 50,
          offset: 0,
        }),
      });
    });

    // Mock printers API for release
    await page.route('**/api/v1/printers*', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          printers: [
            {
              id: 'printer-1',
              name: 'HP LaserJet Pro',
              type: 'network',
              isActive: true,
              isOnline: true,
            },
            {
              id: 'printer-2',
              name: 'Canon PIXMA',
              type: 'local',
              isActive: true,
              isOnline: true,
            },
          ],
        }),
      });
    });

    await loginPage.login(testUsers.admin.email, testUsers.admin.password);
  });

  test('should display print release page', async ({ page }) => {
    await page.goto('/print-release');

    await expect(page.getByRole('heading', { name: /print release/i })).toBeVisible();
    await expect(page.getByText('Release held print jobs')).toBeVisible();
  });

  test('should display all pending jobs', async ({ page }) => {
    await page.goto('/print-release');

    await expect(page.getByText('Quarterly Report.pdf')).toBeVisible();
    await expect(page.getByText('Presentation.pptx')).toBeVisible();
  });

  test('should display released jobs', async ({ page }) => {
    await page.goto('/print-release');

    await expect(page.getByText('Meeting Notes.docx')).toBeVisible();
    await expect(page.getByText('Released')).toBeVisible();
  });

  test('should display expired jobs', async ({ page }) => {
    await page.goto('/print-release');

    await expect(page.getByText('Invoice.pdf')).toBeVisible();
    await expect(page.getByText('Expired')).toBeVisible();
  });

  test('should filter by status', async ({ page }) => {
    await page.goto('/print-release');

    const pendingFilter = page.getByRole('button', { name: /pending/i });
    await pendingFilter.click();

    await expect(pendingFilter).toHaveClass(/active/);
  });

  test('should search jobs', async ({ page }) => {
    await page.goto('/print-release');

    const searchInput = page.getByPlaceholder(/search/i);
    await searchInput.fill('Quarterly');

    await expect(page.getByText('Quarterly Report.pdf')).toBeVisible();
  });

  test('should display page counts', async ({ page }) => {
    await page.goto('/print-release');

    await expect(page.getByText('15 pages')).toBeVisible();
    await expect(page.getByText('24 pages')).toBeVisible();
    await expect(page.getByText('3 pages')).toBeVisible();
  });

  test('should display color page indicators', async ({ page }) => {
    await page.goto('/print-release');

    await expect(page.getByText(/5 color/i)).toBeVisible();
    await expect(page.getByText(/24 color/i)).toBeVisible();
  });

  test('should display submit time', async ({ page }) => {
    await page.goto('/print-release');

    await expect(page.getByText(/10:30/)).toBeVisible();
    await expect(page.getByText(/10:15/)).toBeVisible();
  });
});

test.describe('Print Release Actions', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);

    await page.route('**/api/v1/print-release/**', (route) => {
      if (route.request().method() === 'POST') {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: 'release-1',
            status: 'released',
            releasedAt: '2024-02-27T10:35:00Z',
          }),
        });
      } else if (route.request().method() === 'DELETE') {
        route.fulfill({
          status: 204,
          contentType: 'application/json',
          body: '',
        });
      } else {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: [
              {
                id: 'release-1',
                userId: 'user-1',
                userName: 'John Doe',
                documentName: 'Quarterly Report.pdf',
                pageCount: 15,
                colorPages: 5,
                submittedAt: '2024-02-27T10:30:00Z',
                status: 'pending',
                pinRequired: false,
              },
            ],
            total: 1,
            limit: 50,
            offset: 0,
          }),
        });
      }
    });

    await page.route('**/api/v1/printers*', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          printers: [
            {
              id: 'printer-1',
              name: 'HP LaserJet Pro',
              type: 'network',
              isActive: true,
              isOnline: true,
            },
          ],
        }),
      });
    });

    await loginPage.login(testUsers.admin.email, testUsers.admin.password);
  });

  test('should release job without PIN', async ({ page }) => {
    await page.goto('/print-release');

    await page.getByRole('button', { name: /release/i }).first().click();

    // Should show printer selection
    await expect(page.getByText(/select printer/i)).toBeVisible();
  });

  test('should show printer selection modal', async ({ page }) => {
    await page.goto('/print-release');

    await page.getByRole('button', { name: /release/i }).first().click();

    await expect(page.getByText('HP LaserJet Pro')).toBeVisible();
    await expect(page.getByText('Canon PIXMA')).toBeVisible();
  });

  test('should complete release with printer selection', async ({ page }) => {
    await page.goto('/print-release');

    await page.getByRole('button', { name: /release/i }).first().click();
    await page.getByText('HP LaserJet Pro').click();
    await page.getByRole('button', { name: /confirm/i }).click();

    await expect(page.getByText('Released')).toBeVisible();
  });

  test('should cancel job', async ({ page }) => {
    await page.goto('/print-release');

    await page.getByRole('button', { name: /cancel/i }).first().click();

    await expect(page.getByText(/are you sure/i)).toBeVisible();
    await page.getByRole('button', { name: /confirm/i }).click();

    // Job should be removed
    await expect(page.getByText('Quarterly Report.pdf')).not.toBeVisible();
  });

  test('should show cancel confirmation dialog', async ({ page }) => {
    await page.goto('/print-release');

    await page.getByRole('button', { name: /cancel/i }).first().click();

    await expect(page.getByText(/cancel print job/i)).toBeVisible();
    await expect(page.getByRole('button', { name: /confirm/i })).toBeVisible();
    await expect(page.getByRole('button', { name: /back/i })).toBeVisible();
  });
});

test.describe('Print Release with PIN', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);

    await page.route('**/api/v1/print-release/**', (route) => {
      if (route.request().method() === 'POST') {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            id: 'release-1',
            status: 'released',
            releasedAt: '2024-02-27T10:35:00Z',
          }),
        });
      } else {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            data: [
              {
                id: 'release-1',
                userId: 'user-1',
                userName: 'John Doe',
                documentName: 'Confidential Document.pdf',
                pageCount: 5,
                colorPages: 0,
                submittedAt: '2024-02-27T10:30:00Z',
                status: 'pending',
                pinRequired: true,
              },
            ],
            total: 1,
            limit: 50,
            offset: 0,
          }),
        });
      }
    });

    await page.route('**/api/v1/printers*', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          printers: [
            {
              id: 'printer-1',
              name: 'HP LaserJet Pro',
              type: 'network',
              isActive: true,
              isOnline: true,
            },
          ],
        }),
      });
    });

    await loginPage.login(testUsers.admin.email, testUsers.admin.password);
  });

  test('should show PIN required indicator', async ({ page }) => {
    await page.goto('/print-release');

    await expect(page.getByText(/pin required/i)).toBeVisible();
  });

  test('should prompt for PIN when releasing', async ({ page }) => {
    await page.goto('/print-release');

    await page.getByRole('button', { name: /release/i }).first().click();

    await expect(page.getByLabel(/pin/i)).toBeVisible();
  });

  test('should validate PIN format', async ({ page }) => {
    await page.goto('/print-release');

    await page.getByRole('button', { name: /release/i }).first().click();

    const pinInput = page.getByLabel(/pin/i);
    await pinInput.fill('123');

    await expect(page.getByText(/pin must be/i)).toBeVisible();
  });

  test('should complete release with valid PIN', async ({ page }) => {
    await page.goto('/print-release');

    await page.getByRole('button', { name: /release/i }).first().click();

    await page.getByLabel(/pin/i).fill('1234');
    await page.getByText('HP LaserJet Pro').click();
    await page.getByRole('button', { name: /confirm/i }).click();

    await expect(page.getByText('Released')).toBeVisible();
  });
});

test.describe('Print Release Filters', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);

    await page.route('**/api/v1/print-release/**', (route) => {
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

  test('should show empty state when no jobs', async ({ page }) => {
    await page.goto('/print-release');

    await expect(page.getByText(/no pending jobs/i)).toBeVisible();
  });

  test('should filter by user', async ({ page }) => {
    await page.goto('/print-release');

    const userFilter = page.getByLabel(/user/i);
    await userFilter.selectOption('user-1');

    await expect(userFilter).toHaveValue('user-1');
  });

  test('should filter by date range', async ({ page }) => {
    await page.goto('/print-release');

    const startDateInput = page.getByLabel(/from/i);
    await startDateInput.fill('2024-02-27');

    const endDateInput = page.getByLabel(/to/i);
    await endDateInput.fill('2024-02-28');

    await expect(startDateInput).toHaveValue('2024-02-27');
  });
});

test.describe('Print Release Statistics', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);

    await page.route('**/api/v1/print-release/**', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [],
          total: 0,
          limit: 50,
          offset: 0,
          statistics: {
            pendingJobs: 12,
            releasedToday: 45,
            expiredJobs: 3,
            avgWaitTime: 15,
          },
        }),
      });
    });

    await loginPage.login(testUsers.admin.email, testUsers.admin.password);
  });

  test('should display statistics cards', async ({ page }) => {
    await page.goto('/print-release');

    await expect(page.getByText(/pending jobs/i)).toBeVisible();
    await expect(page.getByText('12')).toBeVisible();
  });

  test('should display released today count', async ({ page }) => {
    await page.goto('/print-release');

    await expect(page.getByText(/released today/i)).toBeVisible();
    await expect(page.getByText('45')).toBeVisible();
  });

  test('should display expired jobs count', async ({ page }) => {
    await page.goto('/print-release');

    await expect(page.getByText(/expired jobs/i)).toBeVisible();
    await expect(page.getByText('3')).toBeVisible();
  });
});

test.describe('Print Release Bulk Actions', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);

    await page.route('**/api/v1/print-release/**', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              id: 'release-1',
              userId: 'user-1',
              userName: 'John Doe',
              documentName: 'Document 1.pdf',
              pageCount: 5,
              submittedAt: '2024-02-27T10:30:00Z',
              status: 'pending',
              pinRequired: false,
            },
            {
              id: 'release-2',
              userId: 'user-2',
              userName: 'Jane Smith',
              documentName: 'Document 2.pdf',
              pageCount: 3,
              submittedAt: '2024-02-27T10:25:00Z',
              status: 'pending',
              pinRequired: false,
            },
          ],
          total: 2,
          limit: 50,
          offset: 0,
        }),
      });
    });

    await loginPage.login(testUsers.admin.email, testUsers.admin.password);
  });

  test('should select all jobs', async ({ page }) => {
    await page.goto('/print-release');

    await page.getByRole('checkbox', { name: /select all/i }).check();

    await expect(page.getByRole('checkbox', { name: /select all/i })).toBeChecked();
  });

  test('should show bulk actions when jobs selected', async ({ page }) => {
    await page.goto('/print-release');

    await page.getByRole('checkbox', { name: /select all/i }).check();

    await expect(page.getByRole('button', { name: /release selected/i })).toBeVisible();
    await expect(page.getByRole('button', { name: /delete selected/i })).toBeVisible();
  });

  test('should release multiple jobs', async ({ page }) => {
    await page.route('**/api/v1/print-release/bulk/**', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ released: 2 }),
      });
    });

    await page.goto('/print-release');

    await page.getByRole('checkbox', { name: /select all/i }).check();
    await page.getByRole('button', { name: /release selected/i }).click();

    await expect(page.getByText(/2 jobs released/i)).toBeVisible();
  });
});

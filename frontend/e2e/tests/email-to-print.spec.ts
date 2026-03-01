import { test, expect } from '@playwright/test';
import { LoginPage } from '../helpers/page-objects';
import { testUsers } from '../helpers/test-data';

test.describe('Email-to-Print', () => {
  let loginPage: LoginPage;

  test.beforeEach(async ({ page }) => {
    loginPage = new LoginPage(page);

    // Mock email-to-print API
    await page.route('**/api/v1/email-to-print/**', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          enabled: true,
          emailDomain: 'print.openprint.test',
          defaultSettings: {
            color: false,
            duplex: true,
            paperSize: 'A4',
            copies: 1,
          },
          allowedSenders: ['user-1', 'user-2'],
          quotas: {
            maxEmailsPerDay: 50,
            maxAttachmentsPerEmail: 10,
            maxAttachmentSizeMB: 25,
          },
          statistics: {
            todayEmailCount: 23,
            totalEmailsProcessed: 1523,
            totalAttachmentsProcessed: 4569,
          },
        }),
      });
    });

    await loginPage.login(testUsers.admin.email, testUsers.admin.password);
  });

  test('should display email-to-print page', async ({ page }) => {
    await page.goto('/email-to-print');

    await expect(page.getByRole('heading', { name: /email-to-print/i })).toBeVisible();
    await expect(page.getByText('Configure email-based printing')).toBeVisible();
  });

  test('should display enable/disable toggle', async ({ page }) => {
    await page.goto('/email-to-print');

    await expect(page.getByRole('switch', { name: /enable email-to-print/i })).toBeVisible();
  });

  test('should display email domain', async ({ page }) => {
    await page.goto('/email-to-print');

    await expect(page.getByText('print.openprint.test')).toBeVisible();
  });

  test('should display statistics cards', async ({ page }) => {
    await page.goto('/email-to-print');

    await expect(page.getByText(/emails today/i)).toBeVisible();
    await expect(page.getByText('23')).toBeVisible();
    await expect(page.getByText(/total processed/i)).toBeVisible();
    await expect(page.getByText('1,523')).toBeVisible();
  });

  test('should display default settings', async ({ page }) => {
    await page.goto('/email-to-print');

    await expect(page.getByText(/default settings/i)).toBeVisible();
    await expect(page.getByText(/color/i)).toBeVisible();
    await expect(page.getByText(/duplex/i)).toBeVisible();
    await expect(page.getByText(/paper size/i)).toBeVisible();
  });

  test('should display quota settings', async ({ page }) => {
    await page.goto('/email-to-print');

    await expect(page.getByText(/max emails per day/i)).toBeVisible();
    await expect(page.getByText('50')).toBeVisible();
    await expect(page.getByText(/max attachments/i)).toBeVisible();
    await expect(page.getByText('10')).toBeVisible();
    await expect(page.getByText(/max attachment size/i)).toBeVisible();
    await expect(page.getByText('25 MB')).toBeVisible();
  });
});

test.describe('Email-to-Print Configuration', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);

    await page.route('**/api/v1/email-to-print/**', (route) => {
      if (route.request().method() === 'PATCH') {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            enabled: true,
            emailDomain: 'print.openprint.test',
          }),
        });
      } else {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            enabled: true,
            emailDomain: 'print.openprint.test',
            defaultSettings: {
              color: false,
              duplex: true,
              paperSize: 'A4',
              copies: 1,
            },
            allowedSenders: [],
            quotas: {
              maxEmailsPerDay: 50,
              maxAttachmentsPerEmail: 10,
              maxAttachmentSizeMB: 25,
            },
            statistics: {
              todayEmailCount: 23,
              totalEmailsProcessed: 1523,
              totalAttachmentsProcessed: 4569,
            },
          }),
        });
      }
    });

    await loginPage.login(testUsers.admin.email, testUsers.admin.password);
  });

  test('should toggle email-to-print on/off', async ({ page }) => {
    await page.goto('/email-to-print');

    const toggle = page.getByRole('switch', { name: /enable email-to-print/i });
    await toggle.click();

    await expect(toggle).toBeVisible();
  });

  test('should update email domain', async ({ page }) => {
    await page.goto('/email-to-print');

    await page.getByRole('button', { name: /edit domain/i }).click();

    const domainInput = page.getByLabel(/email domain/i);
    await domainInput.clear();
    await domainInput.fill('print.example.com');

    await page.getByRole('button', { name: /save/i }).click();

    await expect(page.getByText('print.example.com')).toBeVisible();
  });

  test('should update default color setting', async ({ page }) => {
    await page.goto('/email-to-print');

    await page.getByRole('button', { name: /edit settings/i }).click();

    await page.getByLabel(/color/i).check();

    await page.getByRole('button', { name: /save/i }).click();
  });

  test('should update default duplex setting', async ({ page }) => {
    await page.goto('/email-to-print');

    await page.getByRole('button', { name: /edit settings/i }).click();

    await page.getByLabel(/duplex/i).uncheck();

    await page.getByRole('button', { name: /save/i }).click();
  });

  test('should update paper size', async ({ page }) => {
    await page.goto('/email-to-print');

    await page.getByRole('button', { name: /edit settings/i }).click();

    await page.getByLabel(/paper size/i).selectOption('Letter');

    await page.getByRole('button', { name: /save/i }).click();
  });

  test('should update max emails per day quota', async ({ page }) => {
    await page.goto('/email-to-print');

    await page.getByRole('button', { name: /edit quotas/i }).click();

    const maxEmailsInput = page.getByLabel(/max emails per day/i);
    await maxEmailsInput.clear();
    await maxEmailsInput.fill('100');

    await page.getByRole('button', { name: /save/i }).click();
  });

  test('should update max attachments per email', async ({ page }) => {
    await page.goto('/email-to-print');

    await page.getByRole('button', { name: /edit quotas/i }).click();

    const maxAttachmentsInput = page.getByLabel(/max attachments/i);
    await maxAttachmentsInput.clear();
    await maxAttachmentsInput.fill('20');

    await page.getByRole('button', { name: /save/i }).click();
  });

  test('should update max attachment size', async ({ page }) => {
    await page.goto('/email-to-print');

    await page.getByRole('button', { name: /edit quotas/i }).click();

    const maxSizeInput = page.getByLabel(/max attachment size/i);
    await maxSizeInput.clear();
    await maxSizeInput.fill('50');

    await page.getByRole('button', { name: /save/i }).click();
  });

  test('should show instructions for users', async ({ page }) => {
    await page.goto('/email-to-print');

    await expect(page.getByText(/how to use/i)).toBeVisible();
    await expect(page.getByText(/send your documents to/i)).toBeVisible();
  });
});

test.describe('Email-to-Print Allowed Senders', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);

    await page.route('**/api/v1/email-to-print/**', (route) => {
      if (route.request().method() === 'POST' && route.request().url().includes('senders')) {
        route.fulfill({
          status: 201,
          contentType: 'application/json',
          body: JSON.stringify({
            id: 'sender-1',
            email: 'test@example.com',
            addedAt: '2024-02-27T10:00:00Z',
          }),
        });
      } else if (route.request().method() === 'DELETE' && route.request().url().includes('senders')) {
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
            enabled: true,
            emailDomain: 'print.openprint.test',
            defaultSettings: {},
            allowedSenders: [
              {
                id: 'sender-1',
                email: 'john@example.com',
                addedAt: '2024-02-27T10:00:00Z',
              },
              {
                id: 'sender-2',
                email: 'jane@example.com',
                addedAt: '2024-02-26T10:00:00Z',
              },
            ],
            quotas: {},
            statistics: {},
          }),
        });
      }
    });

    await loginPage.login(testUsers.admin.email, testUsers.admin.password);
  });

  test('should display allowed senders list', async ({ page }) => {
    await page.goto('/email-to-print');

    await expect(page.getByText(/allowed senders/i)).toBeVisible();
    await expect(page.getByText('john@example.com')).toBeVisible();
    await expect(page.getByText('jane@example.com')).toBeVisible();
  });

  test('should add new allowed sender', async ({ page }) => {
    await page.goto('/email-to-print');

    await page.getByRole('button', { name: /add sender/i }).click();

    const emailInput = page.getByLabel(/email/i);
    await emailInput.fill('test@example.com');

    await page.getByRole('button', { name: /add/i }).click();

    await expect(page.getByText('test@example.com')).toBeVisible();
  });

  test('should remove allowed sender', async ({ page }) => {
    await page.goto('/email-to-print');

    await page.locator('button[title="Remove"]').first().click();

    await expect(page.getByText('john@example.com')).not.toBeVisible();
  });

  test('should validate email format', async ({ page }) => {
    await page.goto('/email-to-print');

    await page.getByRole('button', { name: /add sender/i }).click();

    const emailInput = page.getByLabel(/email/i);
    await emailInput.fill('invalid-email');

    await page.getByRole('button', { name: /add/i }).click();

    await expect(page.getByText(/invalid email/i)).toBeVisible();
  });

  test('should show empty state when no senders', async ({ page }) => {
    await page.route('**/api/v1/email-to-print/**', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          enabled: true,
          emailDomain: 'print.openprint.test',
          defaultSettings: {},
          allowedSenders: [],
          quotas: {},
          statistics: {},
        }),
      });
    });

    await page.goto('/email-to-print');

    await expect(page.getByText(/no allowed senders/i)).toBeVisible();
    await expect(page.getByText(/add your first sender/i)).toBeVisible();
  });
});

test.describe('Email-to-Print Recent Activity', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);

    await page.route('**/api/v1/email-to-print/**', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          enabled: true,
          emailDomain: 'print.openprint.test',
          defaultSettings: {},
          allowedSenders: [],
          quotas: {},
          statistics: {},
          recentActivity: [
            {
              id: 'activity-1',
              from: 'john@example.com',
              subject: 'Quarterly Report',
              attachmentCount: 1,
              status: 'completed',
              processedAt: '2024-02-27T10:30:00Z',
              jobsCreated: 1,
            },
            {
              id: 'activity-2',
              from: 'jane@example.com',
              subject: 'Presentation',
              attachmentCount: 3,
              status: 'processing',
              processedAt: '2024-02-27T10:25:00Z',
              jobsCreated: 0,
            },
            {
              id: 'activity-3',
              from: 'bob@example.com',
              subject: 'Invoice',
              attachmentCount: 1,
              status: 'failed',
              processedAt: '2024-02-27T10:20:00Z',
              jobsCreated: 0,
              errorMessage: 'File type not supported',
            },
          ],
        }),
      });
    });

    await loginPage.login(testUsers.admin.email, testUsers.admin.password);
  });

  test('should display recent activity', async ({ page }) => {
    await page.goto('/email-to-print');

    await expect(page.getByText(/recent activity/i)).toBeVisible();
    await expect(page.getByText('john@example.com')).toBeVisible();
    await expect(page.getByText('Quarterly Report')).toBeVisible();
  });

  test('should show activity status', async ({ page }) => {
    await page.goto('/email-to-print');

    await expect(page.getByText('Completed')).toBeVisible();
    await expect(page.getByText('Processing')).toBeVisible();
    await expect(page.getByText('Failed')).toBeVisible();
  });

  test('should show attachment count', async ({ page }) => {
    await page.goto('/email-to-print');

    await expect(page.getByText(/3 attachments/i)).toBeVisible();
    await expect(page.getByText(/1 attachment/i)).toBeVisible();
  });

  test('should show error message for failed emails', async ({ page }) => {
    await page.goto('/email-to-print');

    await expect(page.getByText('File type not supported')).toBeVisible();
  });
});

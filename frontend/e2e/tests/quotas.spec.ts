import { test, expect } from '@playwright/test';
import { LoginPage } from '../helpers/page-objects';
import { testUsers } from '../helpers/test-data';

test.describe('Cost Tracking & Quotas', () => {
  let loginPage: LoginPage;

  test.beforeEach(async ({ page }) => {
    loginPage = new LoginPage(page);

    // Mock quotas API
    await page.route('**/api/v1/quotas/**', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([
          {
            userId: 'user-1',
            userName: 'John Doe',
            userEmail: 'john@example.com',
            monthlyPageLimit: 1000,
            currentMonthPages: 456,
            monthlyColorPageLimit: 200,
            currentMonthColorPages: 89,
            currentMonthCost: 12.34,
            overageActions: ['warn'],
          },
          {
            userId: 'user-2',
            userName: 'Jane Smith',
            userEmail: 'jane@example.com',
            monthlyPageLimit: 500,
            currentMonthPages: 487,
            monthlyColorPageLimit: 100,
            currentMonthColorPages: 45,
            currentMonthCost: 8.76,
            overageActions: ['block'],
          },
          {
            userId: 'user-3',
            userName: 'Bob Johnson',
            userEmail: 'bob@example.com',
            monthlyPageLimit: 1500,
            currentMonthPages: 1623,
            monthlyColorPageLimit: 300,
            currentMonthColorPages: 312,
            currentMonthCost: 18.90,
            overageActions: ['charge'],
          },
          {
            userId: 'user-4',
            userName: 'Alice Williams',
            userEmail: 'alice@example.com',
            monthlyPageLimit: null,
            currentMonthPages: 234,
            monthlyColorPageLimit: null,
            currentMonthColorPages: 56,
            currentMonthCost: 5.43,
            overageActions: ['allow'],
          },
        ]),
      });
    });

    // Mock cost history API
    await page.route('**/api/v1/quotas/history*', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              period: '2024-02',
              startDate: '2024-02-01',
              endDate: '2024-02-29',
              totalPages: 2799,
              totalCost: 45.43,
            },
            {
              period: '2024-01',
              startDate: '2024-01-01',
              endDate: '2024-01-31',
              totalPages: 3124,
              totalCost: 52.18,
            },
            {
              period: '2023-12',
              startDate: '2023-12-01',
              endDate: '2023-12-31',
              totalPages: 2891,
              totalCost: 41.32,
            },
          ],
          total: 3,
          limit: 6,
          offset: 0,
        }),
      });
    });

    await loginPage.login(testUsers.admin.email, testUsers.admin.password);
  });

  test('should display quotas page', async ({ page }) => {
    await page.goto('/quotas');

    await expect(page.getByRole('heading', { name: /cost tracking.*quotas/i })).toBeVisible();
    await expect(page.getByText('Manage page quotas and track printing costs')).toBeVisible();
  });

  test('should display organization overview cards', async ({ page }) => {
    await page.goto('/quotas');

    await expect(page.getByText(/total pages/i)).toBeVisible();
    await expect(page.getByText(/total cost/i)).toBeVisible();
    await expect(page.getByText(/active users/i)).toBeVisible();
  });

  test('should display user quotas table', async ({ page }) => {
    await page.goto('/quotas');

    await expect(page.getByText('John Doe')).toBeVisible();
    await expect(page.getByText('Jane Smith')).toBeVisible();
    await expect(page.getByText('Bob Johnson')).toBeVisible();
    await expect(page.getByText('Alice Williams')).toBeVisible();
  });

  test('should display monthly limits', async ({ page }) => {
    await page.goto('/quotas');

    await expect(page.getByText('1,000')).toBeVisible();
    await expect(page.getByText('500')).toBeVisible();
    await expect(page.getByText('1,500')).toBeVisible();
    await expect(page.getByText('Unlimited')).toBeVisible();
  });

  test('should display usage progress bars', async ({ page }) => {
    await page.goto('/quotas');

    await expect(page.locator('.bg-gray-200, .dark\\:bg-gray-700').first()).toBeVisible();
  });

  test('should display cost history section', async ({ page }) => {
    await page.goto('/quotas');

    await expect(page.getByText(/cost history/i)).toBeVisible();
  });

  test('should display cost history table', async ({ page }) => {
    await page.goto('/quotas');

    await expect(page.getByText('February 2024')).toBeVisible();
    await expect(page.getByText('January 2024')).toBeVisible();
    await expect(page.getByText('December 2023')).toBeVisible();
  });

  test('should open edit quota modal', async ({ page }) => {
    await page.goto('/quotas');

    await page.getByRole('button', { name: /edit/i }).first().click();

    await expect(page.getByText(/edit quota/i)).toBeVisible();
  });

  test('should update quota limit', async ({ page }) => {
    await page.route('**/api/v1/quotas/*', (route) => {
      if (route.request().method() === 'PATCH') {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            userId: 'user-1',
            monthlyPageLimit: 1500,
          }),
        });
      }
    });

    await page.goto('/quotas');

    await page.getByRole('button', { name: /edit/i }).first().click();

    const limitInput = page.getByLabel(/monthly page limit/i);
    await limitInput.clear();
    await limitInput.fill('1500');

    await page.getByRole('button', { name: /save/i }).click();

    await expect(page.getByText(/edit quota/i)).not.toBeVisible();
  });

  test('should update overage action', async ({ page }) => {
    await page.route('**/api/v1/quotas/*', (route) => {
      if (route.request().method() === 'PATCH') {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({}),
        });
      }
    });

    await page.goto('/quotas');

    await page.getByRole('button', { name: /edit/i }).first().click();

    await page.getByLabel(/overage action/i).selectOption('block');

    await page.getByRole('button', { name: /save/i }).click();
  });

  test('should close modal on cancel', async ({ page }) => {
    await page.goto('/quotas');

    await page.getByRole('button', { name: /edit/i }).first().click();
    await page.getByRole('button', { name: /cancel/i }).click();

    await expect(page.getByText(/edit quota/i)).not.toBeVisible();
  });

  test('should display near quota warning', async ({ page }) => {
    await page.goto('/quotas');

    await expect(page.getByText('Near Limit')).toBeVisible();
  });

  test('should display over quota status', async ({ page }) => {
    await page.goto('/quotas');

    await expect(page.getByText('Over Quota')).toBeVisible();
  });

  test('should display OK status for users under limit', async ({ page }) => {
    await page.goto('/quotas');

    await expect(page.getByText('OK')).toBeVisible();
  });
});

test.describe('Quota Validation', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);

    await page.route('**/api/v1/quotas/**', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([
          {
            userId: 'user-1',
            userName: 'Test User',
            userEmail: 'test@example.com',
            monthlyPageLimit: 1000,
            currentMonthPages: 500,
            monthlyColorPageLimit: 200,
            currentMonthColorPages: 50,
            currentMonthCost: 10.00,
            overageActions: ['warn'],
          },
        ]),
      });
    });

    await loginPage.login(testUsers.admin.email, testUsers.admin.password);
  });

  test('should validate quota limit input', async ({ page }) => {
    await page.goto('/quotas');

    await page.getByRole('button', { name: /edit/i }).first().click();

    const limitInput = page.getByLabel(/monthly page limit/i);
    await limitInput.clear();
    await limitInput.fill('-100');

    const saveButton = page.getByRole('button', { name: /save/i });
    await expect(saveButton).toBeDisabled();
  });

  test('should allow unlimited quota', async ({ page }) => {
    await page.goto('/quotas');

    await page.getByRole('button', { name: /edit/i }).first().click();

    const limitInput = page.getByLabel(/monthly page limit/i);
    await limitInput.clear();
    await limitInput.fill('0');

    await expect(page.getByText(/0 for unlimited/i)).toBeVisible();
  });
});

test.describe('Quota Actions', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);

    await page.route('**/api/v1/quotas/**', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([
          {
            userId: 'user-1',
            userName: 'Test User',
            userEmail: 'test@example.com',
            monthlyPageLimit: 1000,
            currentMonthPages: 500,
            monthlyColorPageLimit: 200,
            currentMonthColorPages: 50,
            currentMonthCost: 10.00,
            overageActions: ['warn'],
          },
        ]),
      });
    });

    await loginPage.login(testUsers.admin.email, testUsers.admin.password);
  });

  test('should show block option for overage action', async ({ page }) => {
    await page.goto('/quotas');

    await page.getByRole('button', { name: /edit/i }).first().click();

    const select = page.getByLabel(/overage action/i);
    await select.selectOption('block');

    await expect(select).toHaveValue('block');
  });

  test('should show charge option for overage action', async ({ page }) => {
    await page.goto('/quotas');

    await page.getByRole('button', { name: /edit/i }).first().click();

    const select = page.getByLabel(/overage action/i);
    await select.selectOption('charge');

    await expect(select).toHaveValue('charge');
  });

  test('should show warn option for overage action', async ({ page }) => {
    await page.goto('/quotas');

    await page.getByRole('button', { name: /edit/i }).first().click();

    const select = page.getByLabel(/overage action/i);
    await select.selectOption('warn');

    await expect(select).toHaveValue('warn');
  });

  test('should show allow option for overage action', async ({ page }) => {
    await page.goto('/quotas');

    await page.getByRole('button', { name: /edit/i }).first().click();

    const select = page.getByLabel(/overage action/i);
    await select.selectOption('allow');

    await expect(select).toHaveValue('allow');
  });
});

test.describe('Quota Statistics', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);

    await page.route('**/api/v1/quotas/**', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([
          {
            userId: 'user-1',
            userName: 'User One',
            userEmail: 'user1@example.com',
            monthlyPageLimit: 1000,
            currentMonthPages: 456,
            monthlyColorPageLimit: 200,
            currentMonthColorPages: 89,
            currentMonthCost: 12.34,
            overageActions: ['warn'],
          },
          {
            userId: 'user-2',
            userName: 'User Two',
            userEmail: 'user2@example.com',
            monthlyPageLimit: 500,
            currentMonthPages: 487,
            monthlyColorPageLimit: 100,
            currentMonthColorPages: 45,
            currentMonthCost: 8.76,
            overageActions: ['block'],
          },
        ]),
      });
    });

    await loginPage.login(testUsers.admin.email, testUsers.admin.password);
  });

  test('should calculate total organization pages', async ({ page }) => {
    await page.goto('/quotas');

    await expect(page.getByText('943')).toBeVisible();
  });

  test('should calculate total organization cost', async ({ page }) => {
    await page.goto('/quotas');

    await expect(page.getByText('$21.10')).toBeVisible();
  });

  test('should display active users count', async ({ page }) => {
    await page.goto('/quotas');

    await expect(page.getByText('2')).toBeVisible();
  });
});

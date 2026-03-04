import { test, expect } from '@playwright/test';
import { LoginPage } from '../helpers/page-objects';
import { testUsers } from '../helpers/test-data';

test.describe('Print Policy Engine', () => {
  let loginPage: LoginPage;

  test.beforeEach(async ({ page }) => {
    loginPage = new LoginPage(page);

    // Mock policies API
    await page.route('**/api/v1/policies/**', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([
          {
            id: 'policy-1',
            name: 'Color Printing Restriction',
            description: 'Limit color printing to admin users only',
            priority: 1,
            isEnabled: true,
            orgId: 'org-1',
            conditions: {
              maxPagesPerJob: 100,
            },
            actions: {
              forceColor: 'grayscale',
            },
            appliesTo: 'all',
            createdAt: '2024-01-01T00:00:00Z',
            updatedAt: '2024-01-01T00:00:00Z',
          },
          {
            id: 'policy-2',
            name: 'Duplex Default',
            description: 'Enable double-sided printing by default',
            priority: 2,
            isEnabled: true,
            orgId: 'org-1',
            conditions: {},
            actions: {
              forceDuplex: true,
            },
            appliesTo: 'all',
            createdAt: '2024-01-02T00:00:00Z',
            updatedAt: '2024-01-02T00:00:00Z',
          },
          {
            id: 'policy-3',
            name: 'Large Job Approval',
            description: 'Require approval for jobs over 50 pages',
            priority: 3,
            isEnabled: false,
            orgId: 'org-1',
            conditions: {
              requireApproval: true,
              maxPagesPerJob: 50,
            },
            actions: {
              requireApproval: true,
            },
            appliesTo: 'all',
            createdAt: '2024-01-03T00:00:00Z',
            updatedAt: '2024-01-03T00:00:00Z',
          },
        ]),
      });
    });

    await loginPage.login(testUsers.admin.email, testUsers.admin.password);
  });

  test('should display policies page', async ({ page }) => {
    await page.goto('/policies');

    await expect(page.getByRole('heading', { name: /print policies/i })).toBeVisible();
    await expect(page.getByText('Configure print policies to control printing behavior')).toBeVisible();
  });

  test('should display all policies', async ({ page }) => {
    await page.goto('/policies');

    await expect(page.getByText('Color Printing Restriction')).toBeVisible();
    await expect(page.getByText('Duplex Default')).toBeVisible();
    await expect(page.getByText('Large Job Approval')).toBeVisible();
  });

  test('should display create policy button', async ({ page }) => {
    await page.goto('/policies');

    await expect(page.getByRole('button', { name: /create policy/i })).toBeVisible();
  });

  test('should open create policy modal', async ({ page }) => {
    await page.goto('/policies');

    await page.getByRole('button', { name: /create policy/i }).click();

    await expect(page.getByText(/create policy/i)).toBeVisible();
  });

  test('should display policy priority', async ({ page }) => {
    await page.goto('/policies');

    await expect(page.getByText(/priority.*1/i)).toBeVisible();
    await expect(page.getByText(/priority.*2/i)).toBeVisible();
  });

  test('should display policy enabled status', async ({ page }) => {
    await page.goto('/policies');

    await expect(page.locator('.text-green-600, .bg-green-').first()).toBeVisible();
  });

  test('should display policy disabled status', async ({ page }) => {
    await page.goto('/policies');

    await expect(page.locator('.text-gray-400, .bg-gray-').filter({ hasText: 'Disabled' }).or(page.locator('text=Disabled'))).toBeVisible();
  });

  test('should toggle policy enabled state', async ({ page }) => {
    await page.route('**/api/v1/policies/*/toggle', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: 'policy-1',
          name: 'Color Printing Restriction',
          description: 'Limit color printing to admin users only',
          priority: 1,
          isEnabled: false,
          orgId: 'org-1',
          conditions: {},
          actions: {},
          appliesTo: 'all',
          createdAt: '2024-01-01T00:00:00Z',
          updatedAt: '2024-01-01T00:00:00Z',
        }),
      });
    });

    await page.goto('/policies');

    const toggleButton = page.locator('[data-testid="policy-toggle"]').first();
    await toggleButton.click();

    // Wait for the API call to complete
    await page.waitForTimeout(500);
  });

  test('should open edit policy modal', async ({ page }) => {
    await page.goto('/policies');

    await page.locator('button[title="Edit"]').first().click();

    await expect(page.locator('[data-testid="policy-editor"]')).toBeVisible();
  });

  test('should delete policy with confirmation', async ({ page }) => {
    await page.route('**/api/v1/policies/*', (route) => {
      if (route.request().method() === 'DELETE') {
        route.fulfill({
          status: 204,
          contentType: 'application/json',
          body: '',
        });
      }
    });

    // Handle browser confirmation dialog
    page.on('dialog', dialog => dialog.accept());

    await page.goto('/policies');

    await page.locator('button[title="Delete"]').first().click();

    await expect(page.getByText('Color Printing Restriction')).not.toBeVisible();
  });

  test('should show empty state when no policies', async ({ page }) => {
    await page.route('**/api/v1/policies/**', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([]),
      });
    });

    await page.goto('/policies');

    await expect(page.locator('[data-testid="empty-state"]')).toBeVisible();
    await expect(page.getByText(/no policies configured/i)).toBeVisible();
    await expect(page.getByText(/create your first print policy/i)).toBeVisible();
  });
});

test.describe('Policy Form', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);

    await page.route('**/api/v1/policies/**', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([]),
      });
    });

    await loginPage.login(testUsers.admin.email, testUsers.admin.password);
  });

  test('should show policy name input', async ({ page }) => {
    await page.goto('/policies');
    await page.getByRole('button', { name: /create policy/i }).click();

    await expect(page.getByLabel(/policy name/i)).toBeVisible();
  });

  test('should show description textarea', async ({ page }) => {
    await page.goto('/policies');
    await page.getByRole('button', { name: /create policy/i }).click();

    await expect(page.getByLabel(/description/i)).toBeVisible();
  });

  test('should show conditions section', async ({ page }) => {
    await page.goto('/policies');
    await page.getByRole('button', { name: /create policy/i }).click();

    await expect(page.getByText(/conditions/i)).toBeVisible();
  });

  test('should show actions section', async ({ page }) => {
    await page.goto('/policies');
    await page.getByRole('button', { name: /create policy/i }).click();

    await expect(page.getByText(/actions/i)).toBeVisible();
  });

  test('should show force duplex checkbox', async ({ page }) => {
    await page.goto('/policies');
    await page.getByRole('button', { name: /create policy/i }).click();

    await expect(page.getByLabel(/force duplex/i)).toBeVisible();
  });

  test('should show force grayscale checkbox', async ({ page }) => {
    await page.goto('/policies');
    await page.getByRole('button', { name: /create policy/i }).click();

    await expect(page.getByLabel(/force grayscale/i)).toBeVisible();
  });

  test('should show require approval checkbox', async ({ page }) => {
    await page.goto('/policies');
    await page.getByRole('button', { name: /create policy/i }).click();

    await expect(page.getByLabel(/require approval/i)).toBeVisible();
  });

  test('should validate policy name is required', async ({ page }) => {
    await page.goto('/policies');
    await page.getByRole('button', { name: /create policy/i }).click();

    await page.getByRole('button', { name: /create/i, exact: true }).click();

    await expect(page.getByText(/name is required/i)).toBeVisible();
  });

  test('should create new policy', async ({ page }) => {
    await page.route('**/api/v1/policies/**', (route) => {
      if (route.request().method() === 'POST') {
        route.fulfill({
          status: 201,
          contentType: 'application/json',
          body: JSON.stringify({
            id: 'policy-new',
            name: 'Test Policy',
            description: 'Test description',
            priority: 1,
            isEnabled: true,
          }),
        });
      }
    });

    await page.goto('/policies');
    await page.getByRole('button', { name: /create policy/i }).click();

    await page.getByLabel(/policy name/i).fill('Test Policy');
    await page.getByLabel(/description/i).fill('Test description');
    await page.getByLabel(/force duplex/i).check();

    await page.getByRole('button', { name: /create/i, exact: true }).click();

    await expect(page.getByText(/create policy/i)).not.toBeVisible();
  });

  test('should close modal on cancel', async ({ page }) => {
    await page.goto('/policies');
    await page.getByRole('button', { name: /create policy/i }).click();
    await page.getByRole('button', { name: /cancel/i }).click();

    await expect(page.getByText(/create policy/i)).not.toBeVisible();
  });

  test('should show priority input', async ({ page }) => {
    await page.goto('/policies');
    await page.getByRole('button', { name: /create policy/i }).click();

    await expect(page.getByLabel(/priority/i)).toBeVisible();
  });

  test('should show user role condition', async ({ page }) => {
    await page.goto('/policies');
    await page.getByRole('button', { name: /create policy/i }).click();

    await expect(page.getByText(/user role/i)).toBeVisible();
  });

  test('should show max pages condition', async ({ page }) => {
    await page.goto('/policies');
    await page.getByRole('button', { name: /create policy/i }).click();

    await expect(page.getByText(/max pages/i)).toBeVisible();
  });
});

test.describe('Policy Conditions', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);

    await page.route('**/api/v1/policies/**', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([]),
      });
    });

    await loginPage.login(testUsers.admin.email, testUsers.admin.password);
  });

  test('should add max pages condition', async ({ page }) => {
    await page.goto('/policies');
    await page.getByRole('button', { name: /create policy/i }).click();

    const maxPagesInput = page.getByLabel(/max pages/i);
    await maxPagesInput.fill('100');

    await expect(maxPagesInput).toHaveValue('100');
  });

  test('should add user role condition', async ({ page }) => {
    await page.goto('/policies');
    await page.getByRole('button', { name: /create policy/i }).click();

    const roleSelect = page.getByLabel(/user role/i);
    await roleSelect.selectOption('user');

    await expect(roleSelect).toHaveValue('user');
  });

  test('should add color mode condition', async ({ page }) => {
    await page.goto('/policies');
    await page.getByRole('button', { name: /create policy/i }).click();

    await page.getByLabel(/color mode/i).check();

    await expect(page.getByLabel(/color mode/i)).toBeChecked();
  });
});

test.describe('Policy Actions', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);

    await page.route('**/api/v1/policies/**', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([]),
      });
    });

    await loginPage.login(testUsers.admin.email, testUsers.admin.password);
  });

  test('should check force duplex action', async ({ page }) => {
    await page.goto('/policies');
    await page.getByRole('button', { name: /create policy/i }).click();

    await page.getByLabel(/force duplex/i).check();

    await expect(page.getByLabel(/force duplex/i)).toBeChecked();
  });

  test('should check force grayscale action', async ({ page }) => {
    await page.goto('/policies');
    await page.getByRole('button', { name: /create policy/i }).click();

    await page.getByLabel(/force grayscale/i).check();

    await expect(page.getByLabel(/force grayscale/i)).toBeChecked();
  });

  test('should check require approval action', async ({ page }) => {
    await page.goto('/policies');
    await page.getByRole('button', { name: /create policy/i }).click();

    await page.getByLabel(/require approval/i).check();

    await expect(page.getByLabel(/require approval/i)).toBeChecked();
  });

  test('should uncheck actions', async ({ page }) => {
    await page.goto('/policies');
    await page.getByRole('button', { name: /create policy/i }).click();

    await page.getByLabel(/force duplex/i).check();
    await page.getByLabel(/force duplex/i).uncheck();

    await expect(page.getByLabel(/force duplex/i)).not.toBeChecked();
  });
});

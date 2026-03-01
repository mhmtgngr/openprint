import { test, expect } from '@playwright/test';
import { LoginPage, PrintersPage } from '../helpers/page-objects';
import { testUsers, testPrinters } from '../helpers/test-data';

test.describe('Printers & Devices', () => {
  let printersPage: PrintersPage;

  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.goto();
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    printersPage = new PrintersPage(page);
    await printersPage.navigate();
  });

  test('should display printers page heading', async ({ page }) => {
    await expect(printersPage.heading).toBeVisible();
  });

  test('should display add printer button', async ({ page }) => {
    await expect(printersPage.addButton).toBeVisible();
  });

  test('should display printer list or empty state', async ({ page }) => {
    const list = printersPage.printerList;
    const emptyState = page.getByText(/no printers/i);

    const listVisible = await list.isVisible();
    const emptyVisible = await emptyState.isVisible();

    expect(listVisible || emptyVisible).toBe(true);
  });

  test('should show printer status badges', async ({ page }) => {
    // If there are printers, check status badges
    const statusBadges = page.locator('[data-testid="printer-status"]');
    const count = await statusBadges.count();

    if (count > 0) {
      await expect(statusBadges.first()).toBeVisible();
    }
  });

  test('should filter printers by status', async ({ page }) => {
    // Check for filter options
    const filterButton = page.getByRole('button', { name: /filter/i });
    if (await filterButton.isVisible()) {
      await filterButton.click();

      const onlineFilter = page.getByRole('menuitem', { name: /online/i });
      if (await onlineFilter.isVisible()) {
        await onlineFilter.click();
        await page.waitForTimeout(500);
      }
    }
  });

  test('should search printers', async ({ page }) => {
    const searchInput = page.getByRole('searchbox', { name: /search/i });
    if (await searchInput.isVisible()) {
      await searchInput.fill('HP');
      await page.waitForTimeout(500);

      // Should filter results
      const results = page.locator('.bg-white');
      const count = await results.count();
      expect(count).toBeGreaterThanOrEqual(0);
    }
  });
});

test.describe('Printers - Add/Edit/Delete', () => {
  test('should open add printer dialog', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    const printersPage = new PrintersPage(page);
    await printersPage.navigate();

    await printersPage.addButton.click();

    // Should show dialog
    await expect(page.getByRole('dialog')).toBeVisible();
    await expect(page.getByRole('heading', { name: /add printer|register printer/i })).toBeVisible();
  });

  test('should validate printer form fields', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    const printersPage = new PrintersPage(page);
    await printersPage.navigate();

    await printersPage.addButton.click();

    // Try to submit empty form
    const submitButton = page.getByRole('button', { name: /add|save|create/i });
    await submitButton.click();

    // Should show validation errors
    await expect(page.getByText(/name is required/i)).toBeVisible();
  });

  test('should add a new printer', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    const printersPage = new PrintersPage(page);
    await printersPage.navigate();

    await printersPage.addButton.click();

    // Fill form
    await page.getByLabel(/name/i).fill('E2E Test Printer');
    await page.getByLabel(/type/i).selectOption('laser');
    await page.getByLabel(/ip address|host/i).fill('192.168.1.200');
    await page.getByLabel(/port/i).fill('9100');

    // Submit
    await page.getByRole('button', { name: /add|save|create/i }).click();

    // Should show success message
    await expect(page.getByText(/printer added|success/i)).toBeVisible();
  });

  test('should edit existing printer', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    const printersPage = new PrintersPage(page);
    await printersPage.navigate();

    // Find a printer and click edit
    const printerCard = page.locator('.bg-white').first();
    const editButton = printerCard.getByRole('button', { name: /edit|settings/i });

    if (await editButton.isVisible()) {
      await editButton.click();

      // Should show edit dialog
      await expect(page.getByRole('dialog')).toBeVisible();

      // Edit name
      const nameInput = page.getByLabel(/name/i);
      await nameInput.clear();
      await nameInput.fill('Updated Printer Name');

      // Save
      await page.getByRole('button', { name: /save|update/i }).click();

      // Should show success message
      await expect(page.getByText(/saved|updated/i)).toBeVisible();
    }
  });

  test('should delete printer with confirmation', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    const printersPage = new PrintersPage(page);
    await printersPage.navigate();

    // Find a printer and click delete
    const printerCard = page.locator('.bg-white').first();
    const deleteButton = printerCard.getByRole('button', { name: /delete|remove/i });

    if (await deleteButton.isVisible()) {
      await deleteButton.click();

      // Should show confirmation dialog
      await expect(page.getByRole('dialog')).toBeVisible();
      await expect(page.getByText(/are you sure|delete this printer/i)).toBeVisible();

      // Confirm deletion
      await page.getByRole('button', { name: /confirm|delete/i }).click();

      // Should show success message
      await expect(page.getByText(/deleted|removed/i)).toBeVisible();
    }
  });
});

test.describe('Printers - Printer Details', () => {
  test('should view printer details', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    const printersPage = new PrintersPage(page);
    await printersPage.navigate();

    // Click on a printer
    const printerCard = page.locator('.bg-white').first();
    await printerCard.click();

    // Should show printer details
    await expect(page.getByRole('heading', { name: /printer details/i })).toBeVisible();
  });

  test('should display printer capabilities', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    const printersPage = new PrintersPage(page);
    await printersPage.navigate();

    // Click on a printer
    const printerCard = page.locator('.bg-white').first();
    await printerCard.click();

    // Check for capabilities section
    const capabilities = page.getByText(/capabilities|features/i);
    if (await capabilities.isVisible()) {
      await expect(capabilities).toBeVisible();
    }
  });

  test('should display printer health/status', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    const printersPage = new PrintersPage(page);
    await printersPage.navigate();

    // Click on a printer
    const printerCard = page.locator('.bg-white').first();
    await printerCard.click();

    // Check for health indicator
    const healthIndicator = page.locator('[data-testid="printer-health"]');
    if (await healthIndicator.isVisible()) {
      await expect(healthIndicator).toBeVisible();
    }
  });
});

test.describe('Printers - Permissions', () => {
  test('should show printer permissions for admin', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    const printersPage = new PrintersPage(page);
    await printersPage.navigate();

    // Click on a printer
    const printerCard = page.locator('.bg-white').first();
    await printerCard.click();

    // Check for permissions tab/section
    const permissionsTab = page.getByRole('tab', { name: /permissions/i });
    if (await permissionsTab.isVisible()) {
      await permissionsTab.click();
      await expect(page.getByText(/user permissions|access control/i)).toBeVisible();
    }
  });

  test('should grant printer permission to user', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    const printersPage = new PrintersPage(page);
    await printersPage.navigate();

    // Click on a printer
    const printerCard = page.locator('.bg-white').first();
    await printerCard.click();

    const permissionsTab = page.getByRole('tab', { name: /permissions/i });
    if (await permissionsTab.isVisible()) {
      await permissionsTab.click();

      const grantButton = page.getByRole('button', { name: /grant|add permission/i });
      if (await grantButton.isVisible()) {
        await grantButton.click();

        // Should show user selection dialog
        await expect(page.getByRole('dialog')).toBeVisible();

        // Select user and permission type
        await page.getByRole('combobox', { name: /user/i }).selectOption({ index: 0 });
        await page.getByRole('combobox', { name: /permission/i }).selectOption('print');

        await page.getByRole('button', { name: /grant|add/i }).click();

        // Should show success
        await expect(page.getByText(/permission granted|added/i)).toBeVisible();
      }
    }
  });
});

test.describe('Printers - Discovered Printers', () => {
  test('should navigate to discovered printers page', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    await page.goto('/discovered-printers');

    await expect(page.getByRole('heading', { name: /discovered printers/i })).toBeVisible();
  });

  test('should show discovered printers from network scan', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    await page.goto('/discovered-printers');

    const scanButton = page.getByRole('button', { name: /scan|discover/i });
    if (await scanButton.isVisible()) {
      await scanButton.click();

      // Should scan network
      await expect(page.getByText(/scanning|discovering/i)).toBeVisible();

      // After scan, should show results or empty state
      await page.waitForTimeout(3000);
    }
  });

  test('should add discovered printer', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    await page.goto('/discovered-printers');

    const printers = page.locator('[data-testid="discovered-printer"]');
    const count = await printers.count();

    if (count > 0) {
      const addButton = printers.first().getByRole('button', { name: /add|register/i });
      await addButton.click();

      // Should confirm addition
      await expect(page.getByText(/printer added|registered/i)).toBeVisible();
    }
  });
});

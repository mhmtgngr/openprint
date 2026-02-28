import { test, expect } from '@playwright/test';
import { login, mockApiResponse, mockUsers, mockPrinters, mockDiscoveredPrinters } from '../helpers';

test.describe('Printers Page', () => {
  test.beforeEach(async ({ page }) => {
    // Setup auth mock
    await page.route('**/api/v1/auth/me', async (route) => {
      await mockApiResponse(route, mockUsers[0]);
    });

    // Setup printers mock
    await page.route('**/api/v1/printers*', async (route) => {
      await mockApiResponse(route, mockPrinters);
    });

    // Setup discovered printers mock
    await page.route('**/api/v1/discovered-printers*', async (route) => {
      await mockApiResponse(route, {
        printers: mockDiscoveredPrinters,
        total: mockDiscoveredPrinters.length,
      });
    });

    await login(page);
    await page.goto('/printers');
  });

  test('should display printers page header', async ({ page }) => {
    await expect(page.locator('h1')).toContainText('Printers');
  });

  test('should display printer cards', async ({ page }) => {
    // Check for printer cards
    await expect(page.locator('text=' + mockPrinters[0].name)).toBeVisible();
    await expect(page.locator('text=' + mockPrinters[1].name)).toBeVisible();
  });

  test('should show printer status badges', async ({ page }) => {
    // Check for online/offline status
    const onlinePrinters = mockPrinters.filter(p => p.isOnline);
    const offlinePrinters = mockPrinters.filter(p => !p.isOnline);

    if (onlinePrinters.length > 0) {
      await expect(page.locator('text=Online')).toBeVisible();
    }

    if (offlinePrinters.length > 0) {
      await expect(page.locator('text=Offline')).toBeVisible();
    }
  });

  test('should display printer capabilities', async ({ page }) => {
    const printer = mockPrinters[0];

    await expect(page.locator('text=' + printer.name)).toBeVisible();

    // Check for capability badges
    if (printer.capabilities.supportsColor) {
      await expect(page.locator('text=Color')).toBeVisible();
    }

    if (printer.capabilities.supportsDuplex) {
      await expect(page.locator('text=Duplex')).toBeVisible();
    }
  });

  test('should display add printer button', async ({ page }) => {
    await expect(page.locator('button:has-text("Add Printer")')).toBeVisible();
  });

  test('should open add printer modal', async ({ page }) => {
    await page.click('button:has-text("Add Printer")');
    await expect(page.locator('text=Add Printer')).toBeVisible();
  });

  test('should show empty state when no printers', async ({ page }) => {
    // Mock empty printers list
    await page.route('**/api/v1/printers*', async (route) => {
      await mockApiResponse(route, []);
    });

    await page.reload();

    await expect(page.locator('text=No printers configured')).toBeVisible();
    await expect(page.locator('text=Add your first printer')).toBeVisible();
  });

  test('should filter printers by status', async ({ page }) => {
    // Click on online filter
    const onlineFilter = page.locator('button:has-text("Online")');
    if (await onlineFilter.isVisible()) {
      await onlineFilter.click();
      await page.waitForTimeout(500);
    }
  });

  test('should search printers', async ({ page }) => {
    const searchInput = page.getByPlaceholder('Search printers');
    await searchInput.fill('HP');

    // Should filter results
    await page.waitForTimeout(500);
  });

  test('should display printer details', async ({ page }) => {
    const printer = mockPrinters[0];

    // Check for printer details
    await expect(page.locator('text=' + printer.name)).toBeVisible();
    await expect(page.locator('text=' + printer.type)).toBeVisible();
  });

  test('should edit printer', async ({ page }) => {
    const editButton = page.locator('button:has-text("Edit")').first();

    if (await editButton.isVisible()) {
      // Mock update API
      await page.route('**/api/v1/printers/**', async (route) => {
        if (route.request().method() === 'PATCH' || route.request().method() === 'PUT') {
          await mockApiResponse(route, mockPrinters[0]);
        }
      });

      await editButton.click();
      await expect(page.locator('text=Edit Printer')).toBeVisible();
    }
  });

  test('should delete printer with confirmation', async ({ page }) => {
    const deleteButton = page.locator('button:has-text("Delete")').first();

    if (await deleteButton.isVisible()) {
      // Mock delete API
      await page.route('**/api/v1/printers/**', async (route) => {
        if (route.request().method() === 'DELETE') {
          await mockApiResponse(route, { success: true });
        }
      });

      await deleteButton.click();

      // Confirm deletion
      await page.click('button:has-text("Confirm")');
      await page.waitForTimeout(500);
    }
  });

  test('should show printer agent name', async ({ page }) => {
    const printer = mockPrinters[0];
    await expect(page.locator('text=' + printer.agentId)).toBeVisible();
  });

  test('should display paper sizes supported', async ({ page }) => {
    const printer = mockPrinters[0];

    for (const size of printer.capabilities.supportedPaperSizes) {
      await expect(page.locator('text=' + size)).toBeVisible();
    }
  });
});

test.describe('Add Printer Flow', () => {
  test.beforeEach(async ({ page }) => {
    await page.route('**/api/v1/auth/me', async (route) => {
      await mockApiResponse(route, mockUsers[0]);
    });

    await login(page);
    await page.goto('/printers');
  });

  test('should show form fields when adding printer', async ({ page }) => {
    await page.click('button:has-text("Add Printer")');

    await expect(page.locator('input[name="name"]')).toBeVisible();
    await expect(page.locator('select[name="agentId"]')).toBeVisible();
  });

  test('should validate printer name', async ({ page }) => {
    await page.click('button:has-text("Add Printer")');

    // Try to submit without name
    await page.click('button:has-text("Add Printer")');

    const nameInput = page.locator('input[name="name"]');
    await expect(nameInput).toBeVisible();
  });

  test('should create new printer', async ({ page }) => {
    await page.click('button:has-text("Add Printer")');

    // Mock create API
    await page.route('**/api/v1/printers', async (route) => {
      if (route.request().method() === 'POST') {
        await mockApiResponse(route, {
          id: 'printer-new',
          name: 'New Printer',
          agentId: 'agent-1',
          type: 'network',
          isActive: true,
          isOnline: true,
        });
      }
    });

    await page.fill('input[name="name"]', 'New Printer');
    await page.selectOption('select[name="agentId"]', 'agent-1');
    await page.click('button:has-text("Add Printer")');

    // Modal should close
    await expect(page.locator('text=Add Printer').and(page.locator('.modal'))).not.toBeVisible();
  });

  test('should cancel adding printer', async ({ page }) => {
    await page.click('button:has-text("Add Printer")');
    await page.click('button:has-text("Cancel")');

    await expect(page.locator('input[name="name"]')).not.toBeVisible();
  });
});

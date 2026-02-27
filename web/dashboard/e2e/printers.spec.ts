import { test, expect } from '@playwright/test';
import { login, mockApiResponse, mockUsers, mockPrinters } from './helpers';

test.describe('Printers Page', () => {
  test.beforeEach(async ({ page }) => {
    // Setup auth mock
    await page.route('**/api/v1/auth/me', async (route) => {
      await mockApiResponse(route, mockUsers[0]);
    });

    // Setup printers mock
    await page.route('**/api/v1/printers', async (route) => {
      await mockApiResponse(route, mockPrinters);
    });

    // Setup printer update mock
    await page.route('**/api/v1/printers/*', async (route) => {
      const url = route.request().url();
      if (route.request().method() === 'PATCH') {
        await mockApiResponse(route, mockPrinters[0]);
      } else {
        await mockApiResponse(route, mockPrinters[0]);
      }
    });

    await login(page);
    await page.goto('/printers');
    await page.waitForURL('**/printers');
  });

  test('should display printers page header', async ({ page }) => {
    await expect(page.locator('h1')).toContainText('Printers');
    await expect(page.locator('text=Manage your organization\'s printing devices')).toBeVisible();
  });

  test('should display stats cards', async ({ page }) => {
    await expect(page.locator('text=Total Printers')).toBeVisible();
    await expect(page.locator('text=Online')).toBeVisible();
    await expect(page.locator('text=Active')).toBeVisible();
  });

  test('should show correct printer counts', async ({ page }) => {
    const totalCount = mockPrinters.length;
    const onlineCount = mockPrinters.filter(p => p.isOnline).length;
    const activeCount = mockPrinters.filter(p => p.isActive).length;

    await expect(page.locator('text=Total Printers')).toBeVisible();
    await expect(page.locator(`text=${onlineCount}`)).toBeVisible();
    await expect(page.locator(`text=${activeCount}`)).toBeVisible();
  });

  test('should display search and filter controls', async ({ page }) => {
    await expect(page.locator('input[placeholder="Search printers..."]')).toBeVisible();
    await expect(page.locator('select')).toBeVisible();
    await expect(page.locator('text=All Printers')).toBeVisible();
    await expect(page.locator('text=Online Only')).toBeVisible();
    await expect(page.locator('text=Offline Only')).toBeVisible();
  });

  test('should filter printers by search term', async ({ page }) => {
    const searchInput = page.locator('input[placeholder="Search printers..."]');
    await searchInput.fill('HP');

    // Should show only HP printers
    await expect(page.locator('text=HP LaserJet Pro')).toBeVisible();
    await expect(page.locator('text=Canon PIXMA')).not.toBeVisible();
  });

  test('should filter printers by status', async ({ page }) => {
    const filterSelect = page.locator('select');
    await filterSelect.selectOption('online');

    // Should show only online printers
    await expect(page.locator('text=HP LaserJet Pro')).toBeVisible();
    await expect(page.locator('text=Online')).toBeVisible();
  });

  test('should display printer cards', async ({ page }) => {
    await expect(page.locator('text=HP LaserJet Pro')).toBeVisible();
    await expect(page.locator('text=Canon PIXMA')).toBeVisible();
  });

  test('should show printer details', async ({ page }) => {
    const printerCard = page.locator('text=HP LaserJet Pro').locator('..').locator('..');

    await expect(printerCard.locator('text=network')).toBeVisible();
    await expect(printerCard.locator('text=Online')).toBeVisible();
    await expect(printerCard.locator('text=Color')).toBeVisible();
    await expect(printerCard.locator('text=Duplex')).toBeVisible();
  });

  test('should show printer capabilities badges', async ({ page }) => {
    const hpPrinter = page.locator('text=HP LaserJet Pro').locator('..').locator('..');

    // Check for capability badges
    await expect(hpPrinter.locator('text=Color')).toBeVisible();
    await expect(hpPrinter.locator('text=Duplex')).toBeVisible();
    await expect(hpPrinter.locator('text=A4')).toBeVisible();
  });

  test('should show offline status for offline printers', async ({ page }) => {
    const canonPrinter = page.locator('text=Canon PIXMA').locator('..').locator('..');

    await expect(canonPrinter.locator('text=Offline')).toBeVisible();
  });

  test('should toggle printer active status', async ({ page }) => {
    const hpPrinter = page.locator('text=HP LaserJet Pro').locator('..').locator('..');

    // Click disable button
    const disableButton = hpPrinter.locator('button', { hasText: 'Disable' });
    await disableButton.click();

    // Should trigger API call
    // Button text would change after update
  });

  test('should show empty state when no printers match filter', async ({ page }) => {
    const searchInput = page.locator('input[placeholder="Search printers..."]');
    await searchInput.fill('NonExistentPrinter');

    await expect(page.locator('text=No printers found')).toBeVisible();
  });

  test('should show empty state with agent installation notice when no printers', async ({ page }) => {
    // Mock empty printers list
    await page.route('**/api/v1/printers', async (route) => {
      await mockApiResponse(route, []);
    });

    await page.reload();
    await page.waitForLoadState('networkidle');

    await expect(page.locator('text=No printers found')).toBeVisible();
    await expect(page.locator('text=Install the OpenPrint Agent')).toBeVisible();
    await expect(page.locator('text=Download for Windows')).toBeVisible();
    await expect(page.locator('text=Download for macOS')).toBeVisible();
    await expect(page.locator('text=Download for Linux')).toBeVisible();
  });

  test('should have Add Printer button', async ({ page }) => {
    const addButton = page.locator('button', { hasText: 'Add Printer' });
    await expect(addButton).toBeVisible();
  });

  test('should navigate to dashboard via sidebar', async ({ page }) => {
    await page.click('a[href="/dashboard"]');
    await page.waitForURL('**/dashboard');

    await expect(page.locator('h1')).toContainText('Welcome back');
  });

  test('should highlight Printers in navigation', async ({ page }) => {
    const printersLink = page.locator('a[href="/printers"]');
    await expect(printersLink).toHaveClass(/bg-blue-100/);
  });

  test('should handle API errors gracefully', async ({ page }) => {
    // Mock error response
    await page.route('**/api/v1/printers', async (route) => {
      await route.abort('failed');
    });

    await page.reload();
    await page.waitForLoadState('networkidle');

    // Should still show page header
    await expect(page.locator('h1')).toContainText('Printers');
  });

  test('should be responsive on mobile', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });

    // Check that printer cards stack vertically
    const printerGrid = page.locator('.grid.grid-cols-1.md\\:grid-cols-2');
    await expect(printerGrid).toBeVisible();
  });

  test('should show printer resolution', async ({ page }) => {
    const hpPrinter = page.locator('text=HP LaserJet Pro').locator('..').locator('..');

    await expect(hpPrinter.locator('text=600x600')).toBeVisible();
  });

  test('should show different paper sizes', async ({ page }) => {
    const canonPrinter = page.locator('text=Canon PIXMA').locator('..').locator('..');

    await expect(canonPrinter.locator('text=A4')).toBeVisible();
    await expect(canonPrinter.locator('text=Letter')).toBeVisible();
    await expect(canonPrinter.locator('text=A3')).toBeVisible();
  });
});

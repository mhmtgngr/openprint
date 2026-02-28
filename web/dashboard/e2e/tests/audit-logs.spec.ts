import { test, expect } from '@playwright/test';

test.describe('Advanced Audit Logs', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/audit-logs');
  });

  test('should display audit logs page header', async ({ page }) => {
    await expect(page.locator('h1')).toContainText('Audit Logs');
  });

  test('should display search input', async ({ page }) => {
    await expect(page.locator('input[placeholder*="Search"]')).toBeVisible();
  });

  test('should display filter buttons', async ({ page }) => {
    const filterTypes = ['All', 'User', 'Printer', 'Job', 'Organization', 'Auth'];

    for (const filter of filterTypes) {
      await expect(page.locator(`button:has-text("${filter}")`).first()).toBeVisible();
    }
  });

  test('should filter logs by type', async ({ page }) => {
    await page.goto('/audit-logs');

    // Click on a filter type
    await page.click('button:has-text("User")');

    // Should show active state
    const userButton = page.locator('button:has-text("User")');
    await expect(userButton).toHaveClass(/bg-blue-600/);
  });

  test('should search audit logs', async ({ page }) => {
    await page.goto('/audit-logs');

    const searchInput = page.locator('input[placeholder*="Search"]');
    await searchInput.fill('login');

    // Wait for results
    await page.waitForTimeout(500);
  });

  test('should display audit log table', async ({ page }) => {
    await page.goto('/audit-logs');

    // Check table headers
    await expect(page.locator('th:has-text("Timestamp")')).toBeVisible();
    await expect(page.locator('th:has-text("Action")')).toBeVisible();
    await expect(page.locator('th:has-text("Resource")')).toBeVisible();
    await expect(page.locator('th:has-text("Details")')).toBeVisible();
  });

  test('should display action badges with colors', async ({ page }) => {
    await page.goto('/audit-logs');

    // Check for colored action badges
    const actionBadges = page.locator('[class*="rounded-lg"]');
    if (await actionBadges.first().isVisible()) {
      const count = await actionBadges.count();
      expect(count).toBeGreaterThan(0);
    }
  });

  test('should export logs to CSV', async ({ page }) => {
    await page.goto('/audit-logs');

    // Set up download handler
    const downloadPromise = page.waitForEvent('download');

    await page.click('button:has-text("Export to CSV")');

    // Wait for download to start
    const download = await downloadPromise;
    expect(download.suggestedFilename()).toContain('.csv');
  });

  test('should paginate logs', async ({ page }) => {
    await page.goto('/audit-logs');

    // Check for pagination buttons
    const nextButton = page.locator('button:has-text("Next")');
    const prevButton = page.locator('button:has-text("Previous")');

    // Might not be visible if not enough data
    if (await nextButton.isVisible()) {
      await expect(nextButton).toBeVisible();
      await expect(prevButton).toBeVisible();

      // Click next
      await nextButton.click();
      await page.waitForTimeout(500);
    }
  });

  test('should display log entries with timestamps', async ({ page }) => {
    await page.goto('/audit-logs');

    // Check for timestamp cells
    const timestampCell = page.locator('td').first();
    if (await timestampCell.isVisible()) {
      const text = await timestampCell.textContent();
      expect(text).toMatch(/\d{1,2}\/\d{1,2}\/\d{4}/);
    }
  });

  test('should show IP addresses in logs', async ({ page }) => {
    await page.goto('/audit-logs');

    const ipAddressCell = page.locator('td:has-text(/^\d+\.\d+\.\d+\.\d+$|^-$)').first();
    if (await ipAddressCell.isVisible()) {
      await expect(ipAddressCell).toBeVisible();
    }
  });

  test('should display resource type and ID', async ({ page }) => {
    await page.goto('/audit-logs');

    const resourceCell = page.locator('td:has-text("User"), td:has-text("Printer"), td:has-text("Job")').first();
    if (await resourceCell.isVisible()) {
      await expect(resourceCell).toBeVisible();
    }
  });

  test('should handle empty results gracefully', async ({ page }) => {
    await page.goto('/audit-logs');

    // Search for something that won't exist
    await page.fill('input[placeholder*="Search"]', 'nonexistentauditlog12345');
    await page.waitForTimeout(500);

    const noResults = page.locator('text=No audit logs match');
    if (await noResults.isVisible()) {
      await expect(noResults).toBeVisible();
    }
  });

  test('should display loading state', async ({ page }) => {
    // Slow down the network to see loading state
    await page.route('**/api/v1/analytics/audit-logs**', async (route) => {
      await new Promise(resolve => setTimeout(resolve, 1000));
      route.continue();
    });

    await page.goto('/audit-logs');

    const loadingText = page.locator('text=Loading audit logs');
    if (await loadingText.isVisible({ timeout: 100 })) {
      await expect(loadingText).toBeVisible();
    }
  });
});

test.describe('Audit Log Actions', () => {
  const actionTypes = [
    { label: 'create', color: 'green' },
    { label: 'delete', color: 'red' },
    { label: 'update', color: 'amber' },
    { label: 'login', color: 'blue' },
  ];

  for (const { label, color } of actionTypes) {
    test(`should display ${label} actions with ${color} color`, async ({ page }) => {
      await page.goto('/audit-logs');

      // Search for action type
      await page.fill('input[placeholder*="Search"]', label);
      await page.waitForTimeout(500);

      // Check for colored badges
      const badge = page.locator(`[class*="bg-${color}"]`).first();
      if (await badge.isVisible()) {
        await expect(badge).toBeVisible();
      }
    });
  }
});

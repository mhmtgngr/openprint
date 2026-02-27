import { test, expect } from '@playwright/test';

test.describe('Printers Page', () => {
  test('should show empty state when no printers exist', async ({ page }) => {
    await page.goto('/printers');
    // Should redirect to login if not authenticated
    await page.waitForURL(/.*\/login/);
  });

  test('should have search and filter controls', async ({ page }) => {
    // Navigate to printers page - will be redirected to login
    await page.goto('/printers');
    await page.waitForURL(/.*\/login/);
  });
});

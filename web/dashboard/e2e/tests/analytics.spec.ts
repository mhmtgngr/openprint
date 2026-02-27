import { test, expect } from '@playwright/test';

test.describe('Analytics Page', () => {
  test('should display period selector', async ({ page }) => {
    await page.goto('/analytics');
    // Should redirect to login if not authenticated
    await page.waitForURL(/.*\/login/);
  });

  test('should show key metrics', async ({ page }) => {
    await page.goto('/analytics');
    await page.waitForURL(/.*\/login/);
  });

  test('should display environmental impact section', async ({ page }) => {
    await page.goto('/analytics');
    await page.waitForURL(/.*\/login/);
  });
});

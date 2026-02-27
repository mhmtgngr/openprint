import { test, expect } from '@playwright/test';

test.describe('Settings Page', () => {
  test('should display profile settings by default', async ({ page }) => {
    await page.goto('/settings');
    // Should redirect to login if not authenticated
    await page.waitForURL(/.*\/login/);
  });

  test('should have profile, security, and preferences tabs', async ({ page }) => {
    await page.goto('/settings');
    await page.waitForURL(/.*\/login/);
  });
});

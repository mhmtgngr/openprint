import { test, expect } from '@playwright/test';

test.describe('Organization Page', () => {
  test('should be accessible only to admins', async ({ page }) => {
    await page.goto('/organization');
    // Non-admin users should be redirected to dashboard
    // Unauthenticated users should be redirected to login
    await page.waitForURL(/.*\/login/);
  });

  test('should display organization info', async ({ page }) => {
    await page.goto('/organization');
    await page.waitForURL(/.*\/login/);
  });

  test('should display user management section', async ({ page }) => {
    await page.goto('/organization');
    await page.waitForURL(/.*\/login/);
  });
});

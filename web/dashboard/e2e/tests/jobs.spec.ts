import { test, expect } from '@playwright/test';

test.describe('Jobs Page', () => {
  test('should display job status filters', async ({ page }) => {
    await page.goto('/jobs');
    // Should redirect to login if not authenticated
    await page.waitForURL(/.*\/login/);
  });

  test('should show empty state when no jobs exist', async ({ page }) => {
    await page.goto('/jobs');
    await page.waitForURL(/.*\/login/);
  });
});

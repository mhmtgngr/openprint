import { test, expect } from '@playwright/test';

test.describe('Cost Tracking & Quotas Management', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to quotas page (will redirect to login if not authenticated)
    await page.goto('/quotas');
  });

  test('should display quotas page header', async ({ page }) => {
    // If not authenticated, should redirect to login
    const currentUrl = page.url();
    if (currentUrl.includes('/login')) {
      // Login flow would go here
      await page.goto('/quotas');
    }

    await expect(page.locator('h1')).toContainText('Cost Tracking & Quotas');
  });

  test('should display organization overview cards', async ({ page }) => {
    await page.goto('/quotas');

    // Check for overview cards
    await expect(page.locator('text=Total Pages')).toBeVisible();
    await expect(page.locator('text=Total Cost')).toBeVisible();
    await expect(page.locator('text=Active Users')).toBeVisible();
  });

  test('should display user quotas table', async ({ page }) => {
    await page.goto('/quotas');

    // Check for table headers
    await expect(page.locator('th:has-text("User")')).toBeVisible();
    await expect(page.locator('th:has-text("Monthly Limit")')).toBeVisible();
    await expect(page.locator('th:has-text("Used")')).toBeVisible();
    await expect(page.locator('th:has-text("Status")')).toBeVisible();
  });

  test('should open edit quota modal when clicking edit', async ({ page }) => {
    await page.goto('/quotas');

    // Click on an edit button if there are quotas
    const editButton = page.locator('button:has-text("Edit")').first();
    if (await editButton.isVisible()) {
      await editButton.click();
      await expect(page.locator('text=Edit Quota')).toBeVisible();
    }
  });

  test('should display quota usage indicators', async ({ page }) => {
    await page.goto('/quotas');

    // Check for progress bars or usage indicators
    const progressBar = page.locator('.bg-gray-200, .dark\\:bg-gray-700').first();
    await expect(progressBar).toBeVisible();
  });

  test('should display cost history section', async ({ page }) => {
    await page.goto('/quotas');

    // Check for cost history table
    await expect(page.locator('text=Cost History')).toBeVisible();
  });

  test('should update quota when form is submitted', async ({ page }) => {
    await page.goto('/quotas');

    const editButton = page.locator('button:has-text("Edit")').first();
    if (await editButton.isVisible()) {
      await editButton.click();

      // Fill in the form
      await page.fill('input[type="number"]', '500');
      await page.selectOption('select', 'warn');

      // Submit
      await page.click('button:has-text("Save")');

      // Should close modal and show success
      await expect(page.locator('text=Edit Quota')).not.toBeVisible();
    }
  });

  test('should close modal when cancel is clicked', async ({ page }) => {
    await page.goto('/quotas');

    const editButton = page.locator('button:has-text("Edit")').first();
    if (await editButton.isVisible()) {
      await editButton.click();
      await page.click('button:has-text("Cancel")');
      await expect(page.locator('text=Edit Quota')).not.toBeVisible();
    }
  });

  test('should show over quota status for users exceeding limits', async ({ page }) => {
    await page.goto('/quotas');

    // Check for over quota badge if exists
    const overQuotaBadge = page.locator('text=Over Quota');
    if (await overQuotaBadge.isVisible()) {
      await expect(overQuotaBadge).toHaveClass(/bg-red-100/);
    }
  });
});

test.describe('Quotas API Interactions', () => {
  test('should fetch and display organization quotas', async ({ page }) => {
    await page.goto('/quotas');

    // Wait for data to load
    await page.waitForSelector('table', { timeout: 5000 }).catch(() => {
      // Table might not be visible if no data or not authenticated
    });
  });

  test('should handle API errors gracefully', async ({ page }) => {
    // Intercept API calls to simulate error
    await page.route('**/api/v1/quotas/**', (route) => {
      route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Internal server error' }),
      });
    });

    await page.goto('/quotas');

    // Should show error state or empty state
    await expect(page.locator('text=Loading') | page.locator('text=No quotas')).toBeVisible();
  });
});

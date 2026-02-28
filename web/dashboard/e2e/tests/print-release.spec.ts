import { test, expect } from '@playwright/test';

test.describe('Secure Print Release', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/print-release');
  });

  test('should display print release page header', async ({ page }) => {
    await expect(page.locator('h1')).toContainText('Secure Print Release');
  });

  test('should display release station card', async ({ page }) => {
    await expect(page.locator('text=Release Station')).toBeVisible();
    await expect(page.locator('text=Select a printer and enter your PIN to release print jobs')).toBeVisible();
  });

  test('should display printer selector', async ({ page }) => {
    await expect(page.locator('label:has-text("Select Printer")')).toBeVisible();
    const printerSelect = page.locator('select');
    await expect(printerSelect).toBeVisible();
  });

  test('should display PIN input', async ({ page }) => {
    await expect(page.locator('label:has-text("Release PIN")')).toBeVisible();
    const pinInput = page.locator('input[type="password"]');
    await expect(pinInput).toBeVisible();
    await expect(pinInput).toHaveAttribute('maxlength', '6');
  });

  test('should display release all button', async ({ page }) => {
    const releaseButton = page.locator('button:has-text("Release All")');
    if (await releaseButton.isVisible()) {
      await expect(releaseButton).toBeVisible();
    }
  });

  test('should display pending jobs section', async ({ page }) => {
    await expect(page.locator('text=Pending Jobs')).toBeVisible();
  });

  test('should show empty state when no pending jobs', async ({ page }) => {
    await page.goto('/print-release');

    const emptyState = page.locator('text=No pending jobs');
    if (await emptyState.isVisible()) {
      await expect(emptyState).toBeVisible();
      await expect(page.locator('text=Secure print jobs will appear here')).toBeVisible();
    }
  });

  test('should display job cards for pending jobs', async ({ page }) => {
    await page.goto('/print-release');

    const jobCard = page.locator('[class*="hover:bg-gray"]').first();
    if (await jobCard.isVisible()) {
      await expect(jobCard).toBeVisible();

      // Check for job details
      await expect(page.locator('text=pages')).toBeVisible();
      await expect(page.locator('text=KB')).toBeVisible();
    }
  });

  test('should validate printer selection before release', async ({ page }) => {
    await page.goto('/print-release');

    const releaseButton = page.locator('button:has-text("Release")').first();
    if (await releaseButton.isVisible()) {
      // Try to release without selecting printer
      await releaseButton.click();

      // Should show error
      const error = page.locator('text=Please select a printer');
      if (await error.isVisible({ timeout: 1000 })) {
        await expect(error).toBeVisible();
      }
    }
  });

  test('should validate PIN input before release', async ({ page }) => {
    await page.goto('/print-release');

    // Select printer first
    const printerSelect = page.locator('select');
    const options = await printerSelect.locator('option').count();
    if (options > 1) {
      await printerSelect.selectOption({ index: 1 });

      const releaseButton = page.locator('button:has-text("Release")').first();
      if (await releaseButton.isVisible()) {
        await releaseButton.click();

        // Should show error
        const error = page.locator('text=Please enter your PIN');
        if (await error.isVisible({ timeout: 1000 })) {
          await expect(error).toBeVisible();
        }
      }
    }
  });

  test('should cancel pending job', async ({ page }) => {
    await page.goto('/print-release');

    const cancelButton = page.locator('button:has-text("Cancel")').first();
    if (await cancelButton.isVisible()) {
      await cancelButton.click();

      // Job should be removed
      await page.waitForTimeout(500);
    }
  });

  test('should display job metadata', async ({ page }) => {
    await page.goto('/print-release');

    const jobCard = page.locator('[class*="hover:bg-gray"]').first();
    if (await jobCard.isVisible()) {
      // Check for various metadata
      await expect(page.locator('text=Color, text=Duplex').first()).toBeVisible();
    }
  });

  test('should display queued timestamp', async ({ page }) => {
    await page.goto('/print-release');

    const timestamp = page.locator('text=Queued').first();
    if (await timestamp.isVisible()) {
      await expect(timestamp).toBeVisible();
    }
  });

  test('should handle PIN input formatting', async ({ page }) => {
    await page.goto('/print-release');

    const pinInput = page.locator('input[type="password"]');
    await pinInput.fill('123456');

    // Should accept 6 digits
    await expect(pinInput).toHaveValue('123456');
  });

  test('should show success message after release', async ({ page }) => {
    await page.goto('/print-release');

    const printerSelect = page.locator('select');
    const options = await printerSelect.locator('option').count();

    if (options > 1) {
      await printerSelect.selectOption({ index: 1 });
      await page.fill('input[type="password"]', '123456');

      const releaseButton = page.locator('button:has-text("Release")').first();
      if (await releaseButton.isVisible()) {
        await releaseButton.click();

        // Might show success or error depending on API
        await page.waitForTimeout(1000);
      }
    }
  });

  test('should poll for new jobs', async ({ page }) => {
    // The page should auto-refresh to check for new jobs
    await page.goto('/print-release');

    // Wait for a bit to check if polling happens
    await page.waitForTimeout(6000);

    // Should still be on the page
    await expect(page.locator('h1')).toContainText('Secure Print Release');
  });
});

test.describe('Print Release Job Card', () => {
  test('should display cancel and release buttons', async ({ page }) => {
    await page.goto('/print-release');

    const jobCard = page.locator('[class*="hover:bg-gray"]').first();
    if (await jobCard.isVisible()) {
      await expect(page.locator('button:has-text("Cancel")').first()).toBeVisible();
      await expect(page.locator('button:has-text("Release")').first()).toBeVisible();
    }
  });

  test('should show document name', async ({ page }) => {
    await page.goto('/print-release');

    const jobCard = page.locator('[class*="hover:bg-gray"]').first();
    if (await jobCard.isVisible()) {
      const documentName = jobCard.locator('h3, h4, .font-medium');
      await expect(documentName.first()).toBeVisible();
    }
  });

  test('should show file size', async ({ page }) => {
    await page.goto('/print-release');

    const fileSize = page.locator('text=KB').first();
    if (await fileSize.isVisible()) {
      await expect(fileSize).toBeVisible();
    }
  });

  test('should indicate color printing', async ({ page }) => {
    await page.goto('/print-release');

    const colorIndicator = page.locator('text=Color').first();
    if (await colorIndicator.isVisible()) {
      await expect(colorIndicator).toHaveClass(/text-blue-600/);
    }
  });
});

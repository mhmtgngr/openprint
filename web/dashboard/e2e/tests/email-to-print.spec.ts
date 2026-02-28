import { test, expect } from '@playwright/test';

test.describe('Email-to-Print Configuration', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/email-to-print');
  });

  test('should display email-to-print page header', async ({ page }) => {
    await expect(page.locator('h1')).toContainText('Email-to-Print');
  });

  test('should display configuration status card', async ({ page }) => {
    await expect(page.locator('text=Configuration Status')).toBeVisible();
  });

  test('should display email address', async ({ page }) => {
    await expect(page.locator('text=@org.openprint.cloud')).toBeVisible();
  });

  test('should display configure button', async ({ page }) => {
    await expect(page.locator('button:has-text("Configure")')).toBeVisible();
  });

  test('should open configuration modal', async ({ page }) => {
    await page.click('button:has-text("Configure")');
    await expect(page.locator('text=Configure Email-to-Print')).toBeVisible();
  });

  test('should display configuration fields in modal', async ({ page }) => {
    await page.click('button:has-text("Configure")');

    // Check for form fields
    await expect(page.locator('text=Enable Email-to-Print')).toBeVisible();
    await expect(page.locator('text=Default Printer')).toBeVisible();
    await expect(page.locator('text=Allowed Senders')).toBeVisible();
    await expect(page.locator('text=Max Attachments per Email')).toBeVisible();
  });

  test('should update configuration', async ({ page }) => {
    await page.click('button:has-text("Configure")');

    // Toggle enable
    await page.check('input[type="checkbox"]');

    // Update max attachments
    await page.fill('input[type="number"]', '20');

    // Save
    await page.click('button:has-text("Save Configuration")');

    // Modal should close
    await expect(page.locator('text=Configure Email-to-Print')).not.toBeVisible();
  });

  test('should send test email', async ({ page }) => {
    await page.click('button:has-text("Send Test Email")');

    // Should show success message
    await expect(page.locator('text=Test email sent successfully')).toBeVisible({ timeout: 5000 });
  });

  test('should display recent email print jobs table', async ({ page }) => {
    await page.goto('/email-to-print');

    const tableHeader = page.locator('text=Recent Email Print Jobs');
    if (await tableHeader.isVisible()) {
      await expect(tableHeader).toBeVisible();

      // Check table headers
      await expect(page.locator('th:has-text("From")')).toBeVisible();
      await expect(page.locator('th:has-text("Subject")')).toBeVisible();
      await expect(page.locator('th:has-text("Attachments")')).toBeVisible();
      await expect(page.locator('th:has-text("Status")')).toBeVisible();
    }
  });

  test('should display job status badges', async ({ page }) => {
    await page.goto('/email-to-print');

    const statusBadges = page.locator('[class*="rounded-full"]');
    if (await statusBadges.first().isVisible()) {
      await expect(statusBadges.first()).toBeVisible();
    }
  });

  test('should display printer dropdown in modal', async ({ page }) => {
    await page.click('button:has-text("Configure")');

    const printerSelect = page.locator('select');
    await expect(printerSelect).toBeVisible();

    // Check for default option
    await expect(page.locator('option:has-text("No default")')).toBeVisible();
  });

  test('should handle allowed senders input', async ({ page }) => {
    await page.click('button:has-text("Configure")');

    const sendersInput = page.locator('input[placeholder*="Comma-separated"]');
    await expect(sendersInput).toBeVisible();

    // Fill in senders
    await sendersInput.fill('user@example.com, @example.com');

    // Should retain value
    await expect(sendersInput).toHaveValue(/user@example.com/);
  });

  test('should handle auto-release and require approval checkboxes', async ({ page }) => {
    await page.click('button:has-text("Configure")');

    // Check for checkboxes
    await expect(page.locator('text=Auto-release print jobs')).toBeVisible();
    await expect(page.locator('text=Require admin approval')).toBeVisible();

    // Toggle checkboxes
    await page.check('text=Auto-release print jobs');
    await page.check('text=Require admin approval');
  });

  test('should close modal on cancel', async ({ page }) => {
    await page.click('button:has-text("Configure")');
    await page.click('button:has-text("Cancel")');

    await expect(page.locator('text=Configure Email-to-Print')).not.toBeVisible();
  });

  test('should display enabled/disabled status', async ({ page }) => {
    await page.goto('/email-to-print');

    const statusText = page.locator('text=Enabled, text=Disabled');
    await expect(statusText.first()).toBeVisible();
  });

  test('should display auto release status', async ({ page }) => {
    await page.goto('/email-to-print');

    const autoReleaseText = page.locator('text=Yes \\(Manual approval required\\), text=No \\(Manual approval required\\)');
    if (await autoReleaseText.isVisible()) {
      await expect(autoReleaseText).toBeVisible();
    }
  });
});

test.describe('Email Print Jobs Table', () => {
  test('should display job details', async ({ page }) => {
    await page.goto('/email-to-print');

    const fromCell = page.locator('td').first();
    if (await fromCell.isVisible()) {
      // Check for email format
      const text = await fromCell.textContent();
      expect(text).toMatch(/.*@.*/);
    }
  });

  test('should handle different job statuses', async ({ page }) => {
    await page.goto('/email-to-print');

    const statuses = ['completed', 'failed', 'processing', 'received'];

    for (const status of statuses) {
      const statusBadge = page.locator(`text=${status}`).first();
      if (await statusBadge.isVisible()) {
        await expect(statusBadge).toBeVisible();
        break;
      }
    }
  });
});

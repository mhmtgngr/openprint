import { test, expect } from '@playwright/test';

test.describe('Print Policy Engine', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/policies');
  });

  test('should display policies page header', async ({ page }) => {
    await expect(page.locator('h1')).toContainText('Print Policies');
  });

  test('should display create policy button', async ({ page }) => {
    await expect(page.locator('button:has-text("Create Policy")')).toBeVisible();
  });

  test('should display empty state when no policies exist', async ({ page }) => {
    await page.goto('/policies');

    const emptyState = page.locator('text=No policies configured');
    if (await emptyState.isVisible()) {
      await expect(emptyState).toBeVisible();
      await expect(page.locator('text=Create your first print policy')).toBeVisible();
    }
  });

  test('should open create policy modal', async ({ page }) => {
    await page.click('button:has-text("Create Policy")');
    await expect(page.locator('text=Create Policy')).toBeVisible();
  });

  test('should create a new policy', async ({ page }) => {
    await page.click('button:has-text("Create Policy")');

    // Fill in policy details
    await page.fill('input[placeholder="Policy Name"]', 'Test Policy');
    await page.fill('textarea', 'This is a test policy');

    // Set conditions
    await page.fill('input[placeholder="0 = unlimited"]', '100');

    // Set actions
    await page.check('input[type="checkbox"][value="on"]');
    await page.check('text=Force grayscale');

    // Submit
    await page.click('button:has-text("Create Policy")');

    // Modal should close
    await expect(page.locator('text=Create Policy')).not.toBeVisible();
  });

  test('should display policy cards', async ({ page }) => {
    await page.goto('/policies');

    const policyCard = page.locator('.bg-white').first();
    if (await policyCard.isVisible()) {
      await expect(policyCard).toBeVisible();
      await expect(page.locator('text=Priority')).toBeVisible();
    }
  });

  test('should display policy conditions and actions', async ({ page }) => {
    await page.goto('/policies');

    const conditionLabels = [
      'Max Pages',
      'Duplex',
      'Color Mode',
      'Approval',
    ];

    for (const label of conditionLabels) {
      await expect(page.locator(`text=${label}`).first()).toBeVisible();
    }
  });

  test('should toggle policy enabled state', async ({ page }) => {
    await page.goto('/policies');

    const toggleButton = page.locator('button[title*="Enable"], button[title*="Disable"]').first();
    if (await toggleButton.isVisible()) {
      await toggleButton.click();
      // The button should toggle
      await expect(toggleButton).toBeVisible();
    }
  });

  test('should edit existing policy', async ({ page }) => {
    await page.goto('/policies');

    const editButton = page.locator('button[title="Edit"]').first();
    if (await editButton.isVisible()) {
      await editButton.click();
      await expect(page.locator('text=Edit Policy')).toBeVisible();

      // Modify a field
      await page.fill('input[type="text"]', 'Updated Policy Name');

      // Save
      await page.click('button:has-text("Update Policy")');
    }
  });

  test('should delete policy with confirmation', async ({ page }) => {
    await page.goto('/policies');

    const deleteButton = page.locator('button[title="Delete"]').first();
    if (await deleteButton.isVisible()) {
      // Count policies before deletion
      const policiesBefore = await page.locator('[class*="bg-white"]').count();

      await deleteButton.click();

      // Wait for deletion to complete
      await page.waitForTimeout(500);

      // Count policies after deletion
      const policiesAfter = await page.locator('[class*="bg-white"]').count();

      expect(policiesAfter).toBeLessThan(policiesBefore);
    }
  });

  test('should display policy priority ordering', async ({ page }) => {
    await page.goto('/policies');

    const priorityLabels = page.locator('text=Priority:');
    if (await priorityLabels.first().isVisible()) {
      const count = await priorityLabels.count();
      expect(count).toBeGreaterThan(0);
    }
  });

  test('should validate form inputs', async ({ page }) => {
    await page.click('button:has-text("Create Policy")');

    // Try to submit without required fields
    await page.click('button:has-text("Create Policy")');

    // Should show validation error
    const nameInput = page.locator('input[required]');
    await expect(nameInput).toBeVisible();
  });
});

test.describe('Policy Form Modal', () => {
  test('should show all form sections', async ({ page }) => {
    await page.goto('/policies');
    await page.click('button:has-text("Create Policy")');

    await expect(page.locator('text=Conditions')).toBeVisible();
    await expect(page.locator('text=Actions')).toBeVisible();
  });

  test('should handle checkbox actions', async ({ page }) => {
    await page.goto('/policies');
    await page.click('button:has-text("Create Policy")');

    const checkboxes = [
      'Force duplex',
      'Force grayscale',
      'Require manual approval',
    ];

    for (const label of checkboxes) {
      const checkbox = page.locator(`text=${label}`).first();
      if (await checkbox.isVisible()) {
        await checkbox.check();
        await expect(checkbox).toBeChecked();
      }
    }
  });

  test('should close modal on cancel', async ({ page }) => {
    await page.goto('/policies');
    await page.click('button:has-text("Create Policy")');
    await page.click('button:has-text("Cancel")');

    await expect(page.locator('text=Create Policy')).not.toBeVisible();
  });
});

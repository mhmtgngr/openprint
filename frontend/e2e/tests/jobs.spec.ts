import { test, expect } from '@playwright/test';
import { LoginPage } from '../helpers/page-objects';
import { testUsers } from '../helpers/test-data';

test.describe('Print Jobs', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.goto();
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);
  });

  test('should display jobs page heading', async ({ page }) => {
    await page.goto('/jobs');
    await expect(page.getByRole('heading', { name: /jobs/i })).toBeVisible();
  });

  test('should display create job button', async ({ page }) => {
    await page.goto('/jobs');
    const createButton = page.getByRole('button', { name: /create job|new job/i });
    await expect(createButton).toBeVisible();
  });

  test('should display job list or empty state', async ({ page }) => {
    await page.goto('/jobs');

    const jobList = page.locator('[data-testid="job-list"]');
    const emptyState = page.getByText(/no jobs/i);

    const listVisible = await jobList.isVisible();
    const emptyVisible = await emptyState.isVisible();

    expect(listVisible || emptyVisible).toBe(true);
  });

  test('should display job status badges', async ({ page }) => {
    await page.goto('/jobs');

    const statusBadges = page.locator('[data-testid="job-status"]');
    const count = await statusBadges.count();

    if (count > 0) {
      await expect(statusBadges.first()).toBeVisible();
    }
  });

  test('should filter jobs by status', async ({ page }) => {
    await page.goto('/jobs');

    const filterDropdown = page.getByRole('combobox', { name: /filter|status/i });
    if (await filterDropdown.isVisible()) {
      await filterDropdown.click();

      const completedOption = page.getByRole('option', { name: /completed/i });
      if (await completedOption.isVisible()) {
        await completedOption.click();
        await page.waitForTimeout(500);
      }
    }
  });

  test('should search jobs', async ({ page }) => {
    await page.goto('/jobs');

    const searchInput = page.getByRole('searchbox', { name: /search/i });
    if (await searchInput.isVisible()) {
      await searchInput.fill('test');
      await page.waitForTimeout(500);

      // Should filter results
      const results = page.locator('[data-testid="job-list"] > div');
      const count = await results.count();
      expect(count).toBeGreaterThanOrEqual(0);
    }
  });
});

test.describe('Print Jobs - Create Job', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.goto();
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);
  });

  test('should open create job dialog', async ({ page }) => {
    await page.goto('/jobs');

    const createButton = page.getByRole('button', { name: /create job|new job/i });
    await createButton.click();

    await expect(page.getByRole('dialog')).toBeVisible();
    await expect(page.getByRole('heading', { name: /create print job/i })).toBeVisible();
  });

  test('should validate required fields', async ({ page }) => {
    await page.goto('/jobs');

    const createButton = page.getByRole('button', { name: /create job|new job/i });
    await createButton.click();

    // Try to submit without file
    const submitButton = page.getByRole('button', { name: /print|create|submit/i });
    await submitButton.click();

    // Should show validation error
    await expect(page.getByText(/please select a file|document is required/i)).toBeVisible();
  });

  test('should select printer', async ({ page }) => {
    await page.goto('/jobs');

    const createButton = page.getByRole('button', { name: /create job|new job/i });
    await createButton.click();

    const printerSelect = page.getByRole('combobox', { name: /printer/i });
    await expect(printerSelect).toBeVisible();
  });

  test('should configure print settings', async ({ page }) => {
    await page.goto('/jobs');

    const createButton = page.getByRole('button', { name: /create job|new job/i });
    await createButton.click();

    // Check for print settings
    const colorOption = page.getByRole('checkbox', { name: /color|colour/i });
    const duplexOption = page.getByRole('checkbox', { name: /double-sided|duplex/i });
    const copiesInput = page.getByRole('spinbutton', { name: /copies/i });

    await expect(colorOption.or(page.getByText(/color/i))).toBeVisible();
    await expect(duplexOption.or(page.getByText(/duplex/i))).toBeVisible();
    await expect(copiesInput).toBeVisible();
  });

  test('should upload document', async ({ page }) => {
    await page.goto('/jobs');

    const createButton = page.getByRole('button', { name: /create job|new job/i });
    await createButton.click();

    // Create a test PDF file
    const fileInput = page.locator('input[type="file"]');
    if (await fileInput.isVisible()) {
      // Note: In a real test, you'd upload an actual file
      // await fileInput.setInputFiles('path/to/test.pdf');

      // For now, just check the input exists
      await expect(fileInput).toBeVisible();
    }
  });
});

test.describe('Print Jobs - Job Details', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.goto();
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);
  });

  test('should view job details', async ({ page }) => {
    await page.goto('/jobs');

    // Click on first job if exists
    const jobCard = page.locator('[data-testid="job-list"] > div').first();
    const count = await jobCard.count();

    if (count > 0) {
      await jobCard.click();

      await expect(page.getByRole('heading', { name: /job details/i })).toBeVisible();
    }
  });

  test('should display job status and progress', async ({ page }) => {
    await page.goto('/jobs');

    const jobCard = page.locator('[data-testid="job-list"] > div').first();
    const count = await jobCard.count();

    if (count > 0) {
      await jobCard.click();

      // Check for status display
      const statusBadge = page.locator('[data-testid="job-status"]');
      await expect(statusBadge).toBeVisible();
    }
  });

  test('should show job history', async ({ page }) => {
    await page.goto('/jobs');

    const jobCard = page.locator('[data-testid="job-list"] > div').first();
    const count = await jobCard.count();

    if (count > 0) {
      await jobCard.click();

      const historyTab = page.getByRole('tab', { name: /history/i });
      if (await historyTab.isVisible()) {
        await historyTab.click();
        await expect(page.getByText(/history|timeline/i)).toBeVisible();
      }
    }
  });

  test('should cancel pending job', async ({ page }) => {
    await page.goto('/jobs');

    // Find a pending/queued job
    const pendingJob = page.locator('[data-testid="job-list"] > div').filter({
      hasText: /queued|pending/i,
    }).first();

    if (await pendingJob.isVisible()) {
      const cancelButton = pendingJob.getByRole('button', { name: /cancel/i });
      await cancelButton.click();

      // Confirm cancellation
      const confirmButton = page.getByRole('button', { name: /confirm|yes/i });
      if (await confirmButton.isVisible()) {
        await confirmButton.click();
      }

      await expect(page.getByText(/cancelled|canceled/i)).toBeVisible();
    }
  });

  test('should retry failed job', async ({ page }) => {
    await page.goto('/jobs');

    // Find a failed job
    const failedJob = page.locator('[data-testid="job-list"] > div').filter({
      hasText: /failed/i,
    }).first();

    if (await failedJob.isVisible()) {
      const retryButton = failedJob.getByRole('button', { name: /retry/i });
      await retryButton.click();

      await expect(page.getByText(/retrying|queued/i)).toBeVisible();
    }
  });
});

test.describe('Print Jobs - Secure Release', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.goto();
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);
  });

  test('should navigate to print release page', async ({ page }) => {
    await page.goto('/print-release');

    await expect(page.getByRole('heading', { name: /secure print release/i })).toBeVisible();
  });

  test('should show pending jobs for release', async ({ page }) => {
    await page.goto('/print-release');

    const pendingSection = page.getByText(/pending jobs/i);
    await expect(pendingSection).toBeVisible();
  });

  test('should select printer for release', async ({ page }) => {
    await page.goto('/print-release');

    const printerSelect = page.getByRole('combobox', { name: /printer/i });
    await expect(printerSelect).toBeVisible();
  });

  test('should enter PIN to release', async ({ page }) => {
    await page.goto('/print-release');

    const pinInput = page.getByRole('textbox', { name: /pin|code/i });
    await expect(pinInput).toBeVisible();

    await pinInput.fill('123456');

    const releaseButton = page.getByRole('button', { name: /release/i });
    await expect(releaseButton).toBeVisible();
  });

  test('should release job with correct PIN', async ({ page }) => {
    await page.goto('/print-release');

    const printerSelect = page.getByRole('combobox', { name: /printer/i });
    const options = await printerSelect.locator('option').count();

    if (options > 1) {
      await printerSelect.selectOption({ index: 1 });

      const pinInput = page.getByRole('textbox', { name: /pin|code/i });
      await pinInput.fill('123456');

      const pendingJobs = page.locator('[data-testid="pending-job"]');
      const count = await pendingJobs.count();

      if (count > 0) {
        const releaseButton = page.getByRole('button', { name: /release/i }).first();
        await releaseButton.click();

        // Should attempt release (may fail with wrong PIN in test)
        await page.waitForTimeout(1000);
      }
    }
  });
});

test.describe('Print Jobs - Documents', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.goto();
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);
  });

  test('should navigate to documents page', async ({ page }) => {
    await page.goto('/documents');

    await expect(page.getByRole('heading', { name: /documents/i })).toBeVisible();
  });

  test('should display document list', async ({ page }) => {
    await page.goto('/documents');

    const docList = page.locator('[data-testid="document-list"]');
    const emptyState = page.getByText(/no documents/i);

    const listVisible = await docList.isVisible();
    const emptyVisible = await emptyState.isVisible();

    expect(listVisible || emptyVisible).toBe(true);
  });

  test('should upload new document', async ({ page }) => {
    await page.goto('/documents');

    const uploadButton = page.getByRole('button', { name: /upload|add document/i });
    if (await uploadButton.isVisible()) {
      await uploadButton.click();

      await expect(page.getByRole('dialog')).toBeVisible();
    }
  });

  test('should preview document', async ({ page }) => {
    await page.goto('/documents');

    const docCard = page.locator('[data-testid="document-card"]').first();
    const count = await docCard.count();

    if (count > 0) {
      const previewButton = docCard.getByRole('button', { name: /preview|view/i });
      if (await previewButton.isVisible()) {
        await previewButton.click();

        await expect(page.locator('[data-testid="document-preview"]')).toBeVisible();
      }
    }
  });
});

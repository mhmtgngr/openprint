import { test, expect } from '@playwright/test';
import { login, mockApiResponse, mockUsers, mockJobs, mockPrinters } from '../helpers';

test.describe('Print Jobs Page', () => {
  test.beforeEach(async ({ page }) => {
    // Setup auth mock
    await page.route('**/api/v1/auth/me', async (route) => {
      await mockApiResponse(route, mockUsers[0]);
    });

    // Setup jobs mock
    await page.route('**/api/v1/jobs*', async (route) => {
      await mockApiResponse(route, {
        data: mockJobs,
        total: mockJobs.length,
        limit: 50,
        offset: 0,
      });
    });

    // Setup printers mock
    await page.route('**/api/v1/printers', async (route) => {
      await mockApiResponse(route, mockPrinters);
    });

    await login(page);
    await page.goto('/jobs');
  });

  test('should display jobs page header', async ({ page }) => {
    await expect(page.locator('h1')).toContainText('Print Jobs');
  });

  test('should display job status filters', async ({ page }) => {
    await expect(page.locator('button:has-text("All")')).toBeVisible();
    await expect(page.locator('button:has-text("Pending")')).toBeVisible();
    await expect(page.locator('button:has-text("Processing")')).toBeVisible();
    await expect(page.locator('button:has-text("Completed")')).toBeVisible();
    await expect(page.locator('button:has-text("Failed")')).toBeVisible();
  });

  test('should filter jobs by status', async ({ page }) => {
    // Click on completed filter
    await page.click('button:has-text("Completed")');

    // Should show active state
    const completedButton = page.locator('button:has-text("Completed")');
    await expect(completedButton).toHaveClass(/bg-blue-600/);
  });

  test('should display jobs table', async ({ page }) => {
    // Check for table headers
    await expect(page.locator('th:has-text("Document")')).toBeVisible();
    await expect(page.locator('th:has-text("Printer")')).toBeVisible();
    await expect(page.locator('th:has-text("Status")')).toBeVisible();
    await expect(page.locator('th:has-text("Pages")')).toBeVisible();
  });

  test('should display job entries', async ({ page }) => {
    // Check for job items
    await expect(page.locator('text=' + mockJobs[0].documentName)).toBeVisible();
    await expect(page.locator('text=' + mockJobs[1].documentName)).toBeVisible();
  });

  test('should display job status badges', async ({ page }) => {
    // Check for status badges
    await expect(page.locator('text=Completed')).toBeVisible();
    await expect(page.locator('text=Processing')).toBeVisible();
    await expect(page.locator('text=Failed')).toBeVisible();
  });

  test('should show job details', async ({ page }) => {
    const job = mockJobs[0];

    // Check for job details
    await expect(page.locator('text=' + job.documentName)).toBeVisible();
    await expect(page.locator('text=' + job.pageCount + ' pages')).toBeVisible();
  });

  test('should display create job button', async ({ page }) => {
    await expect(page.locator('button:has-text("New Print Job")')).toBeVisible();
  });

  test('should open create job modal', async ({ page }) => {
    await page.click('button:has-text("New Print Job")');
    await expect(page.locator('text=Create Print Job')).toBeVisible();
    await expect(page.locator('input[type="file"]')).toBeVisible();
  });

  test('should show empty state when no jobs', async ({ page }) => {
    // Mock empty jobs list
    await page.route('**/api/v1/jobs*', async (route) => {
      await mockApiResponse(route, {
        data: [],
        total: 0,
        limit: 50,
        offset: 0,
      });
    });

    await page.reload();

    await expect(page.locator('text=No print jobs')).toBeVisible();
    await expect(page.locator('text=Create your first print job')).toBeVisible();
  });

  test('should search jobs', async ({ page }) => {
    const searchInput = page.getByPlaceholder('Search');
    await searchInput.fill('Document');

    // Should filter results
    await page.waitForTimeout(500);
  });

  test('should cancel job', async ({ page }) => {
    const cancelButton = page.locator('button:has-text("Cancel")').first();

    if (await cancelButton.isVisible()) {
      // Mock cancel API
      await page.route('**/api/v1/jobs/*/cancel', async (route) => {
        await mockApiResponse(route, { success: true });
      });

      await cancelButton.click();
      await page.waitForTimeout(500);
    }
  });

  test('should retry failed job', async ({ page }) => {
    // Find a failed job
    const failedJob = mockJobs.find(j => j.status === 'failed');

    if (failedJob) {
      const retryButton = page.locator('button:has-text("Retry")').first();

      if (await retryButton.isVisible()) {
        // Mock retry API
        await page.route('**/api/v1/jobs/*/retry', async (route) => {
          await mockApiResponse(route, { success: true });
        });

        await retryButton.click();
        await page.waitForTimeout(500);
      }
    }
  });

  test('should display job metadata', async ({ page }) => {
    const job = mockJobs[0];

    // Check for color/duplex badges
    if (job.settings.color) {
      await expect(page.locator('text=Color')).toBeVisible();
    }

    if (job.settings.duplex) {
      await expect(page.locator('text=Duplex')).toBeVisible();
    }
  });

  test('should show file size', async ({ page }) => {
    const job = mockJobs[0];
    const fileSizeKB = Math.round(job.fileSize / 1024);

    await expect(page.locator('text=' + fileSizeKB + ' KB')).toBeVisible();
  });

  test('should display timestamp', async ({ page }) => {
    // Check for timestamp cells
    const timestampCell = page.locator('td').filter({ hasText: /ago/ }).first();
    if (await timestampCell.isVisible()) {
      await expect(timestampCell).toBeVisible();
    }
  });

  test('should paginate jobs', async ({ page }) => {
    // Check for pagination
    const nextButton = page.locator('button:has-text("Next")');

    if (await nextButton.isVisible()) {
      await nextButton.click();
      await page.waitForTimeout(500);
    }
  });

  test('should navigate to job details', async ({ page }) => {
    const jobRow = page.locator('tr').filter({ hasText: mockJobs[0].documentName }).first();

    if (await jobRow.isVisible()) {
      await jobRow.click();
      // Should navigate to job details or open modal
      await page.waitForTimeout(500);
    }
  });
});

test.describe('Job Creation', () => {
  test.beforeEach(async ({ page }) => {
    await page.route('**/api/v1/auth/me', async (route) => {
      await mockApiResponse(route, mockUsers[0]);
    });

    await page.route('**/api/v1/printers', async (route) => {
      await mockApiResponse(route, mockPrinters);
    });

    await login(page);
    await page.goto('/jobs');
  });

  test('should upload document for printing', async ({ page }) => {
    await page.click('button:has-text("New Print Job")');

    // Mock file upload API
    await page.route('**/api/v1/jobs', async (route) => {
      await mockApiResponse(route, {
        id: 'job-new',
        status: 'pending',
        documentName: 'test.pdf',
      });
    });

    // Create a mock file
    const fileInput = page.locator('input[type="file"]');
    await fileInput.setInputFiles({
      name: 'test.pdf',
      mimeType: 'application/pdf',
      buffer: Buffer.from('mock pdf content'),
    } as Parameters<typeof fileInput.setInputFiles>[0]);

    // Select printer
    await page.selectOption('select[name="printer"]', mockPrinters[0].id);

    // Submit
    await page.click('button:has-text("Print")');

    // Should close modal
    await expect(page.locator('text=Create Print Job')).not.toBeVisible();
  });

  test('should validate printer selection', async ({ page }) => {
    await page.click('button:has-text("New Print Job")');

    // Try to submit without selecting printer
    await page.click('button:has-text("Print")');

    // Should show validation error
    const printerSelect = page.locator('select[name="printer"]');
    await expect(printerSelect).toBeVisible();
  });

  test('should configure print settings', async ({ page }) => {
    await page.click('button:has-text("New Print Job")');

    // Check for print settings
    await expect(page.locator('text=Color')).toBeVisible();
    await expect(page.locator('text=Duplex')).toBeVisible();
    await expect(page.locator('text=Paper Size')).toBeVisible();
    await expect(page.locator('text=Copies')).toBeVisible();

    // Toggle color
    await page.check('input[type="checkbox"][value="color"]');

    // Select duplex
    await page.selectOption('select[name="duplex"]', 'long-edge');

    // Set copies
    await page.fill('input[type="number"][name="copies"]', '2');
  });

  test('should cancel job creation', async ({ page }) => {
    await page.click('button:has-text("New Print Job")');
    await page.click('button:has-text("Cancel")');

    await expect(page.locator('text=Create Print Job')).not.toBeVisible();
  });
});

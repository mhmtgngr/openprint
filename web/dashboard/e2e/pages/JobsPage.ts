/**
 * Jobs Page Object
 * Handles print job management, creation, filtering, and status changes
 */
import { Page, Locator, expect } from '@playwright/test';
import { BasePage } from './BasePage';
import { mockApiResponse, mockJobs, mockPrinters } from '../helpers';

export class JobsPage extends BasePage {
  // Page elements
  readonly heading: Locator;
  readonly jobsList: Locator;
  readonly jobItems: Locator;
  readonly createJobButton: Locator;
  readonly searchInput: Locator;
  readonly filterButtons: Locator;
  readonly statusFilters: Locator;
  readonly emptyState: Locator;
  readonly jobModal: Locator;
  readonly modalTitle: Locator;
  readonly modalCloseButton: Locator;

  // Job creation form elements
  readonly fileInput: Locator;
  readonly printerSelect: Locator;
  readonly copiesInput: Locator;
  readonly colorCheckbox: Locator;
  readonly duplexCheckbox: Locator;
  readonly paperSizeSelect: Locator;
  readonly submitJobButton: Locator;
  readonly cancelButton: Locator;

  // Job detail elements
  readonly jobDetailPage: Locator;
  readonly jobDetailName: Locator;
  readonly jobDetailStatus: Locator;
  readonly jobDetailPrinter: Locator;
  readonly jobDetailPages: Locator;
  readonly jobDetailCost: Locator;
  readonly jobCancelButton: Locator;
  readonly jobRetryButton: Locator;
  readonly jobDownloadButton: Locator;

  // Pagination
  readonly pagination: Locator;
  readonly nextPageButton: Locator;
  readonly prevPageButton: Locator;
  readonly pageIndicator: Locator;

  constructor(page: Page) {
    super(page);

    // Initialize locators
    this.heading = page.locator('h1, [data-testid="jobs-heading"]');
    this.jobsList = page.locator('[data-testid="jobs-list"], .jobs-list');
    this.jobItems = page.locator('[data-testid="job-item"], .job-item');
    this.createJobButton = page.locator('button:has-text("New Job"), button:has-text("Create Job"), [data-testid="create-job-button"]');
    this.searchInput = page.locator('input[type="search"], [data-testid="search-input"], input[placeholder*="search"]');
    this.filterButtons = page.locator('[data-testid="filter-buttons"], .filter-buttons');
    this.statusFilters = page.locator('button[data-status], [data-testid="status-filter"]');
    this.emptyState = page.locator('[data-testid="empty-state"], .empty-state');
    this.jobModal = page.locator('[data-testid="job-modal"], .modal, dialog');
    this.modalTitle = page.locator('[data-testid="modal-title"], .modal-title');
    this.modalCloseButton = page.locator('button[aria-label="Close"], .modal-close, button:has-text("Cancel")');

    // Job creation form
    this.fileInput = page.locator('input[type="file"], [data-testid="file-input"]');
    this.printerSelect = page.locator('select[name="printer"], [data-testid="printer-select"]');
    this.copiesInput = page.locator('input[name="copies"], [data-testid="copies-input"]');
    this.colorCheckbox = page.locator('input[name="color"], [data-testid="color-checkbox"]');
    this.duplexCheckbox = page.locator('input[name="duplex"], [data-testid="duplex-checkbox"]');
    this.paperSizeSelect = page.locator('select[name="paperSize"], [data-testid="paper-size-select"]');
    this.submitJobButton = page.locator('button[type="submit"]:has-text("Print"), [data-testid="submit-job"]');
    this.cancelButton = page.locator('button:has-text("Cancel")');

    // Job details
    this.jobDetailPage = page.locator('[data-testid="job-detail"]');
    this.jobDetailName = page.locator('[data-testid="job-name"], .job-name');
    this.jobDetailStatus = page.locator('[data-testid="job-status"], .job-status');
    this.jobDetailPrinter = page.locator('[data-testid="job-printer"], .job-printer');
    this.jobDetailPages = page.locator('[data-testid="job-pages"], .job-pages');
    this.jobDetailCost = page.locator('[data-testid="job-cost"], .job-cost');
    this.jobCancelButton = page.locator('button:has-text("Cancel"), [data-testid="cancel-job"]');
    this.jobRetryButton = page.locator('button:has-text("Retry"), [data-testid="retry-job"]');
    this.jobDownloadButton = page.locator('button:has-text("Download"), [data-testid="download-job"]');

    // Pagination
    this.pagination = page.locator('[data-testid="pagination"], .pagination');
    this.nextPageButton = page.locator('button:has-text("Next"), [aria-label="Next page"]');
    this.prevPageButton = page.locator('button:has-text("Previous"), [aria-label="Previous page"]');
    this.pageIndicator = page.locator('[data-testid="page-indicator"], .page-indicator');
  }

  /**
   * Navigate to jobs page
   */
  async goto() {
    await this.goto('/jobs');
    await this.waitForPageLoad();
  }

  /**
   * Setup API mocks for jobs page
   */
  async setupMocks() {
    // Mock jobs list
    await this.page.route('**/api/v1/jobs*', async (route) => {
      await mockApiResponse(route, {
        data: mockJobs,
        total: mockJobs.length,
        limit: 50,
        offset: 0,
      });
    });

    // Mock printers for job creation
    await this.page.route('**/api/v1/printers', async (route) => {
      await mockApiResponse(route, { printers: mockPrinters });
    });

    // Mock job creation
    await this.page.route('**/api/v1/jobs', async (route) => {
      if (route.request().method() === 'POST') {
        await mockApiResponse(route, {
          id: 'new-job-id',
          status: 'queued',
          createdAt: new Date().toISOString(),
        });
      }
    });

    // Mock job cancellation
    await this.page.route('**/api/v1/jobs/*/cancel', async (route) => {
      if (route.request().method() === 'POST') {
        await mockApiResponse(route, {
          id: route.request().url().match(/\/jobs\/(.*)\/cancel/)?.[1],
          status: 'cancelled',
        });
      }
    });

    // Mock job retry
    await this.page.route('**/api/v1/jobs/*/retry', async (route) => {
      if (route.request().method() === 'POST') {
        await mockApiResponse(route, {
          id: route.request().url().match(/\/jobs\/(.*)\/retry/)?.[1],
          status: 'queued',
        });
      }
    });
  }

  /**
   * Verify jobs page is loaded
   */
  async isLoaded(): Promise<boolean> {
    await this.heading.waitFor({ state: 'visible', timeout: 5000 });
    return await this.heading.isVisible();
  }

  /**
   * Click create job button
   */
  async openCreateJobModal() {
    await this.createJobButton.click();
    await expect(this.jobModal).toBeVisible();
  }

  /**
   * Fill job creation form
   */
  async fillJobForm(data: {
    file?: string;
    printer?: string;
    copies?: number;
    color?: boolean;
    duplex?: boolean;
    paperSize?: string;
  }) {
    if (data.file) {
      await this.fileInput.setInputFiles(data.file);
    }

    if (data.printer) {
      await this.printerSelect.selectOption(data.printer);
    }

    if (data.copies !== undefined) {
      await this.copiesInput.fill(String(data.copies));
    }

    if (data.color !== undefined) {
      if (data.color) {
        await this.colorCheckbox.check();
      } else {
        await this.colorCheckbox.uncheck();
      }
    }

    if (data.duplex !== undefined) {
      if (data.duplex) {
        await this.duplexCheckbox.check();
      } else {
        await this.duplexCheckbox.uncheck();
      }
    }

    if (data.paperSize) {
      await this.paperSizeSelect.selectOption(data.paperSize);
    }
  }

  /**
   * Submit job creation form
   */
  async submitJob() {
    await this.submitJobButton.click();
    await expect(this.jobModal).not.toBeVisible();
  }

  /**
   * Cancel job creation
   */
  async cancelJobCreation() {
    await this.cancelButton.click();
    await expect(this.jobModal).not.toBeVisible();
  }

  /**
   * Search for jobs
   */
  async searchJobs(query: string) {
    await this.searchInput.fill(query);
    await this.page.waitForTimeout(500); // Wait for debounce
  }

  /**
   * Filter jobs by status
   */
  async filterByStatus(status: string) {
    const filter = this.statusFilters.filter({ hasText: status });
    await filter.click();
  }

  /**
   * Click on a job item to view details
   */
  async viewJobDetails(jobId: string) {
    const jobItem = this.jobItems.filter({ hasText: jobId }).first();
    await jobItem.click();
    await expect(this.jobDetailPage).toBeVisible();
  }

  /**
   * Get job count from list
   */
  async getJobCount(): Promise<number> {
    return await this.jobItems.count();
  }

  /**
   * Verify empty state is shown
   */
  async verifyEmptyState() {
    await expect(this.emptyState).toBeVisible();
    await expect(this.jobItems).toHaveCount(0);
  }

  /**
   * Cancel a job
   */
  async cancelJob(jobId: string) {
    // Navigate to job detail or use inline action
    await this.viewJobDetails(jobId);
    await this.jobCancelButton.click();
    // Confirm if prompted
    const confirmButton = this.page.locator('button:has-text("Confirm")');
    if (await confirmButton.isVisible()) {
      await confirmButton.click();
    }
  }

  /**
   * Retry a failed job
   */
  async retryJob(jobId: string) {
    await this.viewJobDetails(jobId);
    await this.jobRetryButton.click();
  }

  /**
   * Download job document
   */
  async downloadJob(jobId: string) {
    await this.viewJobDetails(jobId);
    const downloadPromise = this.page.waitForEvent('download');
    await this.jobDownloadButton.click();
    await downloadPromise;
  }

  /**
   * Navigate to next page
   */
  async nextPage() {
    await this.nextPageButton.click();
  }

  /**
   * Navigate to previous page
   */
  async prevPage() {
    await this.prevPageButton.click();
  }

  /**
   * Get current page number
   */
  async getCurrentPage(): Promise<string> {
    return await this.pageIndicator.textContent() || '1';
  }

  /**
   * Verify job status badge
   */
  async verifyJobStatus(jobId: string, status: string) {
    const jobItem = this.jobItems.filter({ hasText: jobId });
    const statusBadge = jobItem.locator('[data-testid="job-status"], .status-badge');
    await expect(statusBadge).toHaveText(status);
  }

  /**
   * Verify job metadata
   */
  async verifyJobMetadata(jobId: string, metadata: {
    documentName?: string;
    pageCount?: number;
    printerName?: string;
  }) {
    const jobItem = this.jobItems.filter({ hasText: jobId });

    if (metadata.documentName) {
      await expect(jobItem).toContainText(metadata.documentName);
    }

    if (metadata.pageCount) {
      await expect(jobItem).toContainText(String(metadata.pageCount));
    }

    if (metadata.printerName) {
      await expect(jobItem).toContainText(metadata.printerName);
    }
  }

  /**
   * Get all visible job statuses
   */
  async getVisibleStatuses(): Promise<string[]> {
    const badges = this.jobItems.locator('[data-testid="job-status"], .status-badge');
    const count = await badges.count();
    const statuses: string[] = [];

    for (let i = 0; i < count; i++) {
      statuses.push(await badges.nth(i).textContent() || '');
    }

    return statuses;
  }

  /**
   * Sort jobs by column
   */
  async sortBy(column: string) {
    const header = this.page.locator(`th:has-text("${column}"), [data-testid="sort-${column}"]`);
    await header.click();
  }

  /**
   * Verify job creation validation
   */
  async verifyJobCreationValidation() {
    await this.openCreateJobModal();
    await this.submitJobButton.click();

    // Should show validation errors
    const errorMessages = this.jobModal.locator('.error, [data-testid="validation-error"]');
    await expect(errorMessages).toBeVisible();
  }

  /**
   * Close job detail modal/page
   */
  async closeJobDetail() {
    const closeButton = this.page.locator('button[aria-label="Close"], .back-button');
    await closeButton.click();
  }

  /**
   * Verify jobs are sorted by date
   */
  async verifyJobsSortedByDate(order: 'asc' | 'desc' = 'desc') {
    // Implementation depends on how dates are displayed
    // This is a placeholder for the logic
  }

  /**
   * Filter by date range
   */
  async filterByDateRange(from: string, to: string) {
    const dateFilterButton = this.page.locator('button:has-text("Date"), [data-testid="date-filter"]');
    await dateFilterButton.click();

    await this.page.fill('input[placeholder*="From"]', from);
    await this.page.fill('input[placeholder*="To"]', to);

    await this.page.locator('button:has-text("Apply")').click();
  }

  /**
   * Bulk cancel jobs
   */
  async bulkCancelJobs(jobIds: string[]) {
    for (const id of jobIds) {
      const checkbox = this.jobItems.filter({ hasText: id }).locator('input[type="checkbox"]');
      await checkbox.check();
    }

    const bulkCancelButton = this.page.locator('button:has-text("Cancel Selected")');
    await bulkCancelButton.click();

    const confirmButton = this.page.locator('button:has-text("Confirm")');
    await confirmButton.click();
  }

  /**
   * Verify job cost estimate
   */
  async verifyJobCostEstimate(expectedCost: number) {
    const costElement = this.jobModal.locator('[data-testid="cost-estimate"], .cost-estimate');
    const costText = await costElement.textContent();
    expect(costText).toContain(String(expectedCost));
  }
}

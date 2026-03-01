/**
 * Secure Print Release Page Object
 * Handles PIN-based job release and secure print queue
 */
import { Page, Locator, expect } from '@playwright/test';
import { BasePage } from './BasePage';
import { mockApiResponse } from '../helpers';

export interface SecurePrintJob {
  id: string;
  documentName: string;
  userId: string;
  userName: string;
  printerId: string;
  status: 'pending' | 'released' | 'expired';
  pin: string;
  createdAt: string;
  expiresAt: string;
}

export class SecurePrintPage extends BasePage {
  // Page elements
  readonly heading: Locator;
  readonly releaseForm: Locator;
  readonly pinInput: Locator;
  readonly releaseButton: Locator;
  readonly jobList: Locator;
  readonly jobItems: Locator;
  readonly emptyState: Locator;

  // Job detail
  readonly jobDetailName: Locator;
  readonly jobDetailUser: Locator;
  readonly jobDetailPrinter: Locator;
  readonly jobDetailExpiry: Locator;
  readonly jobDetailStatus: Locator;
  readonly releaseJobButton: Locator;
  readonly cancelJobButton: Locator;

  // Settings
  readonly settingsButton: Locator;
  readonly pinLengthInput: Locator;
  readonly expiryTimeInput: Locator;
  readonly requireAuthToggle: Locator;
  readonly saveSettingsButton: Locator;

  // PIN display
  readonly pinDisplay: Locator;
  readonly copyPinButton: Locator;
  readonly regeneratePinButton: Locator;

  constructor(page: Page) {
    super(page);

    // Initialize locators
    this.heading = page.locator('h1, [data-testid="secure-print-heading"]');
    this.releaseForm = page.locator('[data-testid="release-form"], form[data-type="release"]');
    this.pinInput = page.locator('input[name="pin"], input[type="password"], [data-testid="pin-input"]');
    this.releaseButton = page.locator('button[type="submit"]:has-text("Release"), [data-testid="release-button"]');
    this.jobList = page.locator('[data-testid="secure-jobs-list"], .secure-jobs-list');
    this.jobItems = page.locator('[data-testid="job-item"], .job-item');
    this.emptyState = page.locator('[data-testid="empty-state"], .empty-state');

    // Job detail
    this.jobDetailName = page.locator('[data-testid="job-name"], .job-name');
    this.jobDetailUser = page.locator('[data-testid="job-user"], .job-user');
    this.jobDetailPrinter = page.locator('[data-testid="job-printer"], .job-printer');
    this.jobDetailExpiry = page.locator('[data-testid="job-expiry"], .job-expiry');
    this.jobDetailStatus = page.locator('[data-testid="job-status"], .job-status');
    this.releaseJobButton = page.locator('button:has-text("Release"), [data-testid="release-job"]');
    this.cancelJobButton = page.locator('button:has-text("Cancel"), [data-testid="cancel-job"]');

    // Settings
    this.settingsButton = page.locator('button:has-text("Settings"), [data-testid="settings-button"]');
    this.pinLengthInput = page.locator('input[name="pinLength"], [data-testid="pin-length"]');
    this.expiryTimeInput = page.locator('input[name="expiryTime"], [data-testid="expiry-time"]');
    this.requireAuthToggle = page.locator('input[name="requireAuth"], [data-testid="require-auth"]');
    this.saveSettingsButton = page.locator('button:has-text("Save"), [data-testid="save-settings"]');

    // PIN display
    this.pinDisplay = page.locator('[data-testid="pin-display"], .pin-display');
    this.copyPinButton = page.locator('button:has-text("Copy"), [data-testid="copy-pin"]');
    this.regeneratePinButton = page.locator('button:has-text("Regenerate"), [data-testid="regenerate-pin"]');
  }

  /**
   * Navigate to secure print page
   */
  async goto() {
    await this.goto('/print-release');
    await this.waitForPageLoad();
  }

  /**
   * Setup API mocks for secure print
   */
  async setupMocks() {
    const mockSecureJobs: SecurePrintJob[] = [
      {
        id: 'secure-job-1',
        documentName: 'Confidential Report.pdf',
        userId: 'user-1',
        userName: 'John Doe',
        printerId: 'printer-1',
        status: 'pending',
        pin: '1234',
        createdAt: new Date().toISOString(),
        expiresAt: new Date(Date.now() + 3600000).toISOString(),
      },
      {
        id: 'secure-job-2',
        documentName: 'Payroll Data.xlsx',
        userId: 'user-2',
        userName: 'Jane Smith',
        printerId: 'printer-1',
        status: 'pending',
        pin: '5678',
        createdAt: new Date().toISOString(),
        expiresAt: new Date(Date.now() + 3600000).toISOString(),
      },
    ];

    // Mock secure jobs list
    await this.page.route('**/api/v1/secure-print/jobs', async (route) => {
      await mockApiResponse(route, {
        jobs: mockSecureJobs,
        total: mockSecureJobs.length,
      });
    });

    // Mock job release
    await this.page.route('**/api/v1/secure-print/jobs/*/release', async (route) => {
      if (route.request().method() === 'POST') {
        await mockApiResponse(route, {
          message: 'Job released successfully',
          status: 'released',
        });
      }
    });

    // Mock job cancellation
    await this.page.route('**/api/v1/secure-print/jobs/*/cancel', async (route) => {
      if (route.request().method() === 'POST') {
        await mockApiResponse(route, {
          message: 'Job cancelled',
          status: 'cancelled',
        });
      }
    });

    // Mock settings
    await this.page.route('**/api/v1/secure-print/settings', async (route) => {
      if (route.request().method() === 'GET') {
        await mockApiResponse(route, {
          pinLength: 4,
          expiryMinutes: 60,
          requireAuth: true,
        });
      } else if (route.request().method() === 'PUT' || route.request().method() === 'PATCH') {
        await mockApiResponse(route, {
          message: 'Settings updated',
        });
      }
    });

    // Mock PIN generation
    await this.page.route('**/api/v1/secure-print/jobs/*/pin', async (route) => {
      if (route.request().method() === 'POST') {
        await mockApiResponse(route, {
          pin: '9999',
        });
      }
    });
  }

  /**
   * Verify secure print page is loaded
   */
  async isLoaded(): Promise<boolean> {
    await this.heading.waitFor({ state: 'visible', timeout: 5000 });
    return await this.heading.isVisible();
  }

  /**
   * Release a job using PIN
   */
  async releaseJobWithPin(pin: string) {
    await this.pinInput.fill(pin);
    await this.releaseButton.click();
    await this.verifyToast('Job released', 'success');
  }

  /**
   * Release job from detail view
   */
  async releaseJob(jobId: string) {
    const jobItem = this.jobItems.filter({ hasText: jobId });
    await jobItem.locator('button:has-text("Release"), [data-testid="release-job"]').click();

    // Enter PIN if prompted
    const pinPrompt = this.page.locator('[data-testid="pin-prompt"]');
    if (await pinPrompt.isVisible()) {
      await this.page.locator('input[type="password"]').fill('1234');
      await this.page.locator('button:has-text("Confirm")').click();
    }
  }

  /**
   * Cancel a secure print job
   */
  async cancelJob(jobId: string) {
    const jobItem = this.jobItems.filter({ hasText: jobId });
    await jobItem.locator('button:has-text("Cancel"), [data-testid="cancel-job"]').click();

    const confirmButton = this.page.locator('button:has-text("Confirm")');
    if (await confirmButton.isVisible()) {
      await confirmButton.click();
    }
  }

  /**
   * Get secure job count
   */
  async getJobCount(): Promise<number> {
    return await this.jobItems.count();
  }

  /**
   * Filter jobs by status
   */
  async filterByStatus(status: 'pending' | 'released' | 'expired') {
    const filterButton = this.page.locator(`button:has-text("${status}"), [data-status="${status}"]`);
    await filterButton.click();
  }

  /**
   * Filter jobs by user
   */
  async filterByUser(userName: string) {
    const userFilter = this.page.locator('[data-testid="user-filter"]');
    await userFilter.selectOption(userName);
  }

  /**
   * View job details
   */
  async viewJobDetails(jobId: string) {
    const jobItem = this.jobItems.filter({ hasText: jobId });
    await jobItem.click();
  }

  /**
   * Open settings modal
   */
  async openSettings() {
    await this.settingsButton.click();
    const settingsModal = this.page.locator('[data-testid="settings-modal"], .settings-modal');
    await expect(settingsModal).toBeVisible();
  }

  /**
   * Update secure print settings
   */
  async updateSettings(settings: {
    pinLength?: number;
    expiryMinutes?: number;
    requireAuth?: boolean;
  }) {
    await this.openSettings();

    if (settings.pinLength !== undefined) {
      await this.pinLengthInput.fill(String(settings.pinLength));
    }

    if (settings.expiryMinutes !== undefined) {
      await this.expiryTimeInput.fill(String(settings.expiryMinutes));
    }

    if (settings.requireAuth !== undefined) {
      if (settings.requireAuth) {
        await this.requireAuthToggle.check();
      } else {
        await this.requireAuthToggle.uncheck();
      }
    }

    await this.saveSettingsButton.click();
    await this.verifyToast('Settings saved', 'success');
  }

  /**
   * Copy job PIN
   */
  async copyPin(jobId: string) {
    const jobItem = this.jobItems.filter({ hasText: jobId });
    await jobItem.locator('button:has-text("Copy"), [data-testid="copy-pin"]').click();
    await this.verifyToast('PIN copied', 'success');
  }

  /**
   * Regenerate job PIN
   */
  async regeneratePin(jobId: string) {
    const jobItem = this.jobItems.filter({ hasText: jobId });
    await jobItem.locator('button:has-text("Regenerate"), [data-testid="regenerate-pin"]').click();

    const confirmButton = this.page.locator('button:has-text("Confirm")');
    if (await confirmButton.isVisible()) {
      await confirmButton.click();
    }

    await this.verifyToast('PIN regenerated', 'success');
  }

  /**
   * Verify job status
   */
  async verifyJobStatus(jobId: string, status: 'pending' | 'released' | 'expired') {
    const jobItem = this.jobItems.filter({ hasText: jobId });
    const statusBadge = jobItem.locator('[data-testid="job-status"], .status-badge');
    await expect(statusBadge).toHaveText(status);
  }

  /**
   * Verify empty state
   */
  async verifyEmptyState() {
    await expect(this.emptyState).toBeVisible();
    await expect(this.emptyState).toContainText(/no|empty|pending/i);
  }

  /**
   * Verify PIN validation
   */
  async verifyPinValidation() {
    await this.pinInput.fill('12'); // Too short
    await this.pinInput.blur();

    const error = this.pinInput.locator('..').locator('.error, [data-testid="validation-error"]');
    await expect(error).toBeVisible();
  }

  /**
   * Verify expiry countdown
   */
  async verifyExpiryCountdown(jobId: string): Promise<string> {
    const jobItem = this.jobItems.filter({ hasText: jobId });
    const countdown = jobItem.locator('[data-testid="expiry-countdown"], .countdown');
    return await countdown.textContent() || '';
  }

  /**
   * Verify secure print is enabled
   */
  async verifySecurePrintEnabled(): Promise<boolean> {
    const enabledIndicator = this.page.locator('[data-testid="secure-enabled"], .secure-enabled');
    return await enabledIndicator.isVisible();
  }

  /**
   * Print test secure job
   */
  async printTestSecureJob() {
    const testButton = this.page.locator('button:has-text("Test Print"), [data-testid="test-secure-print"]');
    await testButton.click();

    // Should generate a test job with PIN
    await this.verifyToast('Test job created', 'success');
  }

  /**
   * Bulk release jobs
   */
  async bulkReleaseJobs(jobIds: string[], pin: string) {
    for (const id of jobIds) {
      const checkbox = this.jobItems.filter({ hasText: id }).locator('input[type="checkbox"]');
      await checkbox.check();
    }

    const bulkReleaseButton = this.page.locator('button:has-text("Release Selected")');
    await bulkReleaseButton.click();

    // Enter PIN
    await this.pinInput.fill(pin);
    await this.page.locator('button:has-text("Confirm")').click();
  }

  /**
   * Verify user identity check
   */
  async verifyUserIdentityRequired(): Promise<boolean> {
    const identityCheck = this.page.locator('[data-testid="identity-check"], .identity-check');
    return await identityCheck.isVisible();
  }

  /**
   * Get job document name
   */
  async getJobDocumentName(jobId: string): Promise<string> {
    const jobItem = this.jobItems.filter({ hasText: jobId });
    const nameElement = jobItem.locator('[data-testid="job-name"], .document-name');
    return await nameElement.textContent() || '';
  }
}

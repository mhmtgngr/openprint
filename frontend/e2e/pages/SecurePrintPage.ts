import { type Page, type Locator, expect } from '@playwright/test';
import { BasePage } from './BasePage';

/**
 * Secure Print Release Page Object
 * Handles PIN-based secure print job release and management
 */
export class SecurePrintPage extends BasePage {
  // Page heading and sections
  readonly heading: Locator;
  readonly queueSection: Locator;
  readonly releasedSection: Locator;
  readonly settingsSection: Locator;

  // Job queue
  readonly jobList: Locator;
  readonly jobCard: (jobId: string) => Locator;
  readonly noJobsMessage: Locator;
  readonly refreshButton: Locator;
  readonly filterSelect: Locator;
  readonly searchInput: Locator;

  // Job actions
  readonly releaseButton: Locator;
  readonly releaseWithPinButton: Locator;
  readonly cancelButton: Locator;
  readonly previewButton: Locator;

  // PIN input
  readonly pinInput: Locator;
  readonly pinSubmitButton: Locator;
  readonly pinCancelButton: Locator;
  readonly pinVisibilityToggle: Locator;
  readonly pinErrorMessage: Locator;
  readonly pinAttemptsRemaining: Locator;

  // Job details
  readonly jobName: Locator;
  readonly jobPages: Locator;
  readonly jobColorMode: Locator;
  readonly jobDuplex: Locator;
  readonly jobCopies: Locator;
  readonly jobPrinter: Locator;
  readonly jobSubmittedTime: Locator;
  readonly jobFileSize: Locator;

  // Printer selection
  readonly printerSelect: Locator;
  readonly selectedPrinterDisplay: Locator;

  // Release confirmation modal
  readonly releaseConfirmModal: Locator;
  readonly releaseConfirmButton: Locator;
  readonly releaseDenyButton: Locator;
  readonly releaseConfirmPinInput: Locator;

  // Settings
  readonly enableSecurePrintToggle: Locator;
  readonly pinLengthSelect: Locator;
  readonly pinExpiryInput: Locator;
  readonly autoReleaseToggle: Locator;
  readonly saveSettingsButton: Locator;

  // History
  readonly historyTab: Locator;
  readonly historyList: Locator;

  // Status indicators
  readonly jobStatusBadge: Locator;
  readonly pinRequiredBadge: Locator;

  constructor(page: Page) {
    super(page);

    // Page heading and sections
    this.heading = page.getByRole('heading', { name: /secure print|print release|release station/i });
    this.queueSection = page.locator('[data-testid="secure-print-queue"], section:has-text("Queue")');
    this.releasedSection = page.locator('[data-testid="released-jobs"], section:has-text("Released")');
    this.settingsSection = page.locator('[data-testid="secure-print-settings"], section:has-text("Settings")');

    // Job queue
    this.jobList = page.locator('[data-testid="secure-print-jobs"]');
    this.jobCard = (jobId: string) =>
      page.locator(`[data-job-id="${jobId}"]`).or(page.locator('[data-testid="job-card"]').filter({ hasText: jobId }));
    this.noJobsMessage = page.getByText(/no jobs|queue is empty|nothing to release/i);
    this.refreshButton = page.getByRole('button', { name: /refresh/i });
    this.filterSelect = page.getByLabel(/filter|status/i);
    this.searchInput = page.getByPlaceholder(/search|find job/i);

    // Job actions
    this.releaseButton = page.getByRole('button', { name: /release/i }).first();
    this.releaseWithPinButton = page.getByRole('button', { name: /release with pin|enter pin/i });
    this.cancelButton = page.getByRole('button', { name: /cancel/i });
    this.previewButton = page.getByRole('button', { name: /preview/i });

    // PIN input
    this.pinInput = page.getByLabel(/pin|personal identification number/i);
    this.pinSubmitButton = page.getByRole('button', { name: /submit|verify|release/i });
    this.pinCancelButton = page.getByRole('button', { name: /cancel|close/i });
    this.pinVisibilityToggle = page.getByRole('button', { name: /show|hide/i });
    this.pinErrorMessage = page.locator('[data-testid="pin-error"], .error-message');
    this.pinAttemptsRemaining = page.locator('[data-testid="attempts-remaining"]');

    // Job details
    this.jobName = page.locator('[data-testid="job-name"]');
    this.jobPages = page.locator('[data-testid="job-pages"]');
    this.jobColorMode = page.locator('[data-testid="color-mode"]');
    this.jobDuplex = page.locator('[data-testid="duplex"]');
    this.jobCopies = page.locator('[data-testid="copies"]');
    this.jobPrinter = page.locator('[data-testid="printer"]');
    this.jobSubmittedTime = page.locator('[data-testid="submitted-time"]');
    this.jobFileSize = page.locator('[data-testid="file-size"]');

    // Printer selection
    this.printerSelect = page.getByLabel(/printer|destination/i);
    this.selectedPrinterDisplay = page.locator('[data-testid="selected-printer"]');

    // Release confirmation modal
    this.releaseConfirmModal = page.locator('[data-testid="release-modal"], .modal');
    this.releaseConfirmButton = page.getByRole('button', { name: /confirm|yes, release/i });
    this.releaseDenyButton = page.getByRole('button', { name: /no|don't release/i });
    this.releaseConfirmPinInput = page.locator('[data-testid="confirm-pin-input"]');

    // Settings
    this.enableSecurePrintToggle = page.getByRole('switch', { name: /enable secure print/i });
    this.pinLengthSelect = page.getByLabel(/pin length/i);
    this.pinExpiryInput = page.getByLabel(/pin expiry|expiry time/i);
    this.autoReleaseToggle = page.getByRole('switch', { name: /auto release/i });
    this.saveSettingsButton = page.getByRole('button', { name: /save|update settings/i });

    // History
    this.historyTab = page.getByRole('tab', { name: /history/i });
    this.historyList = page.locator('[data-testid="release-history"]');

    // Status indicators
    this.jobStatusBadge = page.locator('[data-testid="job-status"], .status-badge');
    this.pinRequiredBadge = page.locator('[data-testid="pin-required"], .badge:has-text("PIN")');
  }

  /**
   * Navigate to Secure Print page
   */
  async navigate(): Promise<void> {
    await this.goto('/print-release');
  }

  /**
   * Verify Secure Print page is loaded
   */
  async isLoaded(): Promise<boolean> {
    await this.page.waitForLoadState('networkidle');
    return await this.heading.isVisible();
  }

  /**
   * Get the number of jobs in queue
   */
  async getJobCount(): Promise<number> {
    return await this.jobList.locator('[data-testid="job-card"]').count();
  }

  /**
   * Check if queue is empty
   */
  async isQueueEmpty(): Promise<boolean> {
    return await this.noJobsMessage.isVisible();
  }

  /**
   * Refresh the job queue
   */
  async refreshQueue(): Promise<void> {
    await this.refreshButton.click();
    await this.page.waitForTimeout(1000);
  }

  /**
   * Search for a job
   */
  async searchJob(query: string): Promise<void> {
    await this.searchInput.fill(query);
    await this.page.waitForTimeout(500);
  }

  /**
   * Filter jobs by status
   */
  async filterByStatus(status: 'pending' | 'released' | 'cancelled'): Promise<void> {
    await this.filterSelect.selectOption(status);
  }

  /**
   * Release a job with PIN
   */
  async releaseJobWithPin(jobId: string, pin: string): Promise<void> {
    const card = this.jobCard(jobId);
    await card.locator('[data-testid="release-button"], button:has-text("Release")').click();

    // Enter PIN if prompted
    if (await this.pinInput.isVisible()) {
      await this.pinInput.fill(pin);
      await this.pinSubmitButton.click();
    }

    // Confirm if prompted
    if (await this.releaseConfirmButton.isVisible()) {
      await this.releaseConfirmButton.click();
    }
  }

  /**
   * Release a job without PIN (if already authenticated)
   */
  async releaseJob(jobId: string): Promise<void> {
    const card = this.jobCard(jobId);
    await card.locator('[data-testid="release-button"], button:has-text("Release")').click();

    // Confirm if prompted
    if (await this.releaseConfirmButton.isVisible()) {
      await this.releaseConfirmButton.click();
    }
  }

  /**
   * Cancel a job
   */
  async cancelJob(jobId: string): Promise<void> {
    const card = this.jobCard(jobId);
    await card.locator('[data-testid="cancel-button"], button:has-text("Cancel")').click();

    const confirmButton = this.page.getByRole('button', { name: /confirm|yes/i });
    if (await confirmButton.isVisible()) {
      await confirmButton.click();
    }
  }

  /**
   * Preview a job
   */
  async previewJob(jobId: string): Promise<void> {
    const card = this.jobCard(jobId);
    await card.locator('[data-testid="preview-button"], button:has-text("Preview")').click();

    // Preview modal should open
    await expect(this.page.locator('[data-testid="preview-modal"], .modal')).toBeVisible();
  }

  /**
   * Get job details
   */
  async getJobDetails(jobId: string): Promise<{
    name: string;
    pages: number;
    colorMode: string;
    duplex: boolean;
    copies: number;
    printer: string;
    submittedTime: string;
    fileSize: string;
    status: string;
  }> {
    const card = this.jobCard(jobId);

    return {
      name: await card.locator('[data-testid="job-name"]').textContent() || '',
      pages: parseInt(await card.locator('[data-testid="job-pages"]').textContent() || '0'),
      colorMode: await card.locator('[data-testid="color-mode"]').textContent() || '',
      duplex: (await card.locator('[data-testid="duplex"]').textContent() || '').toLowerCase().includes('yes'),
      copies: parseInt(await card.locator('[data-testid="copies"]').textContent() || '1'),
      printer: await card.locator('[data-testid="printer"]').textContent() || '',
      submittedTime: await card.locator('[data-testid="submitted-time"]').textContent() || '',
      fileSize: await card.locator('[data-testid="file-size"]').textContent() || '',
      status: await card.locator('[data-testid="job-status"]').textContent() || '',
    };
  }

  /**
   * Select printer for release
   */
  async selectPrinter(printerName: string): Promise<void> {
    await this.printerSelect.selectOption(printerName);
  }

  /**
   * Get selected printer
   */
  async getSelectedPrinter(): Promise<string> {
    const value = await this.printerSelect.inputValue();
    if (value) return value;
    return await this.selectedPrinterDisplay.textContent() || '';
  }

  /**
   * Enter PIN for job release
   */
  async enterPin(pin: string): Promise<void> {
    await this.pinInput.fill(pin);
    await this.pinSubmitButton.click();
  }

  /**
   * Cancel PIN entry
   */
  async cancelPinEntry(): Promise<void> {
    await this.pinCancelButton.click();
  }

  /**
   * Toggle PIN visibility
   */
  async togglePinVisibility(): Promise<void> {
    await this.pinVisibilityToggle.click();
  }

  /**
   * Get PIN error message
   */
  async getPinErrorMessage(): Promise<string> {
    return await this.pinErrorMessage.textContent() || '';
  }

  /**
   * Get remaining PIN attempts
   */
  async getRemainingAttempts(): Promise<number> {
    const text = await this.pinAttemptsRemaining.textContent() || '';
    const match = text.match(/\d+/);
    return match ? parseInt(match[0]) : 0;
  }

  /**
   * Enable secure print
   */
  async enableSecurePrint(): Promise<void> {
    await this.enableSecurePrintToggle.click();
    await this.saveSettingsButton.click();
  }

  /**
   * Disable secure print
   */
  async disableSecurePrint(): Promise<void> {
    if (await this.enableSecurePrintToggle.isChecked()) {
      await this.enableSecurePrintToggle.click();
    }
    await this.saveSettingsButton.click();
  }

  /**
   * Set PIN length
   */
  async setPinLength(length: 4 | 6 | 8): Promise<void> {
    await this.pinLengthSelect.selectOption(length.toString());
    await this.saveSettingsButton.click();
  }

  /**
   * Set PIN expiry time
   */
  async setPinExpiry(minutes: number): Promise<void> {
    await this.pinExpiryInput.fill(minutes.toString());
    await this.saveSettingsButton.click();
  }

  /**
   * Enable auto-release
   */
  async enableAutoRelease(): Promise<void> {
    await this.autoReleaseToggle.click();
    await this.saveSettingsButton.click();
  }

  /**
   * View release history
   */
  async viewHistory(): Promise<void> {
    await this.historyTab.click();
  }

  /**
   * Get history entries count
   */
  async getHistoryCount(): Promise<number> {
    return await this.historyList.locator('[data-testid="history-item"]').count();
  }

  /**
   * Check if job is in queue
   */
  async hasJob(jobId: string): Promise<boolean> {
    const count = await this.jobCard(jobId).count();
    return count > 0;
  }

  /**
   * Get job status
   */
  async getJobStatus(jobId: string): Promise<string> {
    const card = this.jobCard(jobId);
    return await card.locator('[data-testid="job-status"], .status-badge').textContent() || '';
  }

  /**
   * Verify job is released
   */
  async isJobReleased(jobId: string): Promise<boolean> {
    const status = await this.getJobStatus(jobId);
    return status.toLowerCase().includes('released') || status.toLowerCase().includes('complete');
  }

  /**
   * Verify job is cancelled
   */
  async isJobCancelled(jobId: string): Promise<boolean> {
    const status = await this.getJobStatus(jobId);
    return status.toLowerCase().includes('cancelled') || status.toLowerCase().includes('canceled');
  }

  /**
   * Verify PIN is required for job
   */
  async isPinRequired(jobId: string): Promise<boolean> {
    const card = this.jobCard(jobId);
    const badge = card.locator('[data-testid="pin-required"], .badge');
    return await badge.isVisible();
  }

  /**
   * Bulk release all jobs with PIN
   */
  async releaseAllJobs(pin: string): Promise<void> {
    const releaseAllButton = this.page.getByRole('button', { name: /release all/i });
    if (await releaseAllButton.isVisible()) {
      await releaseAllButton.click();

      // Enter PIN if prompted
      if (await this.pinInput.isVisible()) {
        await this.pinInput.fill(pin);
        await this.pinSubmitButton.click();
      }

      // Confirm if prompted
      if (await this.releaseConfirmButton.isVisible()) {
        await this.releaseConfirmButton.click();
      }
    }
  }

  /**
   * Get jobs by status
   */
  async getJobsByStatus(status: 'pending' | 'released' | 'cancelled'): Promise<string[]> {
    await this.filterByStatus(status);
    await this.page.waitForTimeout(500);

    const jobs: string[] = [];
    const cards = await this.jobList.locator('[data-testid="job-card"]').all();

    for (const card of cards) {
      const id = await card.getAttribute('data-job-id');
      if (id) jobs.push(id);
    }

    return jobs;
  }

  /**
   * Verify queue is displayed correctly
   */
  async verifyQueueDisplay(): Promise<void> {
    const hasJobs = await this.getJobCount() > 0;
    if (hasJobs) {
      await expect(this.jobList).toBeVisible();
    } else {
      await expect(this.noJobsMessage).toBeVisible();
    }
  }

  /**
   * Select job for release
   */
  async selectJob(jobId: string): Promise<void> {
    const card = this.jobCard(jobId);
    const checkbox = card.locator('input[type="checkbox"]');
    await checkbox.check();
  }

  /**
   * Deselect all jobs
   */
  async deselectAllJobs(): Promise<void> {
    const selectAll = this.page.getByRole('checkbox', { name: /select all/i });
    if (await selectAll.isChecked()) {
      await selectAll.click();
    }
  }

  /**
   * Get selected jobs count
   */
  async getSelectedJobsCount(): Promise<number> {
    const selected = await this.jobList.locator('input[type="checkbox"]:checked').all();
    return selected.length;
  }

  /**
   * Verify secure print is enabled
   */
  async isSecurePrintEnabled(): Promise<boolean> {
    return await this.enableSecurePrintToggle.isChecked();
  }

  /**
   * Verify PIN settings
   */
  async verifyPinSettings(expected: {
    length?: number;
    expiry?: number;
  }): Promise<boolean> {
    let valid = true;

    if (expected.length !== undefined) {
      const currentLength = await this.pinLengthSelect.inputValue();
      if (currentLength !== expected.length.toString()) {
        valid = false;
      }
    }

    if (expected.expiry !== undefined) {
      const currentExpiry = await this.pinExpiryInput.inputValue();
      if (currentExpiry !== expected.expiry.toString()) {
        valid = false;
      }
    }

    return valid;
  }

  /**
   * Get job thumbnail/preview
   */
  async getJobThumbnail(jobId: string): Promise<Locator | null> {
    const card = this.jobCard(jobId);
    const thumbnail = card.locator('[data-testid="job-thumbnail"], img.thumbnail');
    return await thumbnail.isVisible() ? thumbnail : null;
  }
}

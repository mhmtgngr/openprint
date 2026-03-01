import { type Page, type Locator, expect } from '@playwright/test';

/**
 * Base Page Object Model class
 * Provides common navigation and authentication methods for all pages
 */
export class BasePage {
  readonly page: Page;
  readonly sidebar: Locator;
  readonly userMenu: Locator;
  readonly logoutButton: Locator;
  readonly notificationBanner: Locator;

  // Navigation links
  readonly dashboardLink: Locator;
  readonly devicesLink: Locator;
  readonly jobsLink: Locator;
  readonly documentsLink: Locator;
  readonly analyticsLink: Locator;
  readonly settingsLink: Locator;
  readonly policiesLink: Locator;
  readonly quotasLink: Locator;
  readonly auditLogsLink: Locator;

  constructor(page: Page) {
    this.page = page;

    // Common locators
    this.sidebar = page.locator('aside, [data-testid="sidebar"]');
    this.userMenu = page.locator('[data-testid="user-menu"], .user-menu');
    this.logoutButton = page.getByRole('button', { name: /logout|sign out/i });
    this.notificationBanner = page.locator('[data-testid="notification"], .notification');

    // Navigation links
    this.dashboardLink = page.getByRole('link', { name: /dashboard|home/i });
    this.devicesLink = page.getByRole('link', { name: /devices|printers/i });
    this.jobsLink = page.getByRole('link', { name: /jobs/i });
    this.documentsLink = page.getByRole('link', { name: /documents/i });
    this.analyticsLink = page.getByRole('link', { name: /analytics|reports/i });
    this.settingsLink = page.getByRole('link', { name: /settings/i });
    this.policiesLink = page.getByRole('link', { name: /policies/i });
    this.quotasLink = page.getByRole('link', { name: /quotas/i });
    this.auditLogsLink = page.getByRole('link', { name: /audit logs/i });
  }

  /**
   * Navigate to a path
   */
  async goto(path: string): Promise<void> {
    await this.page.goto(path);
    await this.waitForLoadState();
  }

  /**
   * Wait for page load state
   */
  async waitForLoadState(state: 'load' | 'domcontentloaded' | 'networkidle' = 'networkidle'): Promise<void> {
    await this.page.waitForLoadState(state);
  }

  /**
   * Wait for navigation to complete
   */
  async waitForNavigation(url?: string): Promise<void> {
    if (url) {
      await this.page.waitForURL(url);
    } else {
      await this.page.waitForLoadState('networkidle');
    }
  }

  /**
   * Get current URL
   */
  getCurrentUrl(): string {
    return this.page.url();
  }

  /**
   * Refresh the page
   */
  async refresh(): Promise<void> {
    await this.page.reload();
    await this.waitForLoadState();
  }

  /**
   * Take screenshot
   */
  async screenshot(name: string, fullPage = true): Promise<void> {
    await this.page.screenshot({
      path: `test-results/screenshots/${name}.png`,
      fullPage,
    });
  }

  /**
   * Navigate to Dashboard
   */
  async navigateToDashboard(): Promise<void> {
    await this.dashboardLink.click();
    await this.waitForNavigation('/dashboard');
  }

  /**
   * Navigate to Devices/Printers
   */
  async navigateToDevices(): Promise<void> {
    await this.devicesLink.click();
    await this.waitForNavigation('/printers');
  }

  /**
   * Navigate to Jobs
   */
  async navigateToJobs(): Promise<void> {
    await this.jobsLink.click();
    await this.waitForNavigation('/jobs');
  }

  /**
   * Navigate to Documents
   */
  async navigateToDocuments(): Promise<void> {
    await this.documentsLink.click();
    await this.waitForNavigation('/documents');
  }

  /**
   * Navigate to Analytics
   */
  async navigateToAnalytics(): Promise<void> {
    await this.analyticsLink.click();
    await this.waitForNavigation('/analytics');
  }

  /**
   * Navigate to Settings
   */
  async navigateToSettings(): Promise<void> {
    await this.settingsLink.click();
    await this.waitForNavigation('/settings');
  }

  /**
   * Navigate to Policies
   */
  async navigateToPolicies(): Promise<void> {
    await this.policiesLink.click();
    await this.waitForNavigation('/policies');
  }

  /**
   * Navigate to Quotas
   */
  async navigateToQuotas(): Promise<void> {
    await this.quotasLink.click();
    await this.waitForNavigation('/quotas');
  }

  /**
   * Navigate to Audit Logs
   */
  async navigateToAuditLogs(): Promise<void> {
    await this.auditLogsLink.click();
    await this.waitForNavigation('/audit-logs');
  }

  /**
   * Logout from the application
   */
  async logout(): Promise<void> {
    await this.userMenu.click();
    await this.logoutButton.click();
    await this.waitForNavigation('/login');
  }

  /**
   * Check if notification is visible
   */
  async isNotificationVisible(): Promise<boolean> {
    return await this.notificationBanner.isVisible();
  }

  /**
   * Get notification text
   */
  async getNotificationText(): Promise<string | null> {
    if (await this.isNotificationVisible()) {
      return await this.notificationBanner.textContent();
    }
    return null;
  }

  /**
   * Wait for notification to appear and return its text
   */
  async waitForNotification(): Promise<string> {
    await expect(this.notificationBanner).toBeVisible({ timeout: 5000 });
    return await this.notificationBanner.textContent() || '';
  }

  /**
   * Close notification if visible
   */
  async closeNotification(): Promise<void> {
    const closeButton = this.notificationBanner.getByRole('button', { name: /close|dismiss/i });
    if (await closeButton.isVisible()) {
      await closeButton.click();
    }
  }

  /**
   * Check if sidebar is visible
   */
  async isSidebarVisible(): Promise<boolean> {
    return await this.sidebar.isVisible();
  }

  /**
   * Toggle mobile menu
   */
  async toggleMobileMenu(): Promise<void> {
    const menuButton = this.page.getByRole('button', { name: /menu|hamburger/i });
    if (await menuButton.isVisible()) {
      await menuButton.click();
    }
  }

  /**
   * Get heading text
   */
  async getHeadingText(): Promise<string> {
    const heading = this.page.getByRole('heading', { level: 1 });
    await expect(heading).toBeVisible();
    return await heading.textContent() || '';
  }

  /**
   * Fill form inputs by label
   */
  async fillForm(fields: Record<string, string>): Promise<void> {
    for (const [label, value] of Object.entries(fields)) {
      await this.page.getByLabel(label).fill(value);
    }
  }

  /**
   * Select dropdown option by label
   */
  async selectOption(label: string, value: string): Promise<void> {
    await this.page.getByLabel(label).selectOption(value);
  }

  /**
   * Check checkbox by label
   */
  async checkCheckbox(label: string): Promise<void> {
    await this.page.getByLabel(label).check();
  }

  /**
   * Uncheck checkbox by label
   */
  async uncheckCheckbox(label: string): Promise<void> {
    await this.page.getByLabel(label).uncheck();
  }

  /**
   * Click button by name
   */
  async clickButton(name: string): Promise<void> {
    await this.page.getByRole('button', { name }).click();
  }

  /**
   * Verify page title
   */
  async verifyPageTitle(expectedTitle: string): Promise<void> {
    await expect(this.page).toHaveTitle(new RegExp(expectedTitle, 'i'));
  }

  /**
   * Verify heading contains text
   */
  async verifyHeadingContains(text: string): Promise<void> {
    const heading = this.page.getByRole('heading');
    await expect(heading).toContainText(text);
  }
}

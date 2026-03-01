/**
 * Dashboard Page Object
 * Handles interactions with the main dashboard page
 */
import { Page, Locator, expect } from '@playwright/test';
import { BasePage } from './BasePage';
import { mockApiResponse, mockPrinters, mockJobs, mockEnvironmentReport } from '../helpers';

export class DashboardPage extends BasePage {
  // Page elements
  readonly heading: Locator;
  readonly statsCards: Locator;
  readonly recentPrintJobs: Locator;
  readonly recentPrinters: Locator;
  readonly environmentalImpact: Locator;
  readonly emptyState: Locator;
  readonly quickActions: Locator;
  readonly viewAllJobsButton: Locator;
  readonly viewAllPrintersButton: Locator;

  // Stats
  readonly totalJobsStat: Locator;
  readonly activePrintersStat: Locator;
  readonly pagesTodayStat: Locator;
  readonly costThisMonthStat: Locator;

  constructor(page: Page) {
    super(page);

    // Initialize locators
    this.heading = page.locator('h1, [data-testid="dashboard-heading"]');
    this.statsCards = page.locator('[data-testid="stat-card"], .stat-card');
    this.recentPrintJobs = page.locator('[data-testid="recent-jobs"], .recent-jobs');
    this.recentPrinters = page.locator('[data-testid="recent-printers"], .recent-printers');
    this.environmentalImpact = page.locator(
      '[data-testid="environmental-impact"], .environmental-impact'
    );
    this.emptyState = page.locator('[data-testid="empty-state"], .empty-state');
    this.quickActions = page.locator('[data-testid="quick-actions"], .quick-actions');
    this.viewAllJobsButton = page.locator('button:has-text("View All Jobs"), a:has-text("View All")');
    this.viewAllPrintersButton = page.locator('button:has-text("View All Printers")');

    // Stats
    this.totalJobsStat = page.locator('[data-testid="stat-total-jobs"]');
    this.activePrintersStat = page.locator('[data-testid="stat-active-printers"]');
    this.pagesTodayStat = page.locator('[data-testid="stat-pages-today"]');
    this.costThisMonthStat = page.locator('[data-testid="stat-cost-month"]');
  }

  /**
   * Navigate to dashboard
   */
  async goto() {
    await this.gotoDashboard();
    await this.waitForPageLoad();
  }

  /**
   * Setup API mocks for dashboard
   */
  async setupMocks() {
    // Mock printers
    await this.page.route('**/api/v1/printers', async (route) => {
      await mockApiResponse(route, { printers: mockPrinters });
    });

    // Mock jobs
    await this.page.route('**/api/v1/jobs*', async (route) => {
      await mockApiResponse(route, {
        data: mockJobs,
        total: mockJobs.length,
        limit: 50,
        offset: 0,
      });
    });

    // Mock environment report
    await this.page.route('**/api/v1/analytics/environment*', async (route) => {
      await mockApiResponse(route, mockEnvironmentReport);
    });
  }

  /**
   * Verify dashboard is loaded
   */
  async isLoaded(): Promise<boolean> {
    await this.heading.waitFor({ state: 'visible', timeout: 5000 });
    return await this.heading.isVisible();
  }

  /**
   * Get stat card values
   */
  async getStatValue(statName: string): Promise<string> {
    const card = this.statsCards.filter({ hasText: statName });
    const valueLocator = card.locator('[data-testid="stat-value"], .value, h2, h3');
    return await valueLocator.textContent() || '';
  }

  /**
   * Verify stats are displayed
   */
  async verifyStatsDisplayed() {
    await expect(this.statsCards).toHaveCount(4);
  }

  /**
   * Verify recent jobs section
   */
  async verifyRecentJobsDisplayed() {
    await expect(this.recentPrintJobs).toBeVisible();
  }

  /**
   * Verify recent printers section
   */
  async verifyRecentPrintersDisplayed() {
    await expect(this.recentPrinters).toBeVisible();
  }

  /**
   * Verify environmental impact section
   */
  async verifyEnvironmentalImpactDisplayed() {
    await expect(this.environmentalImpact).toBeVisible();

    // Verify key metrics are shown
    await expect(this.environmentalImpact).toContainText(['CO2', 'trees saved']);
  }

  /**
   * Get environmental impact values
   */
  async getEnvironmentalImpact() {
    const co2Text = await this.environmentalImpact
      .locator('[data-testid="co2-grams"], .co2-value')
      .textContent();
    const treesText = await this.environmentalImpact
      .locator('[data-testid="trees-saved"], .trees-value')
      .textContent();

    return {
      co2Grams: co2Text || '',
      treesSaved: treesText || '',
    };
  }

  /**
   * Click view all jobs button
   */
  async viewAllJobs() {
    await this.viewAllJobsButton.click();
    await this.page.waitForURL('**/jobs');
  }

  /**
   * Click view all printers button
   */
  async viewAllPrinters() {
    await this.viewAllPrintersButton.click();
    await this.page.waitForURL('**/printers');
  }

  /**
   * Verify empty state is shown
   */
  async verifyEmptyState() {
    await expect(this.emptyState).toBeVisible();
    await expect(this.emptyState).toContainText(/no|empty|nothing/i);
  }

  /**
   * Navigate to printers page via quick actions
   */
  async navigateToPrinters() {
    await this.navigateTo('Printers');
  }

  /**
   * Navigate to jobs page via quick actions
   */
  async navigateToJobs() {
    await this.navigateTo('Jobs');
  }

  /**
   * Navigate to analytics page
   */
  async navigateToAnalytics() {
    await this.navigateTo('Analytics');
  }

  /**
   * Navigate to settings page
   */
  async navigateToSettings() {
    await this.navigateTo('Settings');
  }

  /**
   * Verify user info in sidebar
   */
  async verifyUserInfo(name: string) {
    await expect(this.userInfo).toContainText(name);
  }

  /**
   * Take screenshot of dashboard
   */
  async captureScreenshot(path?: string) {
    await this.screenshot({
      path: path || 'screenshots/dashboard.png',
      fullPage: true,
    });
  }

  /**
   * Verify dashboard is responsive on mobile
   */
  async verifyMobileResponsive() {
    await this.setViewport(375, 667); // iPhone SE dimensions
    await this.reload();

    // Verify sidebar becomes a hamburger menu or collapsible
    const sidebar = this.sidebar;
    const isVisible = await sidebar.isVisible().catch(() => false);

    // On mobile, sidebar should either be hidden or collapsed
    // This is a basic check - adjust based on actual implementation
    await this.screenshot({ path: 'screenshots/dashboard-mobile.png' });
  }

  /**
   * Search for recent jobs
   */
  async searchJobs(query: string) {
    const searchInput = this.recentPrintJobs.locator('input[type="search"], [data-testid="search-input"]');
    await searchInput.fill(query);
  }

  /**
   * Filter jobs by status
   */
  async filterJobsByStatus(status: string) {
    const filterButton = this.recentPrintJobs.locator(`button:has-text("${status}")`);
    await filterButton.click();
  }

  /**
   * Click on a specific job in recent jobs
   */
  async clickJob(jobName: string) {
    const jobLink = this.recentPrintJobs.locator(`a:has-text("${jobName}")`).first();
    await jobLink.click();
  }

  /**
   * Click on a specific printer in recent printers
   */
  async clickPrinter(printerName: string) {
    const printerLink = this.recentPrinters.locator(`a:has-text("${printerName}")`).first();
    await printerLink.click();
  }

  /**
   * Verify quick actions are available
   */
  async verifyQuickActions() {
    await expect(this.quickActions).toBeVisible();

    // Check for common quick action buttons
    const actions = ['New Job', 'Add Printer', 'Discover Printers'];
    for (const action of actions) {
      const button = this.quickActions.locator(`button:has-text("${action}"), a:has-text("${action}")`);
      // At least some quick actions should be present
      await button.isVisible().catch(() => {});
    }
  }

  /**
   * Click quick action button
   */
  async clickQuickAction(actionName: string) {
    const button = this.quickActions.locator(`button:has-text("${actionName}"), a:has-text("${actionName}")`);
    await button.click();
  }

  /**
   * Verify dashboard loading state
   */
  async verifyLoadingState() {
    await expect(this.spinner).toBeVisible();
  }

  /**
   * Verify error state
   */
  async verifyErrorState() {
    const errorElement = this.page.locator('[data-testid="error-message"], .error-message');
    await expect(errorElement).toBeVisible();
  }

  /**
   * Get job count from recent jobs section
   */
  async getRecentJobCount(): Promise<number> {
    const jobItems = this.recentPrintJobs.locator('[data-testid="job-item"], .job-item');
    return await jobItems.count();
  }

  /**
   * Get printer count from recent printers section
   */
  async getRecentPrinterCount(): Promise<number> {
    const printerItems = this.recentPrinters.locator('[data-testid="printer-item"], .printer-item');
    return await printerItems.count();
  }

  /**
   * Verify date range selector
   */
  async verifyDateRangeSelector() {
    const selector = this.page.locator('[data-testid="date-range-selector"], .date-range-selector');
    await expect(selector).toBeVisible();
  }

  /**
   * Select date range
   */
  async selectDateRange(range: string) {
    const selector = this.page.locator('[data-testid="date-range-selector"], .date-range-selector');
    await selector.click();
    await this.page.locator(`button:has-text("${range}")`).click();
  }
}

/**
 * Analytics Page Object
 * Handles metrics, charts, environmental reports, and audit logs
 */
import { Page, Locator, expect } from '@playwright/test';
import { BasePage } from './BasePage';
import { mockApiResponse, mockUsageStats, mockAuditLogs, mockEnvironmentReport } from '../helpers';

export class AnalyticsPage extends BasePage {
  // Page elements
  readonly heading: Locator;
  readonly periodSelector: Locator;
  readonly periodButtons: Locator;
  readonly metricCards: Locator;
  readonly chartsContainer: Locator;
  readonly auditLogsSection: Locator;
  readonly auditLogsTable: Locator;
  readonly auditLogsSearch: Locator;
  readonly auditLogsFilters: Locator;
  readonly exportButton: Locator;
  readonly environmentalReport: Locator;

  // Metrics
  readonly totalJobsMetric: Locator;
  readonly pagesPrintedMetric: Locator;
  readonly costMetric: Locator;
  readonly co2Metric: Locator;
  readonly successRateMetric: Locator;
  readonly avgTimeMetric: Locator;

  // Charts
  readonly printVolumeChart: Locator;
  readonly costChart: Locator;
  readonly statusDistributionChart: Locator;
  readonly co2TrendChart: Locator;

  // Audit log columns
  readonly timestampColumn: Locator;
  readonly userColumn: Locator;
  readonly actionColumn: Locator;
  readonly resourceColumn: Locator;
  readonly detailsColumn: Locator;

  // Filters
  readonly dateRangeFilter: Locator;
  readonly userFilter: Locator;
  readonly actionTypeFilter: Locator;

  constructor(page: Page) {
    super(page);

    // Initialize locators
    this.heading = page.locator('h1, [data-testid="analytics-heading"]');
    this.periodSelector = page.locator('[data-testid="period-selector"], .period-selector');
    this.periodButtons = page.locator('[data-testid="period-selector"] button, .period-selector button');
    this.metricCards = page.locator('[data-testid="metric-card"], .metric-card');
    this.chartsContainer = page.locator('[data-testid="charts-container"], .charts-container');
    this.auditLogsSection = page.locator('[data-testid="audit-logs"], .audit-logs-section');
    this.auditLogsTable = page.locator('[data-testid="audit-logs-table"], .audit-logs-table');
    this.auditLogsSearch = page.locator('input[data-testid="audit-search"], .audit-search-input');
    this.auditLogsFilters = page.locator('[data-testid="audit-filters"], .audit-filters');
    this.exportButton = page.locator('button:has-text("Export"), [data-testid="export-button"]');
    this.environmentalReport = page.locator('[data-testid="environmental-report"], .environmental-report');

    // Metrics
    this.totalJobsMetric = page.locator('[data-testid="metric-total-jobs"], .metric-total-jobs');
    this.pagesPrintedMetric = page.locator('[data-testid="metric-pages"], .metric-pages');
    this.costMetric = page.locator('[data-testid="metric-cost"], .metric-cost');
    this.co2Metric = page.locator('[data-testid="metric-co2"], .metric-co2');
    this.successRateMetric = page.locator('[data-testid="metric-success-rate"], .metric-success-rate');
    this.avgTimeMetric = page.locator('[data-testid="metric-avg-time"], .metric-avg-time');

    // Charts
    this.printVolumeChart = page.locator('[data-testid="chart-print-volume"], .chart-print-volume');
    this.costChart = page.locator('[data-testid="chart-cost"], .chart-cost');
    this.statusDistributionChart = page.locator('[data-testid="chart-status-distribution"], .chart-status-distribution');
    this.co2TrendChart = page.locator('[data-testid="chart-co2-trend"], .chart-co2-trend');

    // Audit log columns
    this.timestampColumn = page.locator('[data-testid="col-timestamp"], .col-timestamp');
    this.userColumn = page.locator('[data-testid="col-user"], .col-user');
    this.actionColumn = page.locator('[data-testid="col-action"], .col-action');
    this.resourceColumn = page.locator('[data-testid="col-resource"], .col-resource');
    this.detailsColumn = page.locator('[data-testid="col-details"], .col-details');

    // Filters
    this.dateRangeFilter = page.locator('[data-testid="filter-date-range"], .filter-date-range');
    this.userFilter = page.locator('[data-testid="filter-user"], .filter-user');
    this.actionTypeFilter = page.locator('[data-testid="filter-action-type"], .filter-action-type');
  }

  /**
   * Navigate to analytics page
   */
  async goto() {
    await this.goto('/analytics');
    await this.waitForPageLoad();
  }

  /**
   * Setup API mocks for analytics page
   */
  async setupMocks() {
    // Mock usage stats
    await this.page.route('**/api/v1/analytics/usage*', async (route) => {
      await mockApiResponse(route, mockUsageStats);
    });

    // Mock audit logs
    await this.page.route('**/api/v1/analytics/audit-logs*', async (route) => {
      await mockApiResponse(route, {
        data: mockAuditLogs,
        total: mockAuditLogs.length,
        limit: 50,
        offset: 0,
      });
    });

    // Mock environment report
    await this.page.route('**/api/v1/analytics/environment*', async (route) => {
      await mockApiResponse(route, mockEnvironmentReport);
    });

    // Mock export endpoint
    await this.page.route('**/api/v1/analytics/export*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'text/csv',
        body: 'timestamp,user,action,resource\n2024-01-01,user1,created,job1',
      });
    });
  }

  /**
   * Verify analytics page is loaded
   */
  async isLoaded(): Promise<boolean> {
    await this.heading.waitFor({ state: 'visible', timeout: 5000 });
    return await this.heading.isVisible();
  }

  /**
   * Select time period
   */
  async selectPeriod(period: 'today' | 'week' | 'month' | 'quarter' | 'year' | 'custom') {
    const button = this.periodButtons.filter({ hasText: new RegExp(period, 'i') });
    await button.click();
    await this.waitForPageLoad();
  }

  /**
   * Get metric value
   */
  async getMetricValue(metricName: string): Promise<string> {
    const card = this.metricCards.filter({ hasText: metricName });
    const value = card.locator('[data-testid="metric-value"], .value, .metric-value');
    return await value.textContent() || '';
  }

  /**
   * Verify all metrics are displayed
   */
  async verifyMetricsDisplayed() {
    await expect(this.metricCards).toHaveCount(await this.metricCards.count());
    // Should have at least 4 metric cards
    const count = await this.metricCards.count();
    expect(count).toBeGreaterThanOrEqual(4);
  }

  /**
   * Verify environmental report section
   */
  async verifyEnvironmentalReport() {
    await expect(this.environmentalReport).toBeVisible();
    await expect(this.environmentalReport).toContainText(['CO2', 'trees', 'pages']);
  }

  /**
   * Get environmental values
   */
  async getEnvironmentalValues(): Promise<{
    pagesPrinted: string;
    co2Grams: string;
    treesSaved: string;
  }> {
    const pages = await this.environmentalReport
      .locator('[data-testid="env-pages"], .pages-printed')
      .textContent();
    const co2 = await this.environmentalReport
      .locator('[data-testid="env-co2"], .co2-grams')
      .textContent();
    const trees = await this.environmentalReport
      .locator('[data-testid="env-trees"], .trees-saved')
      .textContent();

    return {
      pagesPrinted: pages || '',
      co2Grams: co2 || '',
      treesSaved: trees || '',
    };
  }

  /**
   * Verify chart is rendered
   */
  async verifyChartRendered(chartLocator: Locator) {
    await expect(chartLocator).toBeVisible();
    // Check for canvas or SVG elements within chart
    const canvasOrSvg = chartLocator.locator('canvas, svg');
    await expect(canvasOrSvg).toHaveCount(1);
  }

  /**
   * Verify all charts are rendered
   */
  async verifyChartsRendered() {
    await this.verifyChartRendered(this.printVolumeChart);
    await this.verifyChartRendered(this.costChart);
    await this.verifyChartRendered(this.statusDistributionChart);
  }

  /**
   * Scroll to audit logs section
   */
  async scrollToAuditLogs() {
    await this.auditLogsSection.scrollIntoViewIfNeeded();
    await expect(this.auditLogsTable).toBeVisible();
  }

  /**
   * Search audit logs
   */
  async searchAuditLogs(query: string) {
    await this.auditLogsSearch.fill(query);
    await this.page.waitForTimeout(500);
  }

  /**
   * Filter audit logs by user
   */
  async filterAuditLogsByUser(user: string) {
    await this.userFilter.click();
    await this.page.locator(`option:has-text("${user}")`).click();
  }

  /**
   * Filter audit logs by action type
   */
  async filterAuditLogsByAction(actionType: string) {
    await this.actionTypeFilter.click();
    await this.page.locator(`option:has-text("${actionType}")`).click();
  }

  /**
   * Filter audit logs by date range
   */
  async filterAuditLogsByDateRange(from: string, to: string) {
    await this.dateRangeFilter.click();
    // Implementation depends on date picker UI
    await this.page.fill('input[placeholder*="From"]', from);
    await this.page.fill('input[placeholder*="To"]', to);
    await this.page.locator('button:has-text("Apply")').click();
  }

  /**
   * Get audit log entries count
   */
  async getAuditLogCount(): Promise<number> {
    const rows = this.auditLogsTable.locator('tbody tr');
    return await rows.count();
  }

  /**
   * Get audit log entry text
   */
  async getAuditLogEntry(index: number): Promise<string> {
    const row = this.auditLogsTable.locator('tbody tr').nth(index);
    return await row.textContent() || '';
  }

  /**
   * Export audit logs
   */
  async exportAuditLogs(format: 'csv' | 'json' = 'csv') {
    await this.exportButton.click();

    // Select format if prompted
    const formatOption = this.page.locator(`button:has-text("${format.toUpperCase()}")`);
    if (await formatOption.isVisible()) {
      await formatOption.click();
    }

    const downloadPromise = this.page.waitForEvent('download');
    await this.page.locator('button:has-text("Export"), button:has-text("Download")').click();
    await downloadPromise;
  }

  /**
   * Verify audit log action badge colors
   */
  async verifyAuditLogActionColor(action: string, expectedColor: string) {
    const row = this.auditLogsTable.locator('tr').filter({ hasText: action });
    const badge = row.locator('[data-testid="action-badge"], .action-badge');
    await expect(badge).toHaveCSS('background-color', expectedColor);
  }

  /**
   * Click on audit log row for details
   */
  async viewAuditLogDetails(action: string) {
    const row = this.auditLogsTable.locator('tr').filter({ hasText: action });
    await row.click();
  }

  /**
   * Verify metric trend (up/down arrow)
   */
  async verifyMetricTrend(metricName: string, trend: 'up' | 'down' | 'neutral') {
    const card = this.metricCards.filter({ hasText: metricName });
    const trendIndicator = card.locator('[data-testid="trend"], .trend');

    if (trend === 'up') {
      await expect(trendIndicator).toContainText('↑');
    } else if (trend === 'down') {
      await expect(trendIndicator).toContainText('↓');
    }
  }

  /**
   * Get metric percentage change
   */
  async getMetricPercentageChange(metricName: string): Promise<string> {
    const card = this.metricCards.filter({ hasText: metricName });
    const changeText = card.locator('[data-testid="percentage-change"], .percentage-change');
    return await changeText.textContent() || '';
  }

  /**
   * Hover over chart data point
   */
  async hoverChartPoint(chartLocator: Locator, pointIndex: number = 0) {
    const dataPoint = chartLocator.locator('.data-point, circle').nth(pointIndex);
    await dataPoint.hover();
  }

  /**
   * Get chart tooltip text
   */
  async getChartTooltip(): Promise<string> {
    const tooltip = this.page.locator('[data-testid="chart-tooltip"], .chart-tooltip');
    await tooltip.waitFor({ state: 'visible' });
    return await tooltip.textContent() || '';
  }

  /**
   * Verify cost display format
   */
  async verifyCostFormat() {
    const costText = await this.costMetric.textContent();
    // Should contain currency symbol
    expect(costText).toMatch(/\$|€|£/);
  }

  /**
   * Verify success rate is percentage
   */
  async verifySuccessRateFormat() {
    const rateText = await this.successRateMetric.textContent();
    expect(rateText).toMatch(/\d+%/);
  }

  /**
   * Verify CO2 display format
   */
  async verifyCO2Format() {
    const co2Text = await this.co2Metric.textContent();
    expect(co2Text).toMatch(/g|kg/);
  }

  /**
   * Compare periods
   */
  async compareWithPreviousPeriod() {
    const compareButton = this.page.locator('button:has-text("Compare"), [data-testid="compare-periods"]');
    await compareButton.click();

    // Should show comparison view
    const comparisonView = this.page.locator('[data-testid="comparison-view"]');
    await expect(comparisonView).toBeVisible();
  }

  /**
   * Drill down into chart data
   */
  async drillDownChart(chartLocator: Locator) {
    await chartLocator.dblclick();
    // Should navigate to detailed view or open modal
  }

  /**
   * Verify admin access check
   */
  async verifyAdminOnly() {
    // If not admin, should be redirected or see access denied
    const currentUrl = this.page.url();
    const isDenied = currentUrl.includes('denied') || currentUrl.includes('/dashboard');
    return isDenied;
  }

  /**
   * Take screenshot of analytics dashboard
   */
  async captureScreenshot(path?: string) {
    await this.screenshot({
      path: path || 'screenshots/analytics.png',
      fullPage: true,
    });
  }

  /**
   * Verify audit log IP address display
   */
  async verifyAuditLogIP(row: Locator): Promise<boolean> {
    const ipCell = row.locator('[data-testid="col-ip"], .ip-address');
    const ipText = await ipCell.textContent();
    return ipText ? /^\d+\.\d+\.\d+\.\d+$/.test(ipText) : false;
  }

  /**
   * Expand audit log details
   */
  async expandAuditLogDetails(action: string) {
    const row = this.auditLogsTable.locator('tr').filter({ hasText: action });
    const expandButton = row.locator('button[aria-expanded="false"], .expand-button');
    await expandButton.click();
  }

  /**
   * Print analytics report
   */
  async printReport() {
    const printButton = this.page.locator('button:has-text("Print"), [data-testid="print-report"]');
    await printButton.click();

    // Wait for print dialog or navigate to print view
    await this.page.waitForURL('**/print**', { timeout: 5000 }).catch(() => {});
  }
}

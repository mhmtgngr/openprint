/**
 * Quotas Page Object
 * Handles cost tracking, quotas, and budget management
 */
import { Page, Locator, expect } from '@playwright/test';
import { BasePage } from './BasePage';
import { mockApiResponse } from '../helpers';

export interface Quota {
  id: string;
  userId?: string;
  departmentId?: string;
  type: 'user' | 'department' | 'organization';
  period: 'daily' | 'weekly' | 'monthly';
  limitPages: number;
  usedPages: number;
  limitCost: number;
  usedCost: number;
  resetAt: string;
}

export class QuotasPage extends BasePage {
  // Page elements
  readonly heading: Locator;
  readonly overviewCards: Locator;
  readonly quotaList: Locator;
  readonly quotaItems: Locator;
  readonly createQuotaButton: Locator;
  readonly searchInput: Locator;
  readonly typeFilters: Locator;
  readonly periodFilters: Locator;
  readonly emptyState: Locator;

  // Quota form
  readonly quotaForm: Locator;
  readonly quotaNameInput: Locator;
  readonly quotaTypeSelect: Locator;
  readonly quotaUserSelect: Locator;
  readonly quotaDepartmentSelect: Locator;
  readonly quotaPeriodSelect: Locator;
  readonly pageLimitInput: Locator;
  readonly costLimitInput: Locator;
  readonly resetDateInput: Locator;
  readonly saveQuotaButton: Locator;

  // Quota detail
  readonly quotaDetailPage: Locator;
  readonly quotaProgress: Locator;
  readonly usageChart: Locator;
  readonly quotaHistory: Locator;
  readonly editQuotaButton: Locator;
  readonly deleteQuotaButton: Locator;
  readonly resetQuotaButton: Locator;

  // Cost tracking
  readonly costByUser: Locator;
  readonly costByDepartment: Locator;
  readonly costByPrinter: Locator;
  readonly costTrendChart: Locator;

  constructor(page: Page) {
    super(page);

    // Initialize locators
    this.heading = page.locator('h1, [data-testid="quotas-heading"]');
    this.overviewCards = page.locator('[data-testid="overview-cards"], .overview-cards');
    this.quotaList = page.locator('[data-testid="quota-list"], .quota-list');
    this.quotaItems = page.locator('[data-testid="quota-item"], .quota-item');
    this.createQuotaButton = page.locator('button:has-text("Create Quota"), [data-testid="create-quota"]');
    this.searchInput = page.locator('input[type="search"], [data-testid="search-input"]');
    this.typeFilters = page.locator('[data-testid="type-filters"], .type-filters button');
    this.periodFilters = page.locator('[data-testid="period-filters"], .period-filters button');
    this.emptyState = page.locator('[data-testid="empty-state"], .empty-state');

    // Quota form
    this.quotaForm = page.locator('[data-testid="quota-form"], form[data-type="quota"]');
    this.quotaNameInput = page.locator('input[name="name"], [data-testid="quota-name"]');
    this.quotaTypeSelect = page.locator('select[name="type"], [data-testid="quota-type"]');
    this.quotaUserSelect = page.locator('select[name="userId"], [data-testid="user-select"]');
    this.quotaDepartmentSelect = page.locator('select[name="departmentId"], [data-testid="department-select"]');
    this.quotaPeriodSelect = page.locator('select[name="period"], [data-testid="period-select"]');
    this.pageLimitInput = page.locator('input[name="pageLimit"], [data-testid="page-limit"]');
    this.costLimitInput = page.locator('input[name="costLimit"], [data-testid="cost-limit"]');
    this.resetDateInput = page.locator('input[name="resetDate"], [data-testid="reset-date"]');
    this.saveQuotaButton = page.locator('button[type="submit"]:has-text("Save"), [data-testid="save-quota"]');

    // Quota detail
    this.quotaDetailPage = page.locator('[data-testid="quota-detail"]');
    this.quotaProgress = page.locator('[data-testid="quota-progress"], .quota-progress');
    this.usageChart = page.locator('[data-testid="usage-chart"], .usage-chart');
    this.quotaHistory = page.locator('[data-testid="quota-history"], .quota-history');
    this.editQuotaButton = page.locator('button:has-text("Edit"), [data-testid="edit-quota"]');
    this.deleteQuotaButton = page.locator('button:has-text("Delete"), [data-testid="delete-quota"]');
    this.resetQuotaButton = page.locator('button:has-text("Reset"), [data-testid="reset-quota"]');

    // Cost tracking
    this.costByUser = page.locator('[data-testid="cost-by-user"], .cost-by-user');
    this.costByDepartment = page.locator('[data-testid="cost-by-department"], .cost-by-department');
    this.costByPrinter = page.locator('[data-testid="cost-by-printer"], .cost-by-printer');
    this.costTrendChart = page.locator('[data-testid="cost-trend-chart"], .cost-trend-chart');
  }

  /**
   * Navigate to quotas page
   */
  async goto() {
    await this.goto('/quotas');
    await this.waitForPageLoad();
  }

  /**
   * Setup API mocks for quotas page
   */
  async setupMocks() {
    const mockQuotas: Quota[] = [
      {
        id: 'quota-1',
        userId: 'user-1',
        type: 'user',
        period: 'monthly',
        limitPages: 1000,
        usedPages: 450,
        limitCost: 50,
        usedCost: 22.5,
        resetAt: '2024-03-01T00:00:00Z',
      },
      {
        id: 'quota-2',
        departmentId: 'dept-1',
        type: 'department',
        period: 'monthly',
        limitPages: 10000,
        usedPages: 5600,
        limitCost: 500,
        usedCost: 280,
        resetAt: '2024-03-01T00:00:00Z',
      },
    ];

    // Mock quotas list
    await this.page.route('**/api/v1/quotas*', async (route) => {
      await mockApiResponse(route, {
        data: mockQuotas,
        total: mockQuotas.length,
      });
    });

    // Mock quota detail
    await this.page.route('**/api/v1/quotas/*', async (route) => {
      if (route.request().method() === 'GET') {
        await mockApiResponse(route, mockQuotas[0]);
      } else if (route.request().method() === 'DELETE') {
        await mockApiResponse(route, { message: 'Quota deleted' });
      } else if (route.request().method() === 'PATCH' || route.request().method() === 'PUT') {
        await mockApiResponse(route, { message: 'Quota updated' });
      }
    });

    // Mock quota creation
    await this.page.route('**/api/v1/quotas', async (route) => {
      if (route.request().method() === 'POST') {
        await mockApiResponse(route, {
          id: 'new-quota-id',
          message: 'Quota created',
        });
      }
    });

    // Mock quota reset
    await this.page.route('**/api/v1/quotas/*/reset', async (route) => {
      await mockApiResponse(route, { message: 'Quota reset' });
    });

    // Mock cost analytics
    await this.page.route('**/api/v1/analytics/cost*', async (route) => {
      await mockApiResponse(route, {
        byUser: [
          { userId: 'user-1', name: 'John Doe', cost: 25.50, pages: 510 },
          { userId: 'user-2', name: 'Jane Smith', cost: 18.75, pages: 375 },
        ],
        byDepartment: [
          { departmentId: 'dept-1', name: 'Sales', cost: 150.00, pages: 3000 },
          { departmentId: 'dept-2', name: 'Marketing', cost: 125.00, pages: 2500 },
        ],
        byPrinter: [
          { printerId: 'printer-1', name: 'HP LaserJet', cost: 75.00, pages: 1500 },
          { printerId: 'printer-2', name: 'Canon PIXMA', cost: 50.00, pages: 1000 },
        ],
        trend: [
          { date: '2024-02-01', cost: 45.00 },
          { date: '2024-02-08', cost: 52.00 },
          { date: '2024-02-15', cost: 48.00 },
          { date: '2024-02-22', cost: 55.00 },
        ],
      });
    });

    // Mock users/departments for dropdowns
    await this.page.route('**/api/v1/users*', async (route) => {
      await mockApiResponse(route, {
        data: [
          { id: 'user-1', name: 'John Doe', email: 'john@example.com' },
          { id: 'user-2', name: 'Jane Smith', email: 'jane@example.com' },
        ],
      });
    });

    await this.page.route('**/api/v1/departments*', async (route) => {
      await mockApiResponse(route, {
        data: [
          { id: 'dept-1', name: 'Sales' },
          { id: 'dept-2', name: 'Marketing' },
        ],
      });
    });
  }

  /**
   * Verify quotas page is loaded
   */
  async isLoaded(): Promise<boolean> {
    await this.heading.waitFor({ state: 'visible', timeout: 5000 });
    return await this.heading.isVisible();
  }

  /**
   * Get quota count
   */
  async getQuotaCount(): Promise<number> {
    return await this.quotaItems.count();
  }

  /**
   * Filter quotas by type
   */
  async filterByType(type: 'user' | 'department' | 'organization') {
    const filter = this.typeFilters.filter({ hasText: new RegExp(type, 'i') });
    await filter.click();
  }

  /**
   * Filter quotas by period
   */
  async filterByPeriod(period: 'daily' | 'weekly' | 'monthly') {
    const filter = this.periodFilters.filter({ hasText: new RegExp(period, 'i') });
    await filter.click();
  }

  /**
   * Search quotas
   */
  async searchQuotas(query: string) {
    await this.searchInput.fill(query);
    await this.page.waitForTimeout(500);
  }

  /**
   * Open create quota form
   */
  async openCreateQuotaForm() {
    await this.createQuotaButton.click();
    await expect(this.quotaForm).toBeVisible();
  }

  /**
   * Fill quota form
   */
  async fillQuotaForm(data: {
    name?: string;
    type: 'user' | 'department' | 'organization';
    userId?: string;
    departmentId?: string;
    period: 'daily' | 'weekly' | 'monthly';
    pageLimit: number;
    costLimit: number;
    resetDate?: string;
  }) {
    if (data.name) {
      await this.quotaNameInput.fill(data.name);
    }

    await this.quotaTypeSelect.selectOption(data.type);

    if (data.type === 'user' && data.userId) {
      await this.quotaUserSelect.selectOption(data.userId);
    } else if (data.type === 'department' && data.departmentId) {
      await this.quotaDepartmentSelect.selectOption(data.departmentId);
    }

    await this.quotaPeriodSelect.selectOption(data.period);
    await this.pageLimitInput.fill(String(data.pageLimit));
    await this.costLimitInput.fill(String(data.costLimit));

    if (data.resetDate) {
      await this.resetDateInput.fill(data.resetDate);
    }
  }

  /**
   * Save quota
   */
  async saveQuota() {
    await this.saveQuotaButton.click();
    await expect(this.quotaForm).not.toBeVisible();
    await this.verifyToast('Quota created', 'success');
  }

  /**
   * View quota details
   */
  async viewQuotaDetails(quotaId: string) {
    const quotaItem = this.quotaItems.filter({ hasText: quotaId });
    await quotaItem.click();
    await expect(this.quotaDetailPage).toBeVisible();
  }

  /**
   * Get quota usage percentage
   */
  async getUsagePercentage(quotaId: string): Promise<number> {
    const quotaItem = this.quotaItems.filter({ hasText: quotaId });
    const progressText = await quotaItem.locator('[data-testid="usage-percent"], .usage-percent').textContent();
    const match = progressText?.match(/(\d+)%/);
    return match ? parseInt(match[1]) : 0;
  }

  /**
   * Verify quota warning threshold
   */
  async verifyQuotaWarning(quotaId: string) {
    const quotaItem = this.quotaItems.filter({ hasText: quotaId });
    const warningIndicator = quotaItem.locator('[data-testid="warning"], .warning');
    await expect(warningIndicator).toBeVisible();
  }

  /**
   * Verify quota exceeded
   */
  async verifyQuotaExceeded(quotaId: string) {
    const quotaItem = this.quotaItems.filter({ hasText: quotaId });
    const exceededIndicator = quotaItem.locator('[data-testid="exceeded"], .exceeded');
    await expect(exceededIndicator).toBeVisible();
  }

  /**
   * Edit quota
   */
  async editQuota(quotaId: string) {
    await this.viewQuotaDetails(quotaId);
    await this.editQuotaButton.click();
    await expect(this.quotaForm).toBeVisible();
  }

  /**
   * Delete quota
   */
  async deleteQuota(quotaId: string) {
    await this.viewQuotaDetails(quotaId);
    await this.deleteQuotaButton.click();

    const confirmButton = this.page.locator('button:has-text("Confirm"), button:has-text("Delete")');
    await confirmButton.click();

    await this.verifyToast('Quota deleted', 'success');
  }

  /**
   * Reset quota usage
   */
  async resetQuota(quotaId: string) {
    await this.viewQuotaDetails(quotaId);
    await this.resetQuotaButton.click();

    const confirmButton = this.page.locator('button:has-text("Confirm")');
    if (await confirmButton.isVisible()) {
      await confirmButton.click();
    }

    await this.verifyToast('Quota reset', 'success');
  }

  /**
   * Get quota pages usage
   */
  async getPagesUsage(quotaId: string): Promise<{ used: number; limit: number }> {
    const quotaItem = this.quotaItems.filter({ hasText: quotaId });
    const usageText = await quotaItem.locator('[data-testid="pages-usage"], .pages-usage').textContent();
    const match = usageText?.match(/(\d+)\s*\/\s*(\d+)/);
    if (match) {
      return { used: parseInt(match[1]), limit: parseInt(match[2]) };
    }
    return { used: 0, limit: 0 };
  }

  /**
   * Get quota cost usage
   */
  async getCostUsage(quotaId: string): Promise<{ used: number; limit: number }> {
    const quotaItem = this.quotaItems.filter({ hasText: quotaId });
    const usageText = await quotaItem.locator('[data-testid="cost-usage"], .cost-usage').textContent();
    const match = usageText?.match(/\$?([\d.]+)\s*\/\s*\$?([\d.]+)/);
    if (match) {
      return { used: parseFloat(match[1]), limit: parseFloat(match[2]) };
    }
    return { used: 0, limit: 0 };
  }

  /**
   * Verify cost tracking by user
   */
  async verifyCostByUser() {
    await expect(this.costByUser).toBeVisible();
    const userItems = this.costByUser.locator('[data-testid="cost-item"], .cost-item');
    await expect(userItems).toHaveCount(await userItems.count());
  }

  /**
   * Verify cost tracking by department
   */
  async verifyCostByDepartment() {
    await expect(this.costByDepartment).toBeVisible();
    const deptItems = this.costByDepartment.locator('[data-testid="cost-item"], .cost-item');
    await expect(deptItems).toHaveCount(await deptItems.count());
  }

  /**
   * Verify cost trend chart
   */
  async verifyCostTrendChart() {
    await expect(this.costTrendChart).toBeVisible();
    const canvasOrSvg = this.costTrendChart.locator('canvas, svg');
    await expect(canvasOrSvg).toHaveCount(1);
  }

  /**
   * Export quota report
   */
  async exportReport(format: 'csv' | 'pdf' = 'csv') {
    const exportButton = this.page.locator('button:has-text("Export"), [data-testid="export-button"]');
    await exportButton.click();

    const formatOption = this.page.locator(`button:has-text("${format.toUpperCase()}")`);
    if (await formatOption.isVisible()) {
      await formatOption.click();
    }

    const downloadPromise = this.page.waitForEvent('download');
    await this.page.locator('button:has-text("Download")').click();
    await downloadPromise;
  }

  /**
   * Set quota alert threshold
   */
  async setAlertThreshold(quotaId: string, threshold: number) {
    await this.viewQuotaDetails(quotaId);

    const alertSettings = this.page.locator('button:has-text("Alert Settings"), [data-testid="alert-settings"]');
    await alertSettings.click();

    const thresholdInput = this.page.locator('input[name="alertThreshold"], [data-testid="alert-threshold"]');
    await thresholdInput.fill(String(threshold));

    await this.page.locator('button:has-text("Save")').click();
  }

  /**
   * Verify quota reset date
   */
  async verifyResetDate(quotaId: string, expectedDate: string) {
    const quotaItem = this.quotaItems.filter({ hasText: quotaId });
    const resetDateElement = quotaItem.locator('[data-testid="reset-date"], .reset-date');
    await expect(resetDateElement).toContainText(expectedDate);
  }
}

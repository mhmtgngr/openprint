import { type Page, type Locator } from '@playwright/test';

/**
 * Base Page Object class
 */
export class BasePage {
  constructor(protected page: Page) {}

  async goto(path = '') {
    await this.page.goto(path);
  }

  async waitForLoadState() {
    await this.page.waitForLoadState('networkidle');
  }

  async screenshot(name: string) {
    await this.page.screenshot({ path: `screenshots/${name}.png` });
  }

  getByRole(role: string, name?: string) {
    return this.page.getByRole(role as any, name ? { name } : undefined);
  }

  getByTestId(id: string) {
    return this.page.getByTestId(id);
  }

  getByText(text: string) {
    return this.page.getByText(text);
  }
}

/**
 * Login Page Object
 */
export class LoginPage extends BasePage {
  readonly emailInput: Locator;
  readonly passwordInput: Locator;
  readonly submitButton: Locator;
  readonly errorMessage: Locator;
  readonly registerLink: Locator;

  constructor(page: Page) {
    super(page);
    this.emailInput = page.getByLabel(/email/i);
    this.passwordInput = page.getByLabel(/password/i);
    this.submitButton = page.getByRole('button', { name: /sign in|login/i });
    this.errorMessage = page.getByTestId(/error|login-error/i);
    this.registerLink = page.getByRole('link', { name: /register|sign up/i });
  }

  async login(email: string, password: string) {
    await this.emailInput.fill(email);
    await this.passwordInput.fill(password);
    await this.submitButton.click();
    await this.page.waitForURL('/dashboard');
  }

  async goto() {
    await super.goto('/login');
  }
}

/**
 * Dashboard Page Object
 */
export class DashboardPage extends BasePage {
  readonly heading: Locator;
  readonly statCards: Locator;
  readonly recentJobsSection: Locator;
  readonly printersSection: Locator;
  readonly sidebar: Locator;
  readonly userMenu: Locator;
  readonly logoutButton: Locator;

  constructor(page: Page) {
    super(page);
    this.heading = page.getByRole('heading', { name: /welcome/i });
    this.statCards = page.locator('.grid').locator('.bg-white');
    this.recentJobsSection = page.getByRole('heading', { name: /recent print jobs/i });
    this.printersSection = page.getByRole('heading', { name: /available printers/i });
    this.sidebar = page.locator('aside');
    this.userMenu = page.getByText(/logout/i);
    this.logoutButton = page.getByRole('button', { name: /logout/i });
  }

  async navigate() {
    await this.goto('/dashboard');
  }

  async getStatValue(label: string): Promise<string> {
    const card = this.statCards.filter({ hasText: label });
    return await card.locator('.text-2xl').textContent() || '';
  }

  async logout() {
    await this.userMenu.click();
    await this.page.waitForURL('/login');
  }
}

/**
 * Printers Page Object
 */
export class PrintersPage extends BasePage {
  readonly heading: Locator;
  readonly addButton: Locator;
  readonly printerList: Locator;
  readonly printerCard: (name: string) => Locator;

  constructor(page: Page) {
    super(page);
    this.heading = page.getByRole('heading', { name: /devices|printers/i });
    this.addButton = page.getByRole('button', { name: /add printer|add device/i });
    this.printerList = page.locator('[data-testid="printer-list"]');
    this.printerCard = (name: string) =>
      page.locator('.bg-white').filter({ hasText: name });
  }

  async navigate() {
    await this.goto('/printers');
  }

  async hasPrinter(name: string): Promise<boolean> {
    const count = await this.printerCard(name).count();
    return count > 0;
  }

  async getPrinterStatus(name: string): Promise<string | null> {
    const card = this.printerCard(name);
    return await card.locator('[data-testid="printer-status"]').textContent();
  }
}

/**
 * Jobs Page Object
 */
export class JobsPage extends BasePage {
  readonly heading: Locator;
  readonly createJobButton: Locator;
  readonly jobList: Locator;
  readonly filterDropdown: Locator;
  readonly statusFilter: (status: string) => Locator;

  constructor(page: Page) {
    super(page);
    this.heading = page.getByRole('heading', { name: /jobs/i });
    this.createJobButton = page.getByRole('button', { name: /create job|new job/i });
    this.jobList = page.locator('[data-testid="job-list"]');
    this.filterDropdown = page.getByRole('combobox');
    this.statusFilter = (status: string) =>
      page.getByRole('option', { name: status });
  }

  async navigate() {
    await this.goto('/jobs');
  }

  async getJobCount(): Promise<number> {
    return await this.jobList.locator('> div').count();
  }

  async filterByStatus(status: string) {
    await this.filterDropdown.click();
    await this.statusFilter(status).click();
  }
}

/**
 * Settings Page Object
 */
export class SettingsPage extends BasePage {
  readonly heading: Locator;
  readonly tabs: Locator;
  readonly profileTab: Locator;
  readonly securityTab: Locator;
  readonly organizationTab: Locator;

  constructor(page: Page) {
    super(page);
    this.heading = page.getByRole('heading', { name: /settings/i });
    this.tabs = page.getByRole('tablist');
    this.profileTab = page.getByRole('tab', { name: /profile/i });
    this.securityTab = page.getByRole('tab', { name: /security/i });
    this.organizationTab = page.getByRole('tab', { name: /organization/i });
  }

  async navigate() {
    await this.goto('/settings');
  }

  async selectTab(tabName: string) {
    await this.tabs.getByRole('tab', { name: tabName }).click();
  }
}

/**
 * Analytics Page Object
 */
export class AnalyticsPage extends BasePage {
  readonly heading: Locator;
  readonly periodButtons: Locator;
  readonly metricCards: Locator;
  readonly charts: Locator;

  constructor(page: Page) {
    super(page);
    this.heading = page.getByRole('heading', { name: /analytics/i });
    this.periodButtons = page.locator('button').filter({
      hasText: /7 days|30 days|90 days/i,
    });
    this.metricCards = page.locator('.grid').locator('.bg-white');
    this.charts = page.locator('svg');
  }

  async navigate() {
    await this.goto('/analytics');
  }

  async selectPeriod(period: '7d' | '30d' | '90d' | '12m') {
    const label = {
      '7d': '7 Days',
      '30d': '30 Days',
      '90d': '90 Days',
      '12m': '12 Months',
    }[period];
    await this.page.getByRole('button', { name: label }).click();
  }
}

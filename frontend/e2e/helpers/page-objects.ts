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

/**
 * Agents Page Object
 */
export class AgentsPage extends BasePage {
  readonly heading: Locator;
  readonly addButton: Locator;
  readonly agentList: Locator;
  readonly statusFilter: (status: string) => Locator;

  constructor(page: Page) {
    super(page);
    this.heading = page.getByRole('heading', { name: /agents/i });
    this.addButton = page.getByRole('button', { name: /add agent/i });
    this.agentList = page.locator('[data-testid="agent-list"]');
    this.statusFilter = (status: string) =>
      page.getByRole('button', { name: status });
  }

  async navigate() {
    await this.goto('/agents');
  }

  async filterByStatus(status: string) {
    await this.statusFilter(status).click();
  }
}

/**
 * Documents Page Object
 */
export class DocumentsPage extends BasePage {
  readonly heading: Locator;
  readonly uploadButton: Locator;
  readonly documentList: Locator;
  readonly searchInput: Locator;

  constructor(page: Page) {
    super(page);
    this.heading = page.getByRole('heading', { name: /documents/i });
    this.uploadButton = page.getByRole('button', { name: /upload/i });
    this.documentList = page.locator('[data-testid="document-list"]');
    this.searchInput = page.getByPlaceholder(/search/i);
  }

  async navigate() {
    await this.goto('/documents');
  }

  async search(query: string) {
    await this.searchInput.fill(query);
  }
}

/**
 * Quotas Page Object
 */
export class QuotasPage extends BasePage {
  readonly heading: Locator;
  readonly editButton: Locator;
  readonly quotaTable: Locator;

  constructor(page: Page) {
    super(page);
    this.heading = page.getByRole('heading', { name: /quotas/i });
    this.editButton = page.getByRole('button', { name: /edit/i }).first();
    this.quotaTable = page.locator('table');
  }

  async navigate() {
    await this.goto('/quotas');
  }
}

/**
 * Policies Page Object
 */
export class PoliciesPage extends BasePage {
  readonly heading: Locator;
  readonly createButton: Locator;
  readonly policyList: Locator;

  constructor(page: Page) {
    super(page);
    this.heading = page.getByRole('heading', { name: /policies/i });
    this.createButton = page.getByRole('button', { name: /create policy/i });
    this.policyList = page.locator('[data-testid="policy-list"]');
  }

  async navigate() {
    await this.goto('/policies');
  }

  async openCreateModal() {
    await this.createButton.click();
  }
}

/**
 * AuditLogs Page Object
 */
export class AuditLogsPage extends BasePage {
  readonly heading: Locator;
  readonly exportButton: Locator;
  readonly logsTable: Locator;
  readonly actionFilter: Locator;
  readonly userFilter: Locator;
  readonly searchInput: Locator;

  constructor(page: Page) {
    super(page);
    this.heading = page.getByRole('heading', { name: /audit logs/i });
    this.exportButton = page.getByRole('button', { name: /export/i });
    this.logsTable = page.locator('table');
    this.actionFilter = page.getByLabel(/action/i);
    this.userFilter = page.getByLabel(/user/i);
    this.searchInput = page.getByPlaceholder(/search/i);
  }

  async navigate() {
    await this.goto('/audit-logs');
  }

  async filterByAction(action: string) {
    await this.actionFilter.selectOption(action);
  }
}

/**
 * EmailToPrint Page Object
 */
export class EmailToPrintPage extends BasePage {
  readonly heading: Locator;
  readonly enableToggle: Locator;
  readonly domainInput: Locator;

  constructor(page: Page) {
    super(page);
    this.heading = page.getByRole('heading', { name: /email-to-print/i });
    this.enableToggle = page.getByRole('switch', { name: /enable/i });
    this.domainInput = page.getByLabel(/email domain/i);
  }

  async navigate() {
    await this.goto('/email-to-print');
  }
}

/**
 * PrintRelease Page Object
 */
export class PrintReleasePage extends BasePage {
  readonly heading: Locator;
  readonly releaseButton: Locator;
  readonly cancelButton: Locator;
  readonly jobList: Locator;

  constructor(page: Page) {
    super(page);
    this.heading = page.getByRole('heading', { name: /print release/i });
    this.releaseButton = page.getByRole('button', { name: /release/i }).first();
    this.cancelButton = page.getByRole('button', { name: /cancel/i }).first();
    this.jobList = page.locator('[data-testid="release-job-list"]');
  }

  async navigate() {
    await this.goto('/print-release');
  }
}

/**
 * Metrics Dashboard Page Object
 */
export class MetricsDashboardPage extends BasePage {
  readonly heading: Locator;
  readonly timeRangeButtons: Locator;
  readonly serviceButtons: Locator;
  readonly autoRefreshButton: Locator;
  readonly metricCards: Locator;
  readonly requestRateChart: Locator;
  readonly errorRateChart: Locator;
  readonly latencyChart: Locator;
  readonly serviceHealthGrid: Locator;

  constructor(page: Page) {
    super(page);
    this.heading = page.getByRole('heading', { name: /metrics dashboard/i });
    this.timeRangeButtons = page.locator('button').filter({
      hasText: /5 min|15 min|30 min|1 hour|3 hours|6 hours|12 hours|24 hours/i,
    });
    this.serviceButtons = page.locator('button').filter({
      hasText: /all services|auth-service|registry-service|job-service|storage-service|notification-service/i,
    });
    this.autoRefreshButton = page.locator('button').filter({ hasText: /auto-refresh/i }).or(
      page.locator('button[title*="Auto-refresh"]')
    );
    this.metricCards = page.locator('.grid').locator('.rounded-xl');
    this.requestRateChart = page.locator('.bg-white').filter({ hasText: /request rate/i });
    this.errorRateChart = page.locator('.bg-white').filter({ hasText: /error rate/i });
    this.latencyChart = page.locator('.bg-white').filter({ hasText: /p95 latency/i });
    this.serviceHealthGrid = page.locator('.bg-white').filter({ hasText: /service health/i });
  }

  async navigate() {
    await this.goto('/metrics');
  }

  async selectTimeRange(range: string) {
    await this.page.getByRole('button', { name: range }).click();
  }

  async selectService(service: string) {
    await this.page.getByRole('button', { name: service }).click();
  }

  async toggleAutoRefresh() {
    await this.autoRefreshButton.click();
  }

  async getServiceHealthStatus(serviceName: string): Promise<string | null> {
    const serviceCard = this.serviceHealthGrid.locator('.bg-white').filter({ hasText: serviceName });
    const statusElement = serviceCard.locator('[class*="rounded"]').filter({ hasText: /healthy|degraded|unhealthy/i });
    return await statusElement.textContent();
  }
}

/**
 * Monitoring Page Object
 */
export class MonitoringPage extends BasePage {
  readonly heading: Locator;
  readonly tabs: Locator;
  readonly alertsTab: Locator;
  readonly servicesTab: Locator;
  readonly silencesTab: Locator;
  readonly autoRefreshButton: Locator;
  readonly refreshButton: Locator;
  readonly summaryCards: Locator;
  readonly alertPanel: Locator;
  readonly serviceHealthList: Locator;
  readonly silencesList: Locator;

  constructor(page: Page) {
    super(page);
    this.heading = page.getByRole('heading', { name: /monitoring/i });
    this.tabs = page.locator('nav').locator('button');
    this.alertsTab = page.getByRole('button', { name: /alerts/i });
    this.servicesTab = page.getByRole('button', { name: /services/i });
    this.silencesTab = page.getByRole('button', { name: /silences/i });
    this.autoRefreshButton = page.locator('button').filter({ hasText: /auto-refresh/i }).or(
      page.locator('button[title*="Auto-refresh"]')
    );
    this.refreshButton = page.getByRole('button', { name: /refresh/i });
    this.summaryCards = page.locator('.grid').locator('.rounded-xl');
    this.alertPanel = page.locator('.bg-white').filter({ hasText: /alerts/i });
    this.serviceHealthList = page.locator('.space-y-6');
    this.silencesList = page.locator('.space-y-4');
  }

  async navigate() {
    await this.goto('/monitoring');
  }

  async selectTab(tab: 'alerts' | 'services' | 'silences') {
    await this.page.getByRole('button', { name: new RegExp(tab, 'i') }).click();
  }

  async toggleAutoRefresh() {
    await this.autoRefreshButton.click();
  }

  async refresh() {
    await this.refreshButton.click();
  }

  async getFiringAlertsCount(): Promise<number> {
    const card = this.summaryCards.nth(1);
    const text = await card.textContent() || '';
    const match = text.match(/\d+/);
    return match ? parseInt(match[0], 10) : 0;
  }
}

/**
 * ObservabilityHub Page Object (Tracing)
 */
export class ObservabilityHubPage extends BasePage {
  readonly heading: Locator;
  readonly tabs: Locator;
  readonly searchTab: Locator;
  readonly traceTab: Locator;
  readonly summaryTab: Locator;
  readonly timeRangeButtons: Locator;
  readonly serviceFilter: Locator;
  readonly openJaegerButton: Locator;
  readonly openGrafanaButton: Locator;
  readonly autoRefreshButton: Locator;
  readonly summaryCards: Locator;
  readonly traceSearch: Locator;
  readonly traceViewer: Locator;

  constructor(page: Page) {
    super(page);
    this.heading = page.getByRole('heading', { name: /observability hub/i });
    this.tabs = page.locator('nav').locator('button');
    this.searchTab = page.getByRole('button', { name: /search traces/i });
    this.traceTab = page.getByRole('button', { name: /view trace/i });
    this.summaryTab = page.getByRole('button', { name: /summary/i });
    this.timeRangeButtons = page.locator('button').filter({
      hasText: /1 hour|3 hours|6 hours|12 hours|24 hours|7 days/i,
    });
    this.serviceFilter = page.locator('select').or(page.getByLabel(/service/i));
    this.openJaegerButton = page.getByRole('link', { name: /open jaeger/i });
    this.openGrafanaButton = page.getByRole('link', { name: /open grafana/i });
    this.autoRefreshButton = page.locator('button').filter({ hasText: /auto-refresh/i }).or(
      page.locator('button[title*="Auto-refresh"]')
    );
    this.summaryCards = page.locator('.grid').locator('.rounded-xl');
    this.traceSearch = page.locator('.bg-white').filter({ hasText: /search traces/i });
    this.traceViewer = page.locator('.bg-white').filter({ hasText: /trace spans/i });
  }

  async navigate() {
    await this.goto('/observability');
  }

  async selectTab(tab: 'search' | 'trace' | 'summary') {
    await this.page.getByRole('button', { name: new RegExp(tab, 'i') }).click();
  }

  async selectService(service: string) {
    await this.serviceFilter.selectOption(service);
  }

  async selectTimeRange(range: string) {
    await this.page.getByRole('button', { name: range }).click();
  }

  async toggleAutoRefresh() {
    await this.autoRefreshButton.click();
  }

  async searchTrace(query: string) {
    await this.page.getByPlaceholder(/search by operation/i).fill(query);
    await this.page.getByRole('button', { name: /search/i }).click();
  }
}

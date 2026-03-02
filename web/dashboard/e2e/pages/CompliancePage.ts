/**
 * Compliance Page Object
 * Handles FedRAMP and HIPAA compliance reports and audit log exports
 */
import { Page, Locator, expect } from '@playwright/test';
import { BasePage } from './BasePage';
import { mockApiResponse } from '../helpers';

export interface ComplianceReport {
  id: string;
  type: 'fedramp' | 'hipaa' | 'gdpr' | 'soc2';
  title: string;
  description: string;
  status: 'compliant' | 'non_compliant' | 'pending_review';
  lastGenerated: string;
  nextDue: string;
  generatedBy: string;
}

export interface ComplianceCheck {
  id: string;
  category: string;
  control: string;
  status: 'pass' | 'fail' | 'warning';
  description: string;
  evidence?: string;
  lastChecked: string;
}

export class CompliancePage extends BasePage {
  // Page elements
  readonly heading: Locator;
  readonly overviewCards: Locator;
  readonly complianceStatus: Locator;
  readonly lastAuditDate: Locator;
  readonly nextAuditDate: Locator;

  // Reports section
  readonly reportsList: Locator;
  readonly reportItems: Locator;
  readonly generateReportButton: Locator;
  readonly searchReportsInput: Locator;
  readonly typeFilters: Locator;

  // Report detail
  readonly reportDetailPage: Locator;
  readonly reportContent: Locator;
  readonly reportStatus: Locator;
  readonly downloadReportButton: Locator;
  readonly shareReportButton: Locator;
  readonly scheduleReportButton: Locator;

  // Checks section
  readonly checksList: Locator;
  readonly checkItems: Locator;
  readonly runChecksButton: Locator;
  readonly checkCategories: Locator;

  // Check detail
  readonly checkDetailModal: Locator;
  readonly checkDescription: Locator;
  readonly checkEvidence: Locator;
  readonly checkRemediation: Locator;
  readonly markResolvedButton: Locator;

  // Audit log export
  readonly exportAuditLogsButton: Locator;
  readonly exportModal: Locator;
  readonly exportDateRangeFrom: Locator;
  readonly exportDateRangeTo: Locator;
  readonly exportFormatSelect: Locator;
  readonly exportIncludePIIToggle: Locator;
  readonly confirmExportButton: Locator;

  // Data retention
  readonly retentionSettings: Locator;
  readonly retentionPeriodInput: Locator;
  readonly autoDeleteToggle: Locator;
  readonly saveRetentionButton: Locator;

  // Access logs
  readonly accessLogs: Locator;
  readonly accessLogItems: Locator;
  readonly accessLogFilters: Locator;

  // Security settings
  readonly encryptionStatus: Locator;
  readonly keyRotationButton: Locator;
  readonly twoFactorRequired: Locator;
  readonly ipWhitelistButton: Locator;

  constructor(page: Page) {
    super(page);

    // Initialize locators
    this.heading = page.locator('h1, [data-testid="compliance-heading"]');
    this.overviewCards = page.locator('[data-testid="overview-cards"], .overview-cards');
    this.complianceStatus = page.locator('[data-testid="compliance-status"], .compliance-status');
    this.lastAuditDate = page.locator('[data-testid="last-audit"], .last-audit-date');
    this.nextAuditDate = page.locator('[data-testid="next-audit"], .next-audit-date');

    // Reports
    this.reportsList = page.locator('[data-testid="reports-list"], .reports-list');
    this.reportItems = page.locator('[data-testid="report-item"], .report-item');
    this.generateReportButton = page.locator('button:has-text("Generate Report"), [data-testid="generate-report"]');
    this.searchReportsInput = page.locator('input[type="search"], [data-testid="search-input"]');
    this.typeFilters = page.locator('[data-testid="type-filters"], .type-filters button');

    // Report detail
    this.reportDetailPage = page.locator('[data-testid="report-detail"]');
    this.reportContent = page.locator('[data-testid="report-content"], .report-content');
    this.reportStatus = page.locator('[data-testid="report-status"], .report-status');
    this.downloadReportButton = page.locator('button:has-text("Download"), [data-testid="download-report"]');
    this.shareReportButton = page.locator('button:has-text("Share"), [data-testid="share-report"]');
    this.scheduleReportButton = page.locator('button:has-text("Schedule"), [data-testid="schedule-report"]');

    // Checks
    this.checksList = page.locator('[data-testid="checks-list"], .checks-list');
    this.checkItems = page.locator('[data-testid="check-item"], .check-item');
    this.runChecksButton = page.locator('button:has-text("Run Checks"), [data-testid="run-checks"]');
    this.checkCategories = page.locator('[data-testid="check-categories"], .check-categories');

    // Check detail
    this.checkDetailModal = page.locator('[data-testid="check-detail-modal"], .check-detail-modal');
    this.checkDescription = page.locator('[data-testid="check-description"], .check-description');
    this.checkEvidence = page.locator('[data-testid="check-evidence"], .check-evidence');
    this.checkRemediation = page.locator('[data-testid="check-remediation"], .check-remediation');
    this.markResolvedButton = page.locator('button:has-text("Mark Resolved"), [data-testid="mark-resolved"]');

    // Audit log export
    this.exportAuditLogsButton = page.locator('button:has-text("Export Logs"), [data-testid="export-logs"]');
    this.exportModal = page.locator('[data-testid="export-modal"], .export-modal');
    this.exportDateRangeFrom = page.locator('input[name="from"], [data-testid="export-from"]');
    this.exportDateRangeTo = page.locator('input[name="to"], [data-testid="export-to"]');
    this.exportFormatSelect = page.locator('select[name="format"], [data-testid="export-format"]');
    this.exportIncludePIIToggle = page.locator('input[name="includePII"], [data-testid="include-pii"]');
    this.confirmExportButton = page.locator('button:has-text("Export"), [data-testid="confirm-export"]');

    // Data retention
    this.retentionSettings = page.locator('[data-testid="retention-settings"], .retention-settings');
    this.retentionPeriodInput = page.locator('input[name="retentionPeriod"], [data-testid="retention-period"]');
    this.autoDeleteToggle = page.locator('input[name="autoDelete"], [data-testid="auto-delete"]');
    this.saveRetentionButton = page.locator('button:has-text("Save"), [data-testid="save-retention"]');

    // Access logs
    this.accessLogs = page.locator('[data-testid="access-logs"], .access-logs');
    this.accessLogItems = page.locator('[data-testid="access-log-item"], .access-log-item');
    this.accessLogFilters = page.locator('[data-testid="access-log-filters"], .access-log-filters');

    // Security settings
    this.encryptionStatus = page.locator('[data-testid="encryption-status"], .encryption-status');
    this.keyRotationButton = page.locator('button:has-text("Rotate Keys"), [data-testid="rotate-keys"]');
    this.twoFactorRequired = page.locator('[data-testid="2fa-status"], .2fa-status');
    this.ipWhitelistButton = page.locator('button:has-text("IP Whitelist"), [data-testid="ip-whitelist"]');
  }

  /**
   * Navigate to compliance page
   */
  async goto() {
    await this.goto('/compliance');
    await this.waitForPageLoad();
  }

  /**
   * Setup API mocks for compliance page
   */
  async setupMocks() {
    // Mock compliance reports
    await this.page.route('**/api/v1/compliance/reports*', async (route) => {
      const mockReports: ComplianceReport[] = [
        {
          id: 'report-1',
          type: 'fedramp',
          title: 'FedRAMP Moderate Assessment',
          description: 'Annual FedRAMP compliance assessment',
          status: 'compliant',
          lastGenerated: '2024-01-15T00:00:00Z',
          nextDue: '2025-01-15T00:00:00Z',
          generatedBy: 'admin@example.com',
        },
        {
          id: 'report-2',
          type: 'hipaa',
          title: 'HIPAA Security Rule Assessment',
          description: 'HIPAA security rule compliance review',
          status: 'compliant',
          lastGenerated: '2024-02-01T00:00:00Z',
          nextDue: '2024-08-01T00:00:00Z',
          generatedBy: 'compliance@example.com',
        },
      ];
      await mockApiResponse(route, {
        data: mockReports,
        total: mockReports.length,
      });
    });

    // Mock compliance checks
    await this.page.route('**/api/v1/compliance/checks*', async (route) => {
      const mockChecks: ComplianceCheck[] = [
        {
          id: 'check-1',
          category: 'Access Control',
          control: 'AC-1: Access Control Policy',
          status: 'pass',
          description: 'System has documented access control policies',
          lastChecked: '2024-02-27T00:00:00Z',
        },
        {
          id: 'check-2',
          category: 'Encryption',
          control: 'SC-13: Cryptographic Protection',
          status: 'pass',
          description: 'Data is encrypted at rest and in transit',
          lastChecked: '2024-02-27T00:00:00Z',
        },
        {
          id: 'check-3',
          category: 'Audit Logging',
          control: 'AU-2: Audit Events',
          status: 'warning',
          description: 'Some audit log retention periods below recommended',
          lastChecked: '2024-02-27T00:00:00Z',
        },
      ];
      await mockApiResponse(route, {
        data: mockChecks,
        total: mockChecks.length,
      });
    });

    // Mock report generation
    await this.page.route('**/api/v1/compliance/reports/generate', async (route) => {
      await mockApiResponse(route, {
        reportId: 'new-report-id',
        message: 'Report generation started',
      });
    });

    // Mock report download
    await this.page.route('**/api/v1/compliance/reports/*/download', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/pdf',
        body: 'Mock PDF content',
      });
    });

    // Mock audit log export
    await this.page.route('**/api/v1/compliance/audit-logs/export', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'text/csv',
        body: 'timestamp,user,action,resource\n2024-01-01,user1,login,system',
      });
    });

    // Mock data retention settings
    await this.page.route('**/api/v1/compliance/retention', async (route) => {
      if (route.request().method() === 'GET') {
        await mockApiResponse(route, {
          retentionPeriod: 2555, // 7 years in days
          autoDelete: true,
        });
      } else {
        await mockApiResponse(route, {
          message: 'Retention settings updated',
        });
      }
    });

    // Mock security settings
    await this.page.route('**/api/v1/compliance/security*', async (route) => {
      if (route.request().method() === 'GET') {
        await mockApiResponse(route, {
          encryptionEnabled: true,
          encryptionLevel: 'AES-256',
          keyRotationRequired: false,
          twoFactorRequired: true,
          ipWhitelist: ['192.168.1.0/24'],
        });
      } else if (route.request().method() === 'POST') {
        await mockApiResponse(route, {
          message: 'Security operation completed',
        });
      }
    });

    // Mock run compliance checks
    await this.page.route('**/api/v1/compliance/checks/run', async (route) => {
      await mockApiResponse(route, {
        message: 'Compliance checks initiated',
        checkId: 'run-1',
      });
    });
  }

  /**
   * Verify compliance page is loaded
   */
  async isLoaded(): Promise<boolean> {
    await this.heading.waitFor({ state: 'visible', timeout: 5000 });
    return await this.heading.isVisible();
  }

  /**
   * Get overall compliance status
   */
  async getComplianceStatus(): Promise<string> {
    const statusText = await this.complianceStatus.textContent();
    return statusText?.toLowerCase() || 'unknown';
  }

  /**
   * Verify compliance status
   */
  async verifyComplianceStatus(status: 'compliant' | 'non_compliant' | 'pending_review') {
    const statusClass = await this.complianceStatus.getAttribute('data-status');
    expect(statusClass).toBe(status);
  }

  /**
   * Get report count
   */
  async getReportCount(): Promise<number> {
    return await this.reportItems.count();
  }

  /**
   * Filter reports by type
   */
  async filterReportsByType(type: 'fedramp' | 'hipaa' | 'gdpr' | 'soc2') {
    const filter = this.typeFilters.filter({ hasText: new RegExp(type, 'i') });
    await filter.click();
  }

  /**
   * Search reports
   */
  async searchReports(query: string) {
    await this.searchReportsInput.fill(query);
    await this.page.waitForTimeout(500);
  }

  /**
   * Generate new compliance report
   */
  async generateReport(type: 'fedramp' | 'hipaa' | 'gdpr' | 'soc2') {
    await this.generateReportButton.click();

    // Select report type
    const typeSelect = this.page.locator('select[name="reportType"], [data-testid="report-type"]');
    await typeSelect.selectOption(type);

    await this.page.locator('button:has-text("Generate"), button[type="submit"]').click();
    await this.verifyToast('Report generation started', 'info');
  }

  /**
   * View report details
   */
  async viewReportDetails(reportId: string) {
    const reportItem = this.reportItems.filter({ hasText: reportId });
    await reportItem.click();
    await expect(this.reportDetailPage).toBeVisible();
  }

  /**
   * Download report
   */
  async downloadReport(reportId: string) {
    await this.viewReportDetails(reportId);

    const downloadPromise = this.page.waitForEvent('download');
    await this.downloadReportButton.click();
    await downloadPromise;
  }

  /**
   * Share report
   */
  async shareReport(reportId: string, email: string) {
    await this.viewReportDetails(reportId);
    await this.shareReportButton.click();

    const emailInput = this.page.locator('input[name="email"], [data-testid="share-email"]');
    await emailInput.fill(email);

    await this.page.locator('button:has-text("Share"), button[type="submit"]').click();
    await this.verifyToast('Report shared', 'success');
  }

  /**
   * Schedule report generation
   */
  async scheduleReport(type: string, frequency: 'daily' | 'weekly' | 'monthly', recipients: string[]) {
    await this.scheduleReportButton.click();

    const typeSelect = this.page.locator('select[name="reportType"]');
    await typeSelect.selectOption(type);

    const frequencySelect = this.page.locator('select[name="frequency"]');
    await frequencySelect.selectOption(frequency);

    const recipientsInput = this.page.locator('input[name="recipients"]');
    await recipientsInput.fill(recipients.join(', '));

    await this.page.locator('button:has-text("Schedule"), button[type="submit"]').click();
    await this.verifyToast('Report scheduled', 'success');
  }

  /**
   * Get compliance check count
   */
  async getCheckCount(): Promise<number> {
    return await this.checkItems.count();
  }

  /**
   * Filter checks by category
   */
  async filterChecksByCategory(category: string) {
    const categoryButton = this.checkCategories.locator(`button:has-text("${category}")`);
    await categoryButton.click();
  }

  /**
   * Run compliance checks
   */
  async runComplianceChecks() {
    await this.runChecksButton.click();
    await this.verifyToast('Compliance checks initiated', 'info');
  }

  /**
   * View check details
   */
  async viewCheckDetails(checkId: string) {
    const checkItem = this.checkItems.filter({ hasText: checkId });
    await checkItem.click();
    await expect(this.checkDetailModal).toBeVisible();
  }

  /**
   * Get check status
   */
  async getCheckStatus(checkId: string): Promise<'pass' | 'fail' | 'warning'> {
    const checkItem = this.checkItems.filter({ hasText: checkId });
    const statusBadge = checkItem.locator('[data-testid="check-status"], .status-badge');
    const statusText = await statusBadge.textContent();
    return (statusText?.toLowerCase() as 'pass' | 'fail' | 'warning') || 'pass';
  }

  /**
   * Mark check as resolved
   */
  async markCheckResolved(checkId: string) {
    await this.viewCheckDetails(checkId);
    await this.markResolvedButton.click();

    const confirmButton = this.page.locator('button:has-text("Confirm")');
    if (await confirmButton.isVisible()) {
      await confirmButton.click();
    }

    await this.verifyToast('Check marked as resolved', 'success');
  }

  /**
   * Export audit logs
   */
  async exportAuditLogs(options: {
    from: string;
    to: string;
    format?: 'csv' | 'json' | 'pdf';
    includePII?: boolean;
  }) {
    await this.exportAuditLogsButton.click();
    await expect(this.exportModal).toBeVisible();

    await this.exportDateRangeFrom.fill(options.from);
    await this.exportDateRangeTo.fill(options.to);

    if (options.format) {
      await this.exportFormatSelect.selectOption(options.format);
    }

    if (options.includePII !== undefined) {
      if (options.includePII) {
        await this.exportIncludePIIToggle.check();
      } else {
        await this.exportIncludePIIToggle.uncheck();
      }
    }

    const downloadPromise = this.page.waitForEvent('download');
    await this.confirmExportButton.click();
    await downloadPromise;
  }

  /**
   * Update data retention settings
   */
  async updateRetentionSettings(period: number, autoDelete: boolean) {
    await this.retentionPeriodInput.fill(String(period));

    if (autoDelete) {
      await this.autoDeleteToggle.check();
    } else {
      await this.autoDeleteToggle.uncheck();
    }

    await this.saveRetentionButton.click();
    await this.verifyToast('Retention settings saved', 'success');
  }

  /**
   * Get retention period
   */
  async getRetentionPeriod(): Promise<number> {
    const value = await this.retentionPeriodInput.inputValue();
    return parseInt(value) || 0;
  }

  /**
   * Verify encryption status
   */
  async verifyEncryptionStatus(expectedStatus: 'enabled' | 'disabled') {
    const statusClass = await this.encryptionStatus.getAttribute('data-status');
    expect(statusClass).toBe(expectedStatus);
  }

  /**
   * Rotate encryption keys
   */
  async rotateKeys() {
    await this.keyRotationButton.click();

    const confirmButton = this.page.locator('button:has-text("Confirm")');
    await confirmButton.click();

    await this.verifyToast('Key rotation initiated', 'info');
  }

  /**
   * Verify 2FA status
   */
  async verify2FARequired(required: boolean) {
    const statusText = await this.twoFactorRequired.textContent();
    const isRequired = statusText?.toLowerCase().includes('required') || false;
    expect(isRequired).toBe(required);
  }

  /**
   * Manage IP whitelist
   */
  async manageIPWhelist() {
    await this.ipWhitelistButton.click();

    const whitelistModal = this.page.locator('[data-testid="ip-whitelist-modal"]');
    await expect(whitelistModal).toBeVisible();
  }

  /**
   * Add IP to whitelist
   */
  async addIPToWhitelist(ip: string) {
    await this.manageIPWhitelist();

    const ipInput = this.page.locator('input[name="ip"], [data-testid="ip-input"]');
    await ipInput.fill(ip);

    await this.page.locator('button:has-text("Add"), button[type="submit"]').click();
    await this.verifyToast('IP added to whitelist', 'success');
  }

  /**
   * Get access log count
   */
  async getAccessLogCount(): Promise<number> {
    return await this.accessLogItems.count();
  }

  /**
   * Filter access logs
   */
  async filterAccessLogs(filters: {
    user?: string;
    action?: string;
    dateFrom?: string;
    dateTo?: string;
  }) {
    if (filters.user) {
      const userFilter = this.accessLogFilters.locator('input[name="user"]');
      await userFilter.fill(filters.user);
    }

    if (filters.action) {
      const actionFilter = this.accessLogFilters.locator('select[name="action"]');
      await actionFilter.selectOption(filters.action);
    }

    if (filters.dateFrom) {
      const fromInput = this.accessLogFilters.locator('input[name="from"]');
      await fromInput.fill(filters.dateFrom);
    }

    if (filters.dateTo) {
      const toInput = this.accessLogFilters.locator('input[name="to"]');
      await toInput.fill(filters.dateTo);
    }

    const applyButton = this.accessLogFilters.locator('button:has-text("Apply"), button[type="submit"]');
    await applyButton.click();
  }

  /**
   * Verify report includes required sections
   */
  async verifyReportSections(reportId: string, sections: string[]) {
    await this.viewReportDetails(reportId);

    for (const section of sections) {
      await expect(this.reportContent).toContainText(section);
    }
  }

  /**
   * Get failed checks count
   */
  async getFailedChecksCount(): Promise<number> {
    const failedItems = this.checkItems.locator('[data-status="fail"], .status-fail');
    return await failedItems.count();
  }

  /**
   * Get warning checks count
   */
  async getWarningChecksCount(): Promise<number> {
    const warningItems = this.checkItems.locator('[data-status="warning"], .status-warning');
    return await warningItems.count();
  }
}

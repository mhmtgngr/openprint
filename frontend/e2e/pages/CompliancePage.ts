import { type Page, type Locator, expect } from '@playwright/test';
import { BasePage } from './BasePage';

/**
 * Compliance Page Object
 * Handles FedRAMP, HIPAA compliance tracking, audit log export, and compliance reports
 */
export class CompliancePage extends BasePage {
  // Page heading and sections
  readonly heading: Locator;
  readonly overviewSection: Locator;
  readonly auditLogsSection: Locator;
  readonly complianceReportsSection: Locator;
  readonly dataRetentionSection: Locator;
  readonly securitySettingsSection: Locator;

  // Compliance status badges
  readonly fedrampStatusBadge: Locator;
  readonly hipaaStatusBadge: Locator;
  readonly gdprStatusBadge: Locator;
  readonly soc2StatusBadge: Locator;

  // Audit logs
  readonly auditLogsTable: Locator;
  readonly auditLogsSearchInput: Locator;
  readonly auditLogsDateFilter: Locator;
  readonly auditLogsActionFilter: Locator;
  readonly auditLogsUserFilter: Locator;
  readonly exportLogsButton: Locator;
  readonly clearLogsButton: Locator;

  // Compliance reports
  readonly generateReportButton: Locator;
  readonly reportTypeSelect: Locator;
  readonly reportDateRangeFrom: Locator;
  readonly reportDateRangeTo: Locator;
  readonly reportFormatSelect: Locator;
  readonly reportsList: Locator;
  readonly downloadReportButton: (reportId: string) => Locator;
  readonly deleteReportButton: (reportId: string) => Locator;

  // Data retention
  readonly retentionPolicyToggle: Locator;
  readonly retentionPeriodInput: Locator;
  readonly retentionPeriodSelect: Locator;
  readonly saveRetentionButton: Locator;

  // Security settings
  readonly encryptionToggle: Locator;
  readonly encryptionAlgorithm: Locator;
  readonly twoFactorToggle: Locator;
  readonly sessionTimeoutInput: Locator;
  readonly ipWhitelistInput: Locator;
  readonly addIpWhitelistButton: Locator;
  readonly ipWhitelistList: Locator;

  // Audit log specific columns
  readonly timestampColumn: Locator;
  readonly userColumn: Locator;
  readonly actionColumn: Locator;
  readonly resourceColumn: Locator;
  readonly detailsColumn: Locator;
  readonly ipAddressColumn: Locator;

  // Compliance checklist
  readonly complianceChecklist: Locator;
  readonly checklistItem: (itemName: string) => Locator;
  readonly runChecklistButton: Locator;

  // Risk assessment
  readonly riskAssessmentSection: Locator;
  readonly runRiskAssessmentButton: Locator;
  readonly riskScoreDisplay: Locator;
  readonly riskMitigationList: Locator;

  constructor(page: Page) {
    super(page);

    // Page heading and sections
    this.heading = page.getByRole('heading', { name: /compliance/i });
    this.overviewSection = page.locator('[data-testid="compliance-overview"], section:has-text("Overview")');
    this.auditLogsSection = page.locator('[data-testid="audit-logs-section"], section:has-text("Audit Logs")');
    this.complianceReportsSection = page.locator('[data-testid="compliance-reports-section"], section:has-text("Reports")');
    this.dataRetentionSection = page.locator('[data-testid="data-retention-section"], section:has-text("Data Retention")');
    this.securitySettingsSection = page.locator('[data-testid="security-settings-section"], section:has-text("Security Settings")');

    // Compliance status badges
    this.fedrampStatusBadge = page.locator('[data-testid="fedramp-status"]');
    this.hipaaStatusBadge = page.locator('[data-testid="hipaa-status"]');
    this.gdprStatusBadge = page.locator('[data-testid="gdpr-status"]');
    this.soc2StatusBadge = page.locator('[data-testid="soc2-status"]');

    // Audit logs
    this.auditLogsTable = page.locator('[data-testid="audit-logs-table"], table');
    this.auditLogsSearchInput = page.getByPlaceholder(/search|filter/i);
    this.auditLogsDateFilter = page.getByLabel(/date range|date/i);
    this.auditLogsActionFilter = page.getByLabel(/action|event type/i);
    this.auditLogsUserFilter = page.getByLabel(/user/i);
    this.exportLogsButton = page.getByRole('button', { name: /export|download/i });
    this.clearLogsButton = page.getByRole('button', { name: /clear|purge/i });

    // Compliance reports
    this.generateReportButton = page.getByRole('button', { name: /generate report|create report/i });
    this.reportTypeSelect = page.getByLabel(/report type/i);
    this.reportDateRangeFrom = page.getByLabel(/from|start date/i);
    this.reportDateRangeTo = page.getByLabel(/to|end date/i);
    this.reportFormatSelect = page.getByLabel(/format/i);
    this.reportsList = page.locator('[data-testid="reports-list"]');
    this.downloadReportButton = (reportId: string) =>
      page.locator(`[data-report-id="${reportId}"]`).getByRole('button', { name: /download/i });
    this.deleteReportButton = (reportId: string) =>
      page.locator(`[data-report-id="${reportId}"]`).getByRole('button', { name: /delete/i });

    // Data retention
    this.retentionPolicyToggle = page.getByRole('switch', { name: /retention policy|data retention/i });
    this.retentionPeriodInput = page.getByLabel(/retention period/i);
    this.retentionPeriodSelect = page.getByLabel(/period|time unit/i);
    this.saveRetentionButton = page.getByRole('button', { name: /save|update retention/i });

    // Security settings
    this.encryptionToggle = page.getByRole('switch', { name: /encryption/i });
    this.encryptionAlgorithm = page.getByLabel(/encryption algorithm|cipher/i });
    this.twoFactorToggle = page.getByRole('switch', { name: /two factor|2fa|mfa/i });
    this.sessionTimeoutInput = page.getByLabel(/session timeout/i });
    this.ipWhitelistInput = page.getByLabel(/ip address|whitelist/i });
    this.addIpWhitelistButton = page.getByRole('button', { name: /add|allow/i });
    this.ipWhitelistList = page.locator('[data-testid="ip-whitelist"]');

    // Audit log columns
    this.timestampColumn = page.locator('th:has-text("Timestamp"), td[data-col="timestamp"]');
    this.userColumn = page.locator('th:has-text("User"), td[data-col="user"]');
    this.actionColumn = page.locator('th:has-text("Action"), td[data-col="action"]');
    this.resourceColumn = page.locator('th:has-text("Resource"), td[data-col="resource"]');
    this.detailsColumn = page.locator('th:has-text("Details"), td[data-col="details"]');
    this.ipAddressColumn = page.locator('th:has-text("IP"), td[data-col="ip"]');

    // Compliance checklist
    this.complianceChecklist = page.locator('[data-testid="compliance-checklist"]');
    this.checklistItem = (itemName: string) =>
      page.locator('[data-testid="checklist-item"]').filter({ hasText: itemName });
    this.runChecklistButton = page.getByRole('button', { name: /run checklist|verify compliance/i });

    // Risk assessment
    this.riskAssessmentSection = page.locator('[data-testid="risk-assessment-section"], section:has-text("Risk Assessment")');
    this.runRiskAssessmentButton = page.getByRole('button', { name: /run risk assessment|assess risks/i });
    this.riskScoreDisplay = page.locator('[data-testid="risk-score"]');
    this.riskMitigationList = page.locator('[data-testid="risk-mitigation-list"]');
  }

  /**
   * Navigate to Compliance page
   */
  async navigate(): Promise<void> {
    await this.goto('/compliance');
  }

  /**
   * Verify Compliance page is loaded
   */
  async isLoaded(): Promise<boolean> {
    await this.page.waitForLoadState('networkidle');
    return await this.heading.isVisible();
  }

  /**
   * Get FedRAMP compliance status
   */
  async getFedRAMPStatus(): Promise<string> {
    return await this.fedrampStatusBadge.textContent() || '';
  }

  /**
   * Get HIPAA compliance status
   */
  async getHIPAAStatus(): Promise<string> {
    return await this.hipaaStatusBadge.textContent() || '';
  }

  /**
   * Get GDPR compliance status
   */
  async getGDPRStatus(): Promise<string> {
    return await this.gdprStatusBadge.textContent() || '';
  }

  /**
   * Get SOC2 compliance status
   */
  async getSOC2Status(): Promise<string> {
    return await this.soc2StatusBadge.textContent() || '';
  }

  /**
   * Check if compliance standard is met
   */
  async isCompliantWith(standard: 'fedramp' | 'hipaa' | 'gdpr' | 'soc2'): Promise<boolean> {
    const statusText = await (async () => {
      switch (standard) {
        case 'fedramp':
          return await this.getFedRAMPStatus();
        case 'hipaa':
          return await this.getHIPAAStatus();
        case 'gdpr':
          return await this.getGDPRStatus();
        case 'soc2':
          return await this.getSOC2Status();
      }
    })();
    return statusText.toLowerCase().includes('compliant') || statusText.toLowerCase().includes('passed');
  }

  /**
   * Search audit logs
   */
  async searchAuditLogs(query: string): Promise<void> {
    await this.auditLogsSearchInput.fill(query);
    await this.page.waitForTimeout(500); // Debounce wait
  }

  /**
   * Filter audit logs by date range
   */
  async filterAuditLogsByDate(fromDate: string, toDate: string): Promise<void> {
    await this.auditLogsDateFilter.click();
    // Implementation depends on date picker UI
    await this.page.fill('[data-testid="date-from"]', fromDate);
    await this.page.fill('[data-testid="date-to"]', toDate);
    await this.page.getByRole('button', { name: /apply|filter/i }).click();
  }

  /**
   * Filter audit logs by action type
   */
  async filterAuditLogsByAction(action: string): Promise<void> {
    await this.auditLogsActionFilter.selectOption(action);
  }

  /**
   * Filter audit logs by user
   */
  async filterAuditLogsByUser(user: string): Promise<void> {
    await this.auditLogsUserFilter.selectOption(user);
  }

  /**
   * Get audit log entries count
   */
  async getAuditLogCount(): Promise<number> {
    const rows = await this.auditLogsTable.locator('tbody tr').all();
    return rows.length;
  }

  /**
   * Get audit log entry by row
   */
  async getAuditLogEntry(rowNumber: number): Promise<{
    timestamp: string;
    user: string;
    action: string;
    resource: string;
    details: string;
    ipAddress: string;
  }> {
    const row = this.auditLogsTable.locator('tbody tr').nth(rowNumber);
    const cells = await row.locator('td').all();
    return {
      timestamp: await cells[0].textContent() || '',
      user: await cells[1].textContent() || '',
      action: await cells[2].textContent() || '',
      resource: await cells[3].textContent() || '',
      details: await cells[4].textContent() || '',
      ipAddress: await cells[5].textContent() || '',
    };
  }

  /**
   * Export audit logs
   */
  async exportAuditLogs(format: 'csv' | 'json' | 'xlsx'): Promise<void> {
    // Set format if provided
    if (await this.reportFormatSelect.isVisible()) {
      await this.reportFormatSelect.selectOption(format);
    }
    const downloadPromise = this.page.waitForEvent('download');
    await this.exportLogsButton.click();
    const download = await downloadPromise;
    expect(download.suggestedFilename()).toMatch(new RegExp(`\\.${format}$`));
  }

  /**
   * Clear old audit logs
   */
  async clearAuditLogs(confirm: boolean = true): Promise<void> {
    await this.clearLogsButton.click();
    if (confirm) {
      await this.page.getByRole('button', { name: /confirm|yes|clear/i }).click();
    }
  }

  /**
   * Generate compliance report
   */
  async generateReport(options: {
    type: string;
    fromDate: string;
    toDate: string;
    format: string;
  }): Promise<void> {
    await this.generateReportButton.click();
    await this.reportTypeSelect.selectOption(options.type);
    await this.reportDateRangeFrom.fill(options.fromDate);
    await this.reportDateRangeTo.fill(options.toDate);
    await this.reportFormatSelect.selectOption(options.format);
    await this.page.getByRole('button', { name: /generate|create/i }).click();
  }

  /**
   * Get available reports count
   */
  async getReportsCount(): Promise<number> {
    return await this.reportsList.locator('[data-testid="report-item"]').count();
  }

  /**
   * Download compliance report
   */
  async downloadReport(reportId: string): Promise<void> {
    const downloadPromise = this.page.waitForEvent('download');
    await this.downloadReportButton(reportId).click();
    await downloadPromise;
  }

  /**
   * Delete compliance report
   */
  async deleteReport(reportId: string): Promise<void> {
    await this.deleteReportButton(reportId).click();
    await this.page.getByRole('button', { name: /confirm|delete/i }).click();
  }

  /**
   * Enable data retention policy
   */
  async enableRetentionPolicy(period: number, unit: 'days' | 'months' | 'years'): Promise<void> {
    await this.retentionPolicyToggle.click();
    await this.retentionPeriodInput.fill(period.toString());
    if (await this.retentionPeriodSelect.isVisible()) {
      await this.retentionPeriodSelect.selectOption(unit);
    }
    await this.saveRetentionButton.click();
  }

  /**
   * Disable data retention policy
   */
  async disableRetentionPolicy(): Promise<void> {
    if (await this.retentionPolicyToggle.isChecked()) {
      await this.retentionPolicyToggle.click();
    }
    await this.saveRetentionButton.click();
  }

  /**
   * Enable encryption
   */
  async enableEncryption(algorithm: string): Promise<void> {
    await this.encryptionToggle.click();
    if (await this.encryptionAlgorithm.isVisible()) {
      await this.encryptionAlgorithm.selectOption(algorithm);
    }
    await this.saveRetentionButton.click();
  }

  /**
   * Enable two-factor authentication
   */
  async enableTwoFactor(): Promise<void> {
    await this.twoFactorToggle.click();
    await this.saveRetentionButton.click();
  }

  /**
   * Set session timeout
   */
  async setSessionTimeout(minutes: number): Promise<void> {
    await this.sessionTimeoutInput.fill(minutes.toString());
    await this.saveRetentionButton.click();
  }

  /**
   * Add IP to whitelist
   */
  async addIpToWhitelist(ipAddress: string, description?: string): Promise<void> {
    await this.ipWhitelistInput.fill(ipAddress);
    if (description) {
      await this.page.getByLabel(/description|note/i).fill(description);
    }
    await this.addIpWhitelistButton.click();
  }

  /**
   * Remove IP from whitelist
   */
  async removeIpFromWhitelist(ipAddress: string): Promise<void> {
    const ipElement = this.ipWhitelistList.getByText(ipAddress);
    const removeButton = ipElement.locator('../..').getByRole('button', { name: /remove|delete/i });
    await removeButton.click();
  }

  /**
   * Get whitelisted IPs
   */
  async getWhitelistedIPs(): Promise<string[]> {
    const ips: string[] = [];
    const ipElements = await this.ipWhitelistList.locator('[data-testid="whitelist-item"]').all();
    for (const element of ipElements) {
      ips.push(await element.textContent() || '');
    }
    return ips;
  }

  /**
   * Run compliance checklist
   */
  async runComplianceChecklist(): Promise<void> {
    await this.runChecklistButton.click();
    // Wait for checklist to complete
    await this.page.waitForTimeout(3000);
  }

  /**
   * Get checklist item status
   */
  async getChecklistItemStatus(itemName: string): Promise<'pass' | 'fail' | 'warning' | 'pending'> {
    const item = this.checklistItem(itemName);
    const classList = await item.getAttribute('class') || '';
    if (classList.includes('pass') || classList.includes('success')) return 'pass';
    if (classList.includes('fail') || classList.includes('error')) return 'fail';
    if (classList.includes('warning')) return 'warning';
    return 'pending';
  }

  /**
   * Run risk assessment
   */
  async runRiskAssessment(): Promise<void> {
    await this.runRiskAssessmentButton.click();
    // Wait for assessment to complete
    await this.page.waitForTimeout(5000);
  }

  /**
   * Get risk score
   */
  async getRiskScore(): Promise<number> {
    const scoreText = await this.riskScoreDisplay.textContent() || '';
    const match = scoreText.match(/\d+/);
    return match ? parseInt(match[0], 10) : 0;
  }

  /**
   * Get risk mitigation suggestions
   */
  async getRiskMitigationSuggestions(): Promise<string[]> {
    const suggestions: string[] = [];
    const items = await this.riskMitigationList.locator('[data-testid="mitigation-item"]').all();
    for (const item of items) {
      suggestions.push(await item.textContent() || '');
    }
    return suggestions;
  }

  /**
   * Verify audit log entry exists
   */
  async verifyAuditLogEntry(entry: {
    user?: string;
    action?: string;
    resource?: string;
  }): Promise<boolean> {
    let selector = 'tbody tr';
    if (entry.user) {
      selector = `${selector}:has-text("${entry.user}")`;
    }
    if (entry.action) {
      selector = `${selector}:has-text("${entry.action}")`;
    }
    if (entry.resource) {
      selector = `${selector}:has-text("${entry.resource}")`;
    }
    const count = await this.auditLogsTable.locator(selector).count();
    return count > 0;
  }

  /**
   * Get compliance overview statistics
   */
  async getComplianceOverview(): Promise<{
    totalLogs: number;
    compliantStandards: number;
    pendingActions: number;
    lastAuditDate: string;
  }> {
    const totalLogs = await this.auditLogsTable.locator('tbody tr').count();
    const compliantStandards = await this.page.locator('[data-testid*="status"].compliant, .badge.success').count();
    const pendingActions = await this.page.locator('[data-testid="pending-action"]').count();
    const lastAuditDate = await this.page.locator('[data-testid="last-audit-date"]').textContent() || '';

    return { totalLogs, compliantStandards, pendingActions, lastAuditDate };
  }

  /**
   * Verify page has required compliance sections
   */
  async verifyComplianceSections(): Promise<void> {
    await expect(this.overviewSection).toBeVisible();
    await expect(this.auditLogsSection).toBeVisible();
    await expect(this.complianceReportsSection).toBeVisible();
  }
}

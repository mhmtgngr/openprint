import { test, expect } from '@playwright/test';
import { AuthPage } from '../pages/AuthPage';
import { CompliancePage } from '../pages/CompliancePage';
import { testUsers } from '../helpers/test-data';

test.describe('Compliance - Overview', () => {
  let compliancePage: CompliancePage;

  test.beforeEach(async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.admin.email, testUsers.admin.password);

    compliancePage = new CompliancePage(page);

    // Mock compliance API
    await page.route('**/api/v1/compliance/**', (route) => {
      if (route.request().method() === 'GET') {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            fedramp: { status: 'compliant', lastAudit: '2024-01-01' },
            hipaa: { status: 'compliant', lastAudit: '2024-01-01' },
            gdpr: { status: 'compliant', lastAudit: '2024-01-01' },
            soc2: { status: 'in_progress', lastAudit: '2024-01-01' },
            totalLogs: 1523,
            compliantStandards: 3,
            pendingActions: 5,
          }),
        });
      }
    });
  });

  test('should access compliance page', async ({ page }) => {
    await compliancePage.navigate();
    await expect(compliancePage.heading).toBeVisible();
  });

  test('should display compliance overview section', async ({ page }) => {
    await compliancePage.navigate();
    await expect(compliancePage.overviewSection).toBeVisible();
  });

  test('should display FedRAMP status badge', async ({ page }) => {
    await compliancePage.navigate();
    await expect(compliancePage.fedrampStatusBadge).toBeVisible();
    const status = await compliancePage.getFedRAMPStatus();
    expect(status).toBeTruthy();
  });

  test('should display HIPAA status badge', async ({ page }) => {
    await compliancePage.navigate();
    await expect(compliancePage.hipaaStatusBadge).toBeVisible();
    const status = await compliancePage.getHIPAAStatus();
    expect(status).toBeTruthy();
  });

  test('should display GDPR status badge', async ({ page }) => {
    await compliancePage.navigate();
    await expect(compliancePage.gdprStatusBadge).toBeVisible();
    const status = await compliancePage.getGDPRStatus();
    expect(status).toBeTruthy();
  });

  test('should display SOC2 status badge', async ({ page }) => {
    await compliancePage.navigate();
    await expect(compliancePage.soc2StatusBadge).toBeVisible();
    const status = await compliancePage.getSOC2Status();
    expect(status).toBeTruthy();
  });

  test('should check FedRAMP compliance status', async ({ page }) => {
    await compliancePage.navigate();
    const isCompliant = await compliancePage.isCompliantWith('fedramp');
    expect(isCompliant).toBe(true);
  });

  test('should check HIPAA compliance status', async ({ page }) => {
    await compliancePage.navigate();
    const isCompliant = await compliancePage.isCompliantWith('hipaa');
    expect(isCompliant).toBe(true);
  });

  test('should get compliance overview statistics', async ({ page }) => {
    await compliancePage.navigate();
    const stats = await compliancePage.getComplianceOverview();
    expect(stats.totalLogs).toBeGreaterThan(0);
    expect(stats.compliantStandards).toBeGreaterThan(0);
  });
});

test.describe('Compliance - Audit Logs', () => {
  test.beforeEach(async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.admin.email, testUsers.admin.password);

    // Mock audit logs API
    await page.route('**/api/v1/compliance/audit-logs/**', (route) => {
      if (route.request().method() === 'GET') {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            logs: [
              {
                id: '1',
                timestamp: '2024-01-15T10:30:00Z',
                user: 'admin@openprint.test',
                action: 'login',
                resource: '/login',
                details: 'Successful login',
                ipAddress: '192.168.1.100',
              },
              {
                id: '2',
                timestamp: '2024-01-15T10:25:00Z',
                user: 'user@openprint.test',
                action: 'print_job_created',
                resource: '/jobs',
                details: 'Created job "Document.pdf"',
                ipAddress: '192.168.1.101',
              },
              {
                id: '3',
                timestamp: '2024-01-15T10:20:00Z',
                user: 'admin@openprint.test',
                action: 'policy_updated',
                resource: '/policies',
                details: 'Updated policy "Color Restriction"',
                ipAddress: '192.168.1.100',
              },
            ],
            total: 3,
          }),
        });
      }
    });
  });

  test('should display audit logs section', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.navigate();
    await expect(compliancePage.auditLogsSection).toBeVisible();
  });

  test('should display audit logs table', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.navigate();
    await expect(compliancePage.auditLogsTable).toBeVisible();
  });

  test('should get audit log count', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.navigate();
    const count = await compliancePage.getAuditLogCount();
    expect(count).toBeGreaterThan(0);
  });

  test('should get audit log entry details', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.navigate();
    const entry = await compliancePage.getAuditLogEntry(0);
    expect(entry.user).toBeTruthy();
    expect(entry.action).toBeTruthy();
    expect(entry.timestamp).toBeTruthy();
  });

  test('should search audit logs', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.navigate();
    await compliancePage.searchAuditLogs('login');
    await page.waitForTimeout(500); // Wait for debounced search
  });

  test('should filter audit logs by action', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.navigate();
    await compliancePage.filterAuditLogsByAction('login');
    await page.waitForTimeout(500);
  });

  test('should filter audit logs by user', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.navigate();
    await compliancePage.filterAuditLogsByUser('admin@openprint.test');
    await page.waitForTimeout(500);
  });

  test('should verify specific audit log entry exists', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.navigate();
    const exists = await compliancePage.verifyAuditLogEntry({
      user: 'admin@openprint.test',
      action: 'login',
    });
    expect(exists).toBe(true);
  });

  test('should export audit logs as CSV', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.navigate();

    const downloadPromise = page.waitForEvent('download');
    await compliancePage.exportAuditLogs('csv');
    const download = await downloadPromise;
    expect(download.suggestedFilename()).toMatch(/\.csv$/);
  });

  test('should export audit logs as JSON', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.navigate();

    const downloadPromise = page.waitForEvent('download');
    await compliancePage.exportAuditLogs('json');
    const download = await downloadPromise;
    expect(download.suggestedFilename()).toMatch(/\.json$/);
  });

  test('should export audit logs as XLSX', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.navigate();

    const downloadPromise = page.waitForEvent('download');
    await compliancePage.exportAuditLogs('xlsx');
    const download = await downloadPromise;
    expect(download.suggestedFilename()).toMatch(/\.xlsx$/);
  });

  test('should clear audit logs with confirmation', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.navigate();

    // Mock delete API
    await page.route('**/api/v1/compliance/audit-logs', (route) => {
      if (route.request().method() === 'DELETE') {
        route.fulfill({
          status: 204,
          contentType: 'application/json',
          body: '',
        });
      }
    });

    await compliancePage.clearAuditLogs(true);
    await expect(page.getByText(/cleared|deleted/i)).toBeVisible();
  });
});

test.describe('Compliance - Reports', () => {
  test.beforeEach(async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.admin.email, testUsers.admin.password);

    // Mock reports API
    await page.route('**/api/v1/compliance/reports/**', (route) => {
      if (route.request().method() === 'GET') {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            reports: [
              {
                id: 'report-1',
                name: 'FedRAMP Assessment',
                type: 'fedramp',
                createdAt: '2024-01-01T00:00:00Z',
                status: 'complete',
              },
              {
                id: 'report-2',
                name: 'HIPAA Audit',
                type: 'hipaa',
                createdAt: '2024-01-02T00:00:00Z',
                status: 'complete',
              },
            ],
          }),
        });
      } else if (route.request().method() === 'POST') {
        route.fulfill({
          status: 201,
          contentType: 'application/json',
          body: JSON.stringify({
            id: 'report-new',
            status: 'generating',
          }),
        });
      }
    });
  });

  test('should display compliance reports section', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.navigate();
    await expect(compliancePage.complianceReportsSection).toBeVisible();
  });

  test('should display reports list', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.navigate();
    await expect(compliancePage.reportsList).toBeVisible();
  });

  test('should get reports count', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.navigate();
    const count = await compliancePage.getReportsCount();
    expect(count).toBeGreaterThan(0);
  });

  test('should generate compliance report', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.navigate();
    await compliancePage.generateReport({
      type: 'fedramp',
      fromDate: '2024-01-01',
      toDate: '2024-01-31',
      format: 'pdf',
    });
    await expect(page.getByText(/generating|queued/i)).toBeVisible();
  });

  test('should download report', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.navigate();

    // Mock download
    await page.route('**/api/v1/compliance/reports/*/download', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/pdf',
        body: Buffer.from('mock-pdf-content'),
      });
    });

    const downloadPromise = page.waitForEvent('download');
    await compliancePage.downloadReport('report-1');
    await downloadPromise;
  });

  test('should delete report with confirmation', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.navigate();

    // Mock delete
    await page.route('**/api/v1/compliance/reports/*', (route) => {
      if (route.request().method() === 'DELETE') {
        route.fulfill({
          status: 204,
          body: '',
        });
      }
    });

    await compliancePage.deleteReport('report-1');
    await expect(page.getByText(/deleted/i)).toBeVisible();
  });
});

test.describe('Compliance - Data Retention', () => {
  test.beforeEach(async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.admin.email, testUsers.admin.password);

    // Mock retention API
    await page.route('**/api/v1/compliance/retention/**', (route) => {
      if (route.request().method() === 'PUT') {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ success: true }),
        });
      }
    });
  });

  test('should display data retention section', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.navigate();
    await expect(compliancePage.dataRetentionSection).toBeVisible();
  });

  test('should enable retention policy', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.navigate();
    await compliancePage.enableRetentionPolicy(90, 'days');
    await expect(page.getByText(/saved|updated/i)).toBeVisible();
  });

  test('should disable retention policy', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.navigate();
    await compliancePage.disableRetentionPolicy();
    await expect(page.getByText(/saved|disabled/i)).toBeVisible();
  });

  test('should set retention period', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.navigate();
    await compliancePage.retentionPeriodInput.fill('365');
    await compliancePage.saveRetentionButton.click();
    await expect(page.getByText(/saved|updated/i)).toBeVisible();
  });
});

test.describe('Compliance - Security Settings', () => {
  test.beforeEach(async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.admin.email, testUsers.admin.password);

    // Mock security settings API
    await page.route('**/api/v1/compliance/security/**', (route) => {
      if (route.request().method() === 'PUT') {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ success: true }),
        });
      }
    });
  });

  test('should display security settings section', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.navigate();
    await expect(compliancePage.securitySettingsSection).toBeVisible();
  });

  test('should enable encryption', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.navigate();
    await compliancePage.enableEncryption('AES-256');
    await expect(page.getByText(/saved|updated/i)).toBeVisible();
  });

  test('should enable two-factor authentication', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.navigate();
    await compliancePage.enableTwoFactor();
    await expect(page.getByText(/saved|enabled/i)).toBeVisible();
  });

  test('should set session timeout', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.navigate();
    await compliancePage.setSessionTimeout(30);
    await expect(page.getByText(/saved|updated/i)).toBeVisible();
  });

  test('should add IP to whitelist', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.navigate();
    await compliancePage.addIpToWhitelist('192.168.1.100', 'Office Network');
    await expect(page.getByText(/added|saved/i)).toBeVisible();
  });

  test('should remove IP from whitelist', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.navigate();
    await compliancePage.removeIpFromWhitelist('192.168.1.100');
    await expect(page.getByText(/removed/i)).toBeVisible();
  });

  test('should get whitelisted IPs', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.navigate();

    // Mock whitelist API
    await page.route('**/api/v1/compliance/security/whitelist', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          ips: ['192.168.1.100', '192.168.1.101', '10.0.0.1'],
        }),
      });
    });

    const ips = await compliancePage.getWhitelistedIPs();
    expect(ips.length).toBeGreaterThan(0);
  });
});

test.describe('Compliance - Checklist', () => {
  test.beforeEach(async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.admin.email, testUsers.admin.password);

    // Mock checklist API
    await page.route('**/api/v1/compliance/checklist/**', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          checklist: [
            { name: 'Access Control', status: 'pass' },
            { name: 'Audit Logging', status: 'pass' },
            { name: 'Data Encryption', status: 'pass' },
            { name: 'Incident Response', status: 'warning' },
            { name: 'Security Training', status: 'pending' },
          ],
        }),
      });
    });
  });

  test('should display compliance checklist', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.navigate();
    await expect(compliancePage.complianceChecklist).toBeVisible();
  });

  test('should run compliance checklist', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.navigate();
    await compliancePage.runComplianceChecklist();
    await expect(page.getByText(/checklist complete|verification complete/i)).toBeVisible();
  });

  test('should get checklist item status', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.navigate();
    const status = await compliancePage.getChecklistItemStatus('Access Control');
    expect(status).toBe('pass');
  });
});

test.describe('Compliance - Risk Assessment', () => {
  test.beforeEach(async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.admin.email, testUsers.admin.password);

    // Mock risk assessment API
    await page.route('**/api/v1/compliance/risk-assessment/**', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          riskScore: 25,
          level: 'low',
          mitigations: [
            'Enable two-factor authentication for all users',
            'Implement IP whitelist for admin access',
            'Review audit logs weekly',
          ],
        }),
      });
    });
  });

  test('should display risk assessment section', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.navigate();
    await expect(compliancePage.riskAssessmentSection).toBeVisible();
  });

  test('should run risk assessment', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.navigate();
    await compliancePage.runRiskAssessment();
    await expect(compliancePage.riskScoreDisplay).toBeVisible();
  });

  test('should get risk score', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.navigate();
    await compliancePage.runRiskAssessment();
    const score = await compliancePage.getRiskScore();
    expect(score).toBeGreaterThanOrEqual(0);
    expect(score).toBeLessThanOrEqual(100);
  });

  test('should get risk mitigation suggestions', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.navigate();
    const suggestions = await compliancePage.getRiskMitigationSuggestions();
    expect(suggestions.length).toBeGreaterThan(0);
  });
});

test.describe('Compliance - Access Control', () => {
  test('should restrict compliance page to admin users', async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.user.email, testUsers.user.password);

    await page.goto('/compliance');

    const isForbidden = await page.getByText(/forbidden|not authorized|access denied/i).isVisible();
    const isRedirected = page.url().includes('/dashboard');

    expect(isForbidden || isRedirected).toBe(true);
  });

  test('should allow admin access to compliance page', async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.admin.email, testUsers.admin.password);

    const compliancePage = new CompliancePage(page);
    await compliancePage.navigate();
    await expect(compliancePage.heading).toBeVisible();
  });

  test('should allow owner access to compliance page', async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.owner.email, testUsers.owner.password);

    const compliancePage = new CompliancePage(page);
    await compliancePage.navigate();
    await expect(compliancePage.heading).toBeVisible();
  });
});

test.describe('Compliance - Audit Log Columns', () => {
  test.beforeEach(async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.admin.email, testUsers.admin.password);

    // Mock audit logs API
    await page.route('**/api/v1/compliance/audit-logs/**', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          logs: [
            {
              id: '1',
              timestamp: '2024-01-15T10:30:00Z',
              user: 'admin@openprint.test',
              action: 'login',
              resource: '/login',
              details: 'Successful login',
              ipAddress: '192.168.1.100',
            },
          ],
        }),
      });
    });
  });

  test('should display timestamp column', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.navigate();
    await expect(compliancePage.timestampColumn).toBeVisible();
  });

  test('should display user column', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.navigate();
    await expect(compliancePage.userColumn).toBeVisible();
  });

  test('should display action column', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.navigate();
    await expect(compliancePage.actionColumn).toBeVisible();
  });

  test('should display resource column', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.navigate();
    await expect(compliancePage.resourceColumn).toBeVisible();
  });

  test('should display details column', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.navigate();
    await expect(compliancePage.detailsColumn).toBeVisible();
  });

  test('should display IP address column', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.navigate();
    await expect(compliancePage.ipAddressColumn).toBeVisible();
  });
});

test.describe('Compliance - Date Range Filtering', () => {
  test.beforeEach(async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.admin.email, testUsers.admin.password);
  });

  test('should filter logs by date range', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.navigate();
    await compliancePage.filterAuditLogsByDate('2024-01-01', '2024-01-31');
    await page.waitForTimeout(500);
  });

  test('should display date filter button', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.navigate();
    await expect(compliancePage.auditLogsDateFilter).toBeVisible();
  });
});

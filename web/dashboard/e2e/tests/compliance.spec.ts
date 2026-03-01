/**
 * Compliance E2E Tests
 * Tests for FedRAMP, HIPAA compliance, audit logs, and security
 */
import { test, expect } from '@playwright/test';
import { setupAuthAndNavigate, mockUsers } from '../helpers';
import { CompliancePage } from '../pages/CompliancePage';
import { AuditLogFactory, UserFactory } from '../factories/TestDataFactory';

test.describe('Compliance - Overview', () => {
  test('should display compliance dashboard', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.setupMocks();
    await setupAuthAndNavigate(page, '/compliance', mockUsers[1]); // Admin

    await expect(compliancePage.heading).toContainText('Compliance');
    await expect(compliancePage.complianceStatus).toBeVisible();
  });

  test('should show overall compliance status', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.setupMocks();
    await setupAuthAndNavigate(page, '/compliance', mockUsers[1]);

    const status = await compliancePage.getComplianceStatus();
    expect(['compliant', 'non_compliant', 'pending_review']).toContain(status);
  });

  test('should display audit dates', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.setupMocks();
    await setupAuthAndNavigate(page, '/compliance', mockUsers[1]);

    await expect(compliancePage.lastAuditDate).toBeVisible();
    await expect(compliancePage.nextAuditDate).toBeVisible();
  });

  test('should show overview cards', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.setupMocks();
    await setupAuthAndNavigate(page, '/compliance', mockUsers[1]);

    await expect(compliancePage.overviewCards).toBeVisible();
  });
});

test.describe('Compliance - Reports', () => {
  test('should display reports list', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.setupMocks();
    await setupAuthAndNavigate(page, '/compliance', mockUsers[1]);

    const count = await compliancePage.getReportCount();
    expect(count).toBeGreaterThan(0);
  });

  test('should filter reports by type', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.setupMocks();
    await setupAuthAndNavigate(page, '/compliance', mockUsers[1]);

    await compliancePage.filterReportsByType('fedramp');
    await page.waitForTimeout(500);

    // Should show only FedRAMP reports
    await expect(compliancePage.reportItems).toBeVisible();
  });

  test('should search reports', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.setupMocks();
    await setupAuthAndNavigate(page, '/compliance', mockUsers[1]);

    await compliancePage.searchReports('FedRAMP');
    await page.waitForTimeout(500);
  });

  test('should generate new report', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.setupMocks();
    await setupAuthAndNavigate(page, '/compliance', mockUsers[1]);

    await compliancePage.generateReport('hipaa');
    await page.waitForTimeout(1000);
  });

  test('should view report details', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.setupMocks();
    await setupAuthAndNavigate(page, '/compliance', mockUsers[1]);

    await compliancePage.viewReportDetails('report-1');
    await expect(compliancePage.reportDetailPage).toBeVisible();
  });

  test('should download report', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.setupMocks();
    await setupAuthAndNavigate(page, '/compliance', mockUsers[1]);

    const downloadPromise = page.waitForEvent('download');
    await compliancePage.downloadReport('report-1');
    await downloadPromise;
  });

  test('should share report via email', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.setupMocks();
    await setupAuthAndNavigate(page, '/compliance', mockUsers[1]);

    await compliancePage.shareReport('report-1', 'compliance@example.com');
  });

  test('should schedule report generation', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.setupMocks();
    await setupAuthAndNavigate(page, '/compliance', mockUsers[1]);

    await compliancePage.scheduleReport('fedramp', 'monthly', ['admin@example.com']);
  });
});

test.describe('Compliance - Checks', () => {
  test('should display compliance checks', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.setupMocks();
    await setupAuthAndNavigate(page, '/compliance', mockUsers[1]);

    const count = await compliancePage.getCheckCount();
    expect(count).toBeGreaterThan(0);
  });

  test('should filter checks by category', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.setupMocks();
    await setupAuthAndNavigate(page, '/compliance', mockUsers[1]);

    await compliancePage.filterChecksByCategory('Access Control');
    await page.waitForTimeout(500);
  });

  test('should run compliance checks', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.setupMocks();
    await setupAuthAndNavigate(page, '/compliance', mockUsers[1]);

    await compliancePage.runComplianceChecks();
    await page.waitForTimeout(2000);
  });

  test('should view check details', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.setupMocks();
    await setupAuthAndNavigate(page, '/compliance', mockUsers[1]);

    await compliancePage.viewCheckDetails('check-1');
    await expect(compliancePage.checkDetailModal).toBeVisible();
  });

  test('should show check status', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.setupMocks();
    await setupAuthAndNavigate(page, '/compliance', mockUsers[1]);

    const status = await compliancePage.getCheckStatus('check-1');
    expect(['pass', 'fail', 'warning']).toContain(status);
  });

  test('should mark check as resolved', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.setupMocks();
    await setupAuthAndNavigate(page, '/compliance', mockUsers[1]);

    // Mock a failed check
    await page.route('**/api/v1/compliance/checks*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              id: 'check-failed',
              category: 'Audit Logging',
              control: 'AU-2: Audit Events',
              status: 'fail',
              description: 'Some audit logs are missing',
              lastChecked: '2024-02-27T00:00:00Z',
            },
          ],
          total: 1,
        }),
      });
    });

    await page.reload();
    await page.waitForLoadState('networkidle');

    await compliancePage.markCheckResolved('check-failed');
  });

  test('should display check evidence', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.setupMocks();

    // Mock check with evidence
    await page.route('**/api/v1/compliance/checks/*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: 'check-1',
          category: 'Access Control',
          control: 'AC-1',
          status: 'pass',
          description: 'Policy document exists',
          evidence: 'Documented in policy-ac-1.pdf, last updated 2024-01-15',
          lastChecked: '2024-02-27T00:00:00Z',
        }),
      });
    });

    await setupAuthAndNavigate(page, '/compliance', mockUsers[1]);
    await compliancePage.viewCheckDetails('check-1');

    await expect(compliancePage.checkEvidence).toBeVisible();
    await expect(compliancePage.checkEvidence).toContainText('policy-ac-1.pdf');
  });
});

test.describe('Compliance - Audit Log Export', () => {
  test('should open export modal', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.setupMocks();
    await setupAuthAndNavigate(page, '/compliance', mockUsers[1]);

    await compliancePage.exportAuditLogsButton.click();
    await expect(compliancePage.exportModal).toBeVisible();
  });

  test('should export audit logs with date range', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.setupMocks();
    await setupAuthAndNavigate(page, '/compliance', mockUsers[1]);

    const downloadPromise = page.waitForEvent('download');
    await compliancePage.exportAuditLogs({
      from: '2024-01-01',
      to: '2024-01-31',
      format: 'csv',
      includePII: false,
    });
    await downloadPromise;
  });

  test('should export in different formats', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.setupMocks();
    await setupAuthAndNavigate(page, '/compliance', mockUsers[1]);

    // Test JSON export
    const download1 = page.waitForEvent('download');
    await compliancePage.exportAuditLogs({
      from: '2024-01-01',
      to: '2024-01-31',
      format: 'json',
    });
    await download1;

    // Test PDF export
    const download2 = page.waitForEvent('download');
    await compliancePage.exportAuditLogs({
      from: '2024-01-01',
      to: '2024-01-31',
      format: 'pdf',
    });
    await download2;
  });

  test('should include PII option', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.setupMocks();
    await setupAuthAndNavigate(page, '/compliance', mockUsers[1]);

    await compliancePage.exportAuditLogsButton.click();

    await compliancePage.exportIncludePIIToggle.check();
    const isChecked = await compliancePage.exportIncludePIIToggle.isChecked();
    expect(isChecked).toBe(true);
  });
});

test.describe('Compliance - Data Retention', () => {
  test('should display retention settings', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.setupMocks();
    await setupAuthAndNavigate(page, '/compliance', mockUsers[1]);

    await expect(compliancePage.retentionSettings).toBeVisible();
  });

  test('should update retention settings', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.setupMocks();
    await setupAuthAndNavigate(page, '/compliance', mockUsers[1]);

    await compliancePage.updateRetentionSettings(3650, true); // 10 years

    const period = await compliancePage.getRetentionPeriod();
    expect(period).toBe(3650);
  });

  test('should show auto-delete option', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.setupMocks();
    await setupAuthAndNavigate(page, '/compliance', mockUsers[1]);

    await expect(compliancePage.autoDeleteToggle).toBeVisible();
  });
});

test.describe('Compliance - Security Settings', () => {
  test('should display encryption status', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.setupMocks();
    await setupAuthAndNavigate(page, '/compliance', mockUsers[1]);

    await compliancePage.verifyEncryptionStatus('enabled');
  });

  test('should rotate encryption keys', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.setupMocks();
    await setupAuthAndNavigate(page, '/compliance', mockUsers[1]);

    await compliancePage.rotateKeys();
  });

  test('should verify 2FA requirement', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.setupMocks();
    await setupAuthAndNavigate(page, '/compliance', mockUsers[1]);

    await compliancePage.verify2FARequired(true);
  });

  test('should manage IP whitelist', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.setupMocks();
    await setupAuthAndNavigate(page, '/compliance', mockUsers[1]);

    await compliancePage.manageIPWhitelist();

    const whitelistModal = page.locator('[data-testid="ip-whitelist-modal"]');
    await expect(whitelistModal).toBeVisible();
  });

  test('should add IP to whitelist', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.setupMocks();
    await setupAuthAndNavigate(page, '/compliance', mockUsers[1]);

    await compliancePage.addIPToWhitelist('192.168.1.100');
  });
});

test.describe('Compliance - Access Logs', () => {
  test('should display access logs', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.setupMocks();
    await setupAuthAndNavigate(page, '/compliance', mockUsers[1]);

    await expect(compliancePage.accessLogs).toBeVisible();
  });

  test('should show access log entries', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.setupMocks();
    await setupAuthAndNavigate(page, '/compliance', mockUsers[1]);

    const count = await compliancePage.getAccessLogCount();
    expect(count).toBeGreaterThan(0);
  });

  test('should filter access logs', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.setupMocks();
    await setupAuthAndNavigate(page, '/compliance', mockUsers[1]);

    await compliancePage.filterAccessLogs({
      user: 'test@example.com',
      action: 'login',
    });

    await page.waitForTimeout(500);
  });

  test('should filter by date range', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.setupMocks();
    await setupAuthAndNavigate(page, '/compliance', mockUsers[1]);

    await compliancePage.filterAccessLogs({
      dateFrom: '2024-01-01',
      dateTo: '2024-01-31',
    });

    await page.waitForTimeout(500);
  });
});

test.describe('Compliance - FedRAMP Requirements', () => {
  test('should include FedRAMP controls in checks', async ({ page }) => {
    const compliancePage = new CompliancePage(page);

    // Mock FedRAMP-specific checks
    await page.route('**/api/v1/compliance/checks*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              id: 'fedramp-ac-1',
              category: 'Access Control',
              control: 'AC-1: Access Control Policy and Procedures',
              status: 'pass',
              description: 'FedRAMP Moderate Baseline',
              lastChecked: '2024-02-27T00:00:00Z',
            },
            {
              id: 'fedramp-au-2',
              category: 'Audit and Accountability',
              control: 'AU-2: Audit Events',
              status: 'pass',
              description: 'FedRAMP Moderate Baseline',
              lastChecked: '2024-02-27T00:00:00Z',
            },
          ],
          total: 2,
        }),
      });
    });

    await setupAuthAndNavigate(page, '/compliance', mockUsers[1]);

    await compliancePage.filterChecksByCategory('Access Control');
    await expect(compliancePage.checkItems).toContainText('FedRAMP');
  });

  test('should generate FedRAMP assessment report', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.setupMocks();
    await setupAuthAndNavigate(page, '/compliance', mockUsers[1]);

    await compliancePage.generateReport('fedramp');

    // Should trigger report generation
    await page.waitForTimeout(1000);
  });
});

test.describe('Compliance - HIPAA Requirements', () => {
  test('should include HIPAA controls in checks', async ({ page }) => {
    const compliancePage = new CompliancePage(page);

    // Mock HIPAA-specific checks
    await page.route('**/api/v1/compliance/checks*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              id: 'hipaa-164.312(a)(1)',
              category: 'Access Control',
              control: '§164.312(a)(1): Access Control',
              status: 'pass',
              description: 'HIPAA Security Rule',
              lastChecked: '2024-02-27T00:00:00Z',
            },
            {
              id: 'hipaa-164.312(e)(1)',
              category: 'Encryption',
              control: '§164.312(e)(1): Encryption and Decryption',
              status: 'pass',
              description: 'HIPAA Security Rule',
              lastChecked: '2024-02-27T00:00:00Z',
            },
          ],
          total: 2,
        }),
      });
    });

    await setupAuthAndNavigate(page, '/compliance', mockUsers[1]);

    await compliancePage.filterChecksByCategory('Encryption');
    await expect(compliancePage.checkItems).toContainText('HIPAA');
  });

  test('should generate HIPAA assessment report', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.setupMocks();
    await setupAuthAndNavigate(page, '/compliance', mockUsers[1]);

    await compliancePage.generateReport('hipaa');

    // Should trigger report generation
    await page.waitForTimeout(1000);
  });

  test('should verify PHI handling in audit logs', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.setupMocks();
    await setupAuthAndNavigate(page, '/compliance', mockUsers[1]);

    // Verify that audit logs can be exported without PII
    await compliancePage.exportAuditLogs({
      from: '2024-01-01',
      to: '2024-01-31',
      format: 'csv',
      includePII: false,
    });

    const downloadPromise = page.waitForEvent('download');
    await compliancePage.confirmExportButton.click();
    await downloadPromise;
  });
});

test.describe('Compliance - Admin Access Only', () => {
  test('should require admin access', async ({ page }) => {
    // Non-admin user should be denied access
    await setupAuthAndNavigate(page, '/compliance', mockUsers[0]);

    const isDenied = page.url().includes('denied') || page.url().includes('/dashboard');
    expect(isDenied).toBe(true);
  });

  test('should allow admin access', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.setupMocks();
    await setupAuthAndNavigate(page, '/compliance', mockUsers[1]); // Admin

    await compliancePage.isLoaded();
    const isDenied = page.url().includes('denied') || page.url().includes('/dashboard');
    expect(isDenied).toBe(false);
  });
});

test.describe('Compliance - Report Sections', () => {
  test('should include required sections in reports', async ({ page }) => {
    const compliancePage = new CompliancePage(page);
    await compliancePage.setupMocks();
    await setupAuthAndNavigate(page, '/compliance', mockUsers[1]);

    await compliancePage.viewReportDetails('report-1');

    await compliancePage.verifyReportSections('report-1', [
      'Executive Summary',
      'Scope',
      'Controls Assessment',
      'Findings',
      'Remediation Plan',
    ]);
  });
});

test.describe('Compliance - Failed Checks Handling', () => {
  test('should count failed checks', async ({ page }) => {
    const compliancePage = new CompliancePage(page);

    // Mock with failed checks
    await page.route('**/api/v1/compliance/checks*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              id: 'check-pass-1',
              category: 'Access Control',
              control: 'AC-1',
              status: 'pass',
              description: 'Pass',
              lastChecked: '2024-02-27T00:00:00Z',
            },
            {
              id: 'check-fail-1',
              category: 'Encryption',
              control: 'SC-13',
              status: 'fail',
              description: 'Fail',
              lastChecked: '2024-02-27T00:00:00Z',
            },
            {
              id: 'check-warning-1',
              category: 'Audit Logging',
              control: 'AU-2',
              status: 'warning',
              description: 'Warning',
              lastChecked: '2024-02-27T00:00:00Z',
            },
          ],
          total: 3,
        }),
      });
    });

    await setupAuthAndNavigate(page, '/compliance', mockUsers[1]);

    const failedCount = await compliancePage.getFailedChecksCount();
    const warningCount = await compliancePage.getWarningChecksCount();

    expect(failedCount).toBe(1);
    expect(warningCount).toBe(1);
  });
});

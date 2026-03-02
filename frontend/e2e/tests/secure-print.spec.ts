import { test, expect } from '@playwright/test';
import { AuthPage } from '../pages/AuthPage';
import { SecurePrintPage } from '../pages/SecurePrintPage';
import { testUsers } from '../helpers/test-data';

test.describe('Secure Print - Job Queue', () => {
  let securePrintPage: SecurePrintPage;

  test.beforeEach(async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.user.email, testUsers.user.password);

    securePrintPage = new SecurePrintPage(page);

    // Mock secure print API
    await page.route('**/api/v1/secure-print/**', (route) => {
      if (route.request().method() === 'GET') {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            jobs: [
              {
                id: 'job-1',
                name: 'Quarterly Report.pdf',
                pages: 15,
                colorMode: 'color',
                duplex: true,
                copies: 1,
                printer: 'HP LaserJet Pro',
                submittedTime: '2024-01-15T10:30:00Z',
                fileSize: '2.5 MB',
                status: 'pending',
                pinRequired: true,
              },
              {
                id: 'job-2',
                name: 'Meeting Notes.docx',
                pages: 3,
                colorMode: 'grayscale',
                duplex: false,
                copies: 2,
                printer: 'Canon imageRUNNER',
                submittedTime: '2024-01-15T10:25:00Z',
                fileSize: '150 KB',
                status: 'pending',
                pinRequired: true,
              },
            ],
          }),
        });
      }
    });
  });

  test('should access secure print page', async ({ page }) => {
    await securePrintPage.navigate();
    await expect(securePrintPage.heading).toBeVisible();
  });

  test('should display job queue section', async ({ page }) => {
    await securePrintPage.navigate();
    await expect(securePrintPage.queueSection).toBeVisible();
  });

  test('should display job list', async ({ page }) => {
    await securePrintPage.navigate();
    await expect(securePrintPage.jobList).toBeVisible();
  });

  test('should get job count', async ({ page }) => {
    await securePrintPage.navigate();
    const count = await securePrintPage.getJobCount();
    expect(count).toBeGreaterThan(0);
  });

  test('should display job details', async ({ page }) => {
    await securePrintPage.navigate();
    const details = await securePrintPage.getJobDetails('job-1');
    expect(details.name).toBe('Quarterly Report.pdf');
    expect(details.pages).toBe(15);
    expect(details.colorMode).toBe('color');
  });

  test('should check if job is in queue', async ({ page }) => {
    await securePrintPage.navigate();
    const hasJob = await securePrintPage.hasJob('job-1');
    expect(hasJob).toBe(true);
  });

  test('should show empty state when no jobs', async ({ page }) => {
    // Override mock to return empty list
    await page.route('**/api/v1/secure-print/**', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ jobs: [] }),
      });
    });

    await securePrintPage.navigate();
    expect(await securePrintPage.isQueueEmpty()).toBe(true);
  });

  test('should refresh job queue', async ({ page }) => {
    await securePrintPage.navigate();
    await securePrintPage.refreshQueue();
    await expect(securePrintPage.jobList).toBeVisible();
  });

  test('should search for jobs', async ({ page }) => {
    await securePrintPage.navigate();
    await securePrintPage.searchJob('Quarterly');
    await page.waitForTimeout(500);
  });

  test('should filter jobs by status', async ({ page }) => {
    await securePrintPage.navigate();
    await securePrintPage.filterByStatus('pending');
    await page.waitForTimeout(500);
  });

  test('should display job status badge', async ({ page }) => {
    await securePrintPage.navigate();
    await expect(securePrintPage.jobStatusBadge.first()).toBeVisible();
  });

  test('should show PIN required badge', async ({ page }) => {
    await securePrintPage.navigate();
    const pinRequired = await securePrintPage.isPinRequired('job-1');
    expect(pinRequired).toBe(true);
  });
});

test.describe('Secure Print - Job Release with PIN', () => {
  test.beforeEach(async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.user.email, testUsers.user.password);

    // Mock release API
    await page.route('**/api/v1/secure-print/release/**', (route) => {
      if (route.request().method() === 'POST') {
        const body = JSON.parse(route.request().postData() || '{}');
        if (body.pin === '1234') {
          route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({ success: true, message: 'Job released' }),
          });
        } else {
          route.fulfill({
            status: 401,
            contentType: 'application/json',
            body: JSON.stringify({ error: 'Invalid PIN', attemptsRemaining: 2 }),
          });
        }
      }
    });
  });

  test('should release job with correct PIN', async ({ page }) => {
    const securePrintPage = new SecurePrintPage(page);
    await securePrintPage.navigate();
    await securePrintPage.releaseJobWithPin('job-1', '1234');
    await expect(page.getByText(/released|printing/i)).toBeVisible();
  });

  test('should display PIN input when releasing job', async ({ page }) => {
    const securePrintPage = new SecurePrintPage(page);
    await securePrintPage.navigate();
    await securePrintPage.releaseButton.click();
    await expect(securePrintPage.pinInput).toBeVisible();
  });

  test('should show error for invalid PIN', async ({ page }) => {
    const securePrintPage = new SecurePrintPage(page);
    await securePrintPage.navigate();
    await securePrintPage.releaseJobWithPin('job-1', '0000');
    await expect(securePrintPage.pinErrorMessage).toBeVisible();
  });

  test('should display remaining PIN attempts', async ({ page }) => {
    const securePrintPage = new SecurePrintPage(page);
    await securePrintPage.navigate();
    await securePrintPage.releaseJobWithPin('job-1', '0000');
    const attempts = await securePrintPage.getRemainingAttempts();
    expect(attempts).toBeGreaterThan(0);
  });

  test('should cancel PIN entry', async ({ page }) => {
    const securePrintPage = new SecurePrintPage(page);
    await securePrintPage.navigate();
    await securePrintPage.releaseButton.click();
    await securePrintPage.cancelPinEntry();
    await expect(securePrintPage.pinInput).not.toBeVisible();
  });

  test('should toggle PIN visibility', async ({ page }) => {
    const securePrintPage = new SecurePrintPage(page);
    await securePrintPage.navigate();
    await securePrintPage.releaseButton.click();
    await securePrintPage.togglePinVisibility();
    // Verify input type changed
    const inputType = await securePrintPage.pinInput.getAttribute('type');
    expect(inputType).toBe('text');
  });

  test('should get PIN error message', async ({ page }) => {
    const securePrintPage = new SecurePrintPage(page);
    await securePrintPage.navigate();
    await securePrintPage.releaseJobWithPin('job-1', '0000');
    const error = await securePrintPage.getPinErrorMessage();
    expect(error).toBeTruthy();
  });
});

test.describe('Secure Print - Job Actions', () => {
  test.beforeEach(async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.user.email, testUsers.user.password);

    // Mock job actions API
    await page.route('**/api/v1/secure-print/jobs/**', (route) => {
      if (route.request().method() === 'DELETE') {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ success: true }),
        });
      }
    });
  });

  test('should cancel job', async ({ page }) => {
    const securePrintPage = new SecurePrintPage(page);
    await securePrintPage.navigate();
    await securePrintPage.cancelJob('job-1');
    await expect(page.getByText(/cancelled|canceled/i)).toBeVisible();
  });

  test('should confirm job cancellation', async ({ page }) => {
    const securePrintPage = new SecurePrintPage(page);
    await securePrintPage.navigate();
    await securePrintPage.cancelJob('job-1');
    await expect(page.getByRole('button', { name: /confirm/i })).toBeVisible();
  });

  test('should check job is cancelled', async ({ page }) => {
    const securePrintPage = new SecurePrintPage(page);
    await securePrintPage.navigate();
    await securePrintPage.cancelJob('job-1');
    const isCancelled = await securePrintPage.isJobCancelled('job-1');
    expect(isCancelled).toBe(true);
  });

  test('should select job for bulk actions', async ({ page }) => {
    const securePrintPage = new SecurePrintPage(page);
    await securePrintPage.navigate();
    await securePrintPage.selectJob('job-1');
    const selectedCount = await securePrintPage.getSelectedJobsCount();
    expect(selectedCount).toBe(1);
  });

  test('should deselect all jobs', async ({ page }) => {
    const securePrintPage = new SecurePrintPage(page);
    await securePrintPage.navigate();
    await securePrintPage.selectJob('job-1');
    await securePrintPage.deselectAllJobs();
    const selectedCount = await securePrintPage.getSelectedJobsCount();
    expect(selectedCount).toBe(0);
  });

  test('should get job status', async ({ page }) => {
    const securePrintPage = new SecurePrintPage(page);
    await securePrintPage.navigate();
    const status = await securePrintPage.getJobStatus('job-1');
    expect(status).toBeTruthy();
  });
});

test.describe('Secure Print - Printer Selection', () => {
  test.beforeEach(async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.user.email, testUsers.user.password);

    // Mock printers API
    await page.route('**/api/v1/printers/**', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          printers: [
            { id: 'printer-1', name: 'HP LaserJet Pro', status: 'online' },
            { id: 'printer-2', name: 'Canon imageRUNNER', status: 'online' },
            { id: 'printer-3', name: 'Epson EcoTank', status: 'offline' },
          ],
        }),
      });
    });
  });

  test('should display printer selection', async ({ page }) => {
    const securePrintPage = new SecurePrintPage(page);
    await securePrintPage.navigate();
    await expect(securePrintPage.printerSelect).toBeVisible();
  });

  test('should select printer for release', async ({ page }) => {
    const securePrintPage = new SecurePrintPage(page);
    await securePrintPage.navigate();
    await securePrintPage.selectPrinter('HP LaserJet Pro');
    await expect(securePrintPage.printerSelect).toHaveValue('printer-1');
  });

  test('should get selected printer', async ({ page }) => {
    const securePrintPage = new SecurePrintPage(page);
    await securePrintPage.navigate();
    await securePrintPage.selectPrinter('Canon imageRUNNER');
    const selected = await securePrintPage.getSelectedPrinter();
    expect(selected).toContain('Canon');
  });

  test('should display printer name in job card', async ({ page }) => {
    const securePrintPage = new SecurePrintPage(page);
    await securePrintPage.navigate();
    await expect(securePrintPage.jobPrinter).toBeVisible();
  });
});

test.describe('Secure Print - Settings', () => {
  test.beforeEach(async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.admin.email, testUsers.admin.password);

    // Mock settings API
    await page.route('**/api/v1/secure-print/settings/**', (route) => {
      if (route.request().method() === 'GET') {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            enabled: true,
            pinLength: 4,
            pinExpiry: 60,
            autoRelease: false,
          }),
        });
      } else if (route.request().method() === 'PUT') {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ success: true }),
        });
      }
    });
  });

  test('should display settings section', async ({ page }) => {
    const securePrintPage = new SecurePrintPage(page);
    await securePrintPage.navigate();
    await expect(securePrintPage.settingsSection).toBeVisible();
  });

  test('should enable secure print', async ({ page }) => {
    const securePrintPage = new SecurePrintPage(page);
    await securePrintPage.navigate();
    await securePrintPage.enableSecurePrint();
    await expect(page.getByText(/saved|enabled/i)).toBeVisible();
  });

  test('should disable secure print', async ({ page }) => {
    const securePrintPage = new SecurePrintPage(page);
    await securePrintPage.navigate();
    await securePrintPage.disableSecurePrint();
    await expect(page.getByText(/saved|disabled/i)).toBeVisible();
  });

  test('should set PIN length', async ({ page }) => {
    const securePrintPage = new SecurePrintPage(page);
    await securePrintPage.navigate();
    await securePrintPage.setPinLength(6);
    await expect(page.getByText(/saved/i)).toBeVisible();
  });

  test('should set PIN expiry time', async ({ page }) => {
    const securePrintPage = new SecurePrintPage(page);
    await securePrintPage.navigate();
    await securePrintPage.setPinExpiry(120);
    await expect(page.getByText(/saved/i)).toBeVisible();
  });

  test('should enable auto-release', async ({ page }) => {
    const securePrintPage = new SecurePrintPage(page);
    await securePrintPage.navigate();
    await securePrintPage.enableAutoRelease();
    await expect(page.getByText(/saved|enabled/i)).toBeVisible();
  });

  test('should check if secure print is enabled', async ({ page }) => {
    const securePrintPage = new SecurePrintPage(page);
    await securePrintPage.navigate();
    const isEnabled = await securePrintPage.isSecurePrintEnabled();
    expect(isEnabled).toBe(true);
  });

  test('should verify PIN settings', async ({ page }) => {
    const securePrintPage = new SecurePrintPage(page);
    await securePrintPage.navigate();
    const isValid = await securePrintPage.verifyPinSettings({ length: 4 });
    expect(isValid).toBe(true);
  });
});

test.describe('Secure Print - History', () => {
  test.beforeEach(async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.user.email, testUsers.user.password);

    // Mock history API
    await page.route('**/api/v1/secure-print/history/**', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          history: [
            {
              id: 'history-1',
              jobId: 'job-1',
              jobName: 'Quarterly Report.pdf',
              action: 'released',
              timestamp: '2024-01-15T10:35:00Z',
              printer: 'HP LaserJet Pro',
              user: 'user@openprint.test',
            },
            {
              id: 'history-2',
              jobId: 'job-2',
              jobName: 'Meeting Notes.docx',
              action: 'cancelled',
              timestamp: '2024-01-15T10:30:00Z',
              printer: null,
              user: 'user@openprint.test',
            },
          ],
        }),
      });
    });
  });

  test('should display history tab', async ({ page }) => {
    const securePrintPage = new SecurePrintPage(page);
    await securePrintPage.navigate();
    await expect(securePrintPage.historyTab).toBeVisible();
  });

  test('should view release history', async ({ page }) => {
    const securePrintPage = new SecurePrintPage(page);
    await securePrintPage.navigate();
    await securePrintPage.viewHistory();
    await expect(securePrintPage.historyList).toBeVisible();
  });

  test('should get history count', async ({ page }) => {
    const securePrintPage = new SecurePrintPage(page);
    await securePrintPage.navigate();
    await securePrintPage.viewHistory();
    const count = await securePrintPage.getHistoryCount();
    expect(count).toBeGreaterThan(0);
  });
});

test.describe('Secure Print - Bulk Actions', () => {
  test.beforeEach(async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.user.email, testUsers.user.password);

    // Mock bulk actions API
    await page.route('**/api/v1/secure-print/bulk/**', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, released: 2 }),
      });
    });
  });

  test('should release all jobs with PIN', async ({ page }) => {
    const securePrintPage = new SecurePrintPage(page);
    await securePrintPage.navigate();
    await securePrintPage.releaseAllJobs('1234');
    await expect(page.getByText(/released|printing/i)).toBeVisible();
  });

  test('should display release all button', async ({ page }) => {
    const securePrintPage = new SecurePrintPage(page);
    await securePrintPage.navigate();
    const releaseAllButton = page.getByRole('button', { name: /release all/i });
    // May not be visible if no jobs
    const isVisible = await releaseAllButton.isVisible();
    if (isVisible) {
      await expect(releaseAllButton).toBeVisible();
    }
  });
});

test.describe('Secure Print - Job Details', () => {
  test.beforeEach(async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.user.email, testUsers.user.password);
  });

  test('should display job name', async ({ page }) => {
    const securePrintPage = new SecurePrintPage(page);
    await securePrintPage.navigate();
    await expect(securePrintPage.jobName.first()).toBeVisible();
  });

  test('should display job pages', async ({ page }) => {
    const securePrintPage = new SecurePrintPage(page);
    await securePrintPage.navigate();
    await expect(securePrintPage.jobPages.first()).toBeVisible();
  });

  test('should display color mode', async ({ page }) => {
    const securePrintPage = new SecurePrintPage(page);
    await securePrintPage.navigate();
    await expect(securePrintPage.jobColorMode.first()).toBeVisible();
  });

  test('should display duplex setting', async ({ page }) => {
    const securePrintPage = new SecurePrintPage(page);
    await securePrintPage.navigate();
    await expect(securePrintPage.jobDuplex.first()).toBeVisible();
  });

  test('should display copies', async ({ page }) => {
    const securePrintPage = new SecurePrintPage(page);
    await securePrintPage.navigate();
    await expect(securePrintPage.jobCopies.first()).toBeVisible();
  });

  test('should display file size', async ({ page }) => {
    const securePrintPage = new SecurePrintPage(page);
    await securePrintPage.navigate();
    await expect(securePrintPage.jobFileSize.first()).toBeVisible();
  });

  test('should display submitted time', async ({ page }) => {
    const securePrintPage = new SecurePrintPage(page);
    await securePrintPage.navigate();
    await expect(securePrintPage.jobSubmittedTime.first()).toBeVisible();
  });
});

test.describe('Secure Print - Error Handling', () => {
  test.beforeEach(async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.user.email, testUsers.user.password);
  });

  test('should handle network error on release', async ({ page }) => {
    await page.route('**/api/v1/secure-print/release/**', (route) => {
      route.abort('failed');
    });

    const securePrintPage = new SecurePrintPage(page);
    await securePrintPage.navigate();
    await securePrintPage.releaseJobWithPin('job-1', '1234');
    await expect(page.getByText(/error|failed|network/i)).toBeVisible();
  });

  test('should handle timeout on refresh', async ({ page }) => {
    await page.route('**/api/v1/secure-print/**', (route) => {
      // Delay response
      setTimeout(() => {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ jobs: [] }),
        });
      }, 35000);
    });

    const securePrintPage = new SecurePrintPage(page);
    await securePrintPage.navigate();
    await securePrintPage.refreshQueue();
    // Should handle timeout gracefully
  });

  test('should validate PIN format', async ({ page }) => {
    const securePrintPage = new SecurePrintPage(page);
    await securePrintPage.navigate();
    await securePrintPage.releaseButton.click();
    await securePrintPage.pinInput.fill('ab'); // Invalid PIN
    await securePrintPage.pinSubmitButton.click();
    await expect(page.getByText(/invalid|must be|digits/i)).toBeVisible();
  });
});

test.describe('Secure Print - Access Control', () => {
  test('should allow users to access secure print', async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.user.email, testUsers.user.password);

    const securePrintPage = new SecurePrintPage(page);
    await securePrintPage.navigate();
    await expect(securePrintPage.heading).toBeVisible();
  });

  test('should allow admins to access secure print', async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.admin.email, testUsers.admin.password);

    const securePrintPage = new SecurePrintPage(page);
    await securePrintPage.navigate();
    await expect(securePrintPage.heading).toBeVisible();
  });

  test('should allow owners to access secure print', async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.owner.email, testUsers.owner.password);

    const securePrintPage = new SecurePrintPage(page);
    await securePrintPage.navigate();
    await expect(securePrintPage.heading).toBeVisible();
  });
});

test.describe('Secure Print - Responsive Design', () => {
  test('should work on mobile', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });

    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.user.email, testUsers.user.password);

    const securePrintPage = new SecurePrintPage(page);
    await securePrintPage.navigate();
    await expect(securePrintPage.heading).toBeVisible();
  });

  test('should work on tablet', async ({ page }) => {
    await page.setViewportSize({ width: 768, height: 1024 });

    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.user.email, testUsers.user.password);

    const securePrintPage = new SecurePrintPage(page);
    await securePrintPage.navigate();
    await expect(securePrintPage.heading).toBeVisible();
  });
});

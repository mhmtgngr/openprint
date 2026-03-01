import { test, expect } from '@playwright/test';
import { AuthPage } from '../pages/AuthPage';
import { Microsoft365Page } from '../pages/Microsoft365Page';
import { testUsers } from '../helpers/test-data';

test.describe('Microsoft 365 Integration - Configuration', () => {
  let m365Page: Microsoft365Page;

  test.beforeEach(async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.admin.email, testUsers.admin.password);

    m365Page = new Microsoft365Page(page);

    // Mock M365 API responses
    await page.route('**/api/v1/integrations/m365/**', (route) => {
      if (route.request().method() === 'GET') {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            enabled: true,
            tenantId: 'test-tenant-id',
            domain: 'test.onmicrosoft.com',
            connectionStatus: 'connected',
            lastSync: '2024-01-15T10:30:00Z',
            oneDrive: { enabled: true, folder: '/OpenPrint' },
            sharePoint: { enabled: false },
            userSync: { enabled: true, interval: 'hourly' },
          }),
        });
      } else if (route.request().method() === 'POST') {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ success: true, message: 'Configuration saved' }),
        });
      } else {
        route.continue();
      }
    });
  });

  test('should access Microsoft 365 settings page', async ({ page }) => {
    await m365Page.navigate();
    await expect(m365Page.heading).toBeVisible();
  });

  test('should display configuration section', async ({ page }) => {
    await m365Page.navigate();
    await expect(m365Page.configurationSection).toBeVisible();
    await expect(m365Page.tenantIdInput).toBeVisible();
    await expect(m365Page.clientIdInput).toBeVisible();
    await expect(m365Page.clientSecretInput).toBeVisible();
  });

  test('should display connection status', async ({ page }) => {
    await m365Page.navigate();
    await expect(m365Page.connectionStatusBadge).toBeVisible();
    await expect(m365Page.connectionStatusText).toBeVisible();
  });

  test('should show connected status when integration is active', async ({ page }) => {
    await m365Page.navigate();
    const isHealthy = await m365Page.isConnectionHealthy();
    expect(isHealthy).toBe(true);
  });

  test('should enable Microsoft 365 integration', async ({ page }) => {
    await m365Page.navigate();
    await m365Page.enableIntegration();
    await expect(page.getByText(/saved|updated/i)).toBeVisible();
  });

  test('should disable Microsoft 365 integration', async ({ page }) => {
    await m365Page.navigate();
    await m365Page.disableIntegration();
    await expect(page.getByText(/saved|updated/i)).toBeVisible();
  });

  test('should configure M365 integration credentials', async ({ page }) => {
    await m365Page.navigate();
    await m365Page.configure({
      tenantId: 'e2e-test-tenant-id',
      clientId: 'e2e-test-client-id',
      clientSecret: 'e2e-test-secret',
      domain: 'e2etest.onmicrosoft.com',
    });
    await m365Page.verifyConfigurationSaved();
  });

  test('should test connection to Microsoft 365', async ({ page }) => {
    await m365Page.navigate();
    await m365Page.testConnection();
    await m365Page.verifyTestConnectionResult('success');
  });

  test('should show last sync time', async ({ page }) => {
    await m365Page.navigate();
    const lastSync = await m365Page.getLastSyncTime();
    expect(lastSync).toBeTruthy();
  });

  test('should trigger manual sync', async ({ page }) => {
    await m365Page.navigate();
    await m365Page.triggerSync();
    await expect(page.getByText(/sync complete|synced/i)).toBeVisible();
  });

  test('should disconnect integration with confirmation', async ({ page }) => {
    await m365Page.navigate();
    await m365Page.disconnect();
    await expect(page.getByText(/disconnected|removed/i)).toBeVisible();
  });
});

test.describe('Microsoft 365 - OneDrive Integration', () => {
  test.beforeEach(async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.admin.email, testUsers.admin.password);

    // Mock OneDrive API
    await page.route('**/api/v1/integrations/m365/onedrive/**', (route) => {
      if (route.request().method() === 'POST') {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ success: true, message: 'OneDrive enabled' }),
        });
      }
    });
  });

  test('should display OneDrive section', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.navigate();
    await expect(m365Page.oneDriveSection).toBeVisible();
  });

  test('should enable OneDrive integration', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.navigate();
    await m365Page.enableOneDrive('/OpenPrint/Documents');
    await expect(page.getByText(/saved|onedrive enabled/i)).toBeVisible();
  });

  test('should disable OneDrive integration', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.navigate();
    await m365Page.disableOneDrive();
    await expect(page.getByText(/saved|disabled/i)).toBeVisible();
  });

  test('should set OneDrive destination folder', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.navigate();
    await m365Page.oneDriveFolderInput.fill('/OpenPrint/Scanned');
    await m365Page.saveButton.click();
    await expect(page.getByText(/saved/i)).toBeVisible();
  });

  test('should test OneDrive connection', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.navigate();
    await m365Page.testOneDriveConnection();
    await expect(page.getByText(/connection successful|test passed/i)).toBeVisible();
  });
});

test.describe('Microsoft 365 - SharePoint Integration', () => {
  test.beforeEach(async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.admin.email, testUsers.admin.password);

    // Mock SharePoint API
    await page.route('**/api/v1/integrations/m365/sharepoint/**', (route) => {
      if (route.request().method() === 'POST') {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ success: true, message: 'SharePoint enabled' }),
        });
      }
    });
  });

  test('should display SharePoint section', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.navigate();
    await expect(m365Page.sharePointSection).toBeVisible();
  });

  test('should enable SharePoint integration', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.navigate();
    await m365Page.enableSharePoint({
      siteUrl: 'https://test.sharepoint.com/sites/openprint',
      library: 'Documents',
    });
    await expect(page.getByText(/saved|sharepoint enabled/i)).toBeVisible();
  });

  test('should disable SharePoint integration', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.navigate();
    await m365Page.disableSharePoint();
    await expect(page.getByText(/saved|disabled/i)).toBeVisible();
  });

  test('should configure SharePoint site and library', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.navigate();
    await m365Page.sharePointSiteInput.fill('https://test.sharepoint.com/sites/openprint');
    await m365Page.sharePointLibraryInput.fill('Shared Documents');
    await m365Page.saveButton.click();
    await expect(page.getByText(/saved/i)).toBeVisible();
  });

  test('should test SharePoint connection', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.navigate();
    await m365Page.testSharePointConnection();
    await expect(page.getByText(/connection successful|test passed/i)).toBeVisible();
  });
});

test.describe('Microsoft 365 - Email to Print', () => {
  test.beforeEach(async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.admin.email, testUsers.admin.password);

    // Mock Email to Print API
    await page.route('**/api/v1/integrations/m365/email-to-print/**', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          enabled: true,
          emailAddress: 'print@test.onmicrosoft.com',
          allowedSenders: ['@test.com', '@example.com'],
        }),
      });
    });
  });

  test('should display Email to Print section', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.navigate();
    await expect(m365Page.emailToPrintSection).toBeVisible();
  });

  test('should display email address for printing', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.navigate();
    await expect(m365Page.emailAddressDisplay).toBeVisible();
    const email = await m365Page.getEmailToPrintAddress();
    expect(email).toContain('@');
  });

  test('should enable Email to Print', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.navigate();
    await m365Page.enableEmailToPrint();
    await expect(page.getByText(/enabled|saved/i)).toBeVisible();
  });

  test('should add allowed sender domain', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.navigate();
    await m365Page.addAllowedSender('@company.com');
    await expect(page.getByText(/added|saved/i)).toBeVisible();
  });

  test('should remove allowed sender domain', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.navigate();
    await m365Page.removeAllowedSender('@test.com');
    await expect(page.getByText(/removed/i)).toBeVisible();
  });

  test('should display list of allowed senders', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.navigate();
    await expect(m365Page.allowedSendersList).toBeVisible();
  });
});

test.describe('Microsoft 365 - User Sync', () => {
  test.beforeEach(async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.admin.email, testUsers.admin.password);

    // Mock User Sync API
    await page.route('**/api/v1/integrations/m365/sync/**', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          enabled: true,
          interval: 'hourly',
          groups: ['All Employees', 'IT Department', 'Finance Team'],
          lastSync: '2024-01-15T10:30:00Z',
        }),
      });
    });
  });

  test('should display User Sync section', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.navigate();
    await expect(m365Page.userSyncSection).toBeVisible();
  });

  test('should enable user sync', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.navigate();
    await m365Page.enableUserSync('hourly');
    await expect(page.getByText(/saved|sync enabled/i)).toBeVisible();
  });

  test('should set sync interval', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.navigate();
    await m365Page.syncIntervalSelect.selectOption('daily');
    await m365Page.saveButton.click();
    await expect(page.getByText(/saved/i)).toBeVisible();
  });

  test('should add sync group', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.navigate();
    await m365Page.addSyncGroup('Marketing Team');
    await expect(page.getByText(/added/i)).toBeVisible();
  });

  test('should remove sync group', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.navigate();
    await m365Page.removeSyncGroup('IT Department');
    await expect(page.getByText(/removed/i)).toBeVisible();
  });

  test('should display sync groups list', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.navigate();
    await expect(m365Page.syncGroupsList).toBeVisible();
  });

  test('should get sync groups', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.navigate();
    const groups = await m365Page.getSyncGroups();
    expect(groups.length).toBeGreaterThan(0);
  });
});

test.describe('Microsoft 365 - Permissions', () => {
  test.beforeEach(async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.admin.email, testUsers.admin.password);

    // Mock Permissions API
    await page.route('**/api/v1/integrations/m365/permissions/**', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          permissions: [
            { name: 'User.Read', status: 'granted' },
            { name: 'Mail.Send', status: 'granted' },
            { name: 'Files.ReadWrite', status: 'pending' },
            { name: 'Sites.ReadWrite.All', status: 'revoked' },
          ],
        }),
      });
    });
  });

  test('should display permissions section', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.navigate();
    await expect(m365Page.permissionsSection).toBeVisible();
  });

  test('should display list of permissions', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.navigate();
    await expect(m365Page.permissionsList).toBeVisible();
  });

  test('should check if permission is granted', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.navigate();
    const hasPermission = await m365Page.hasPermission('User.Read');
    expect(hasPermission).toBe(true);
  });

  test('should request permission', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.navigate();
    await m365Page.requestPermission('Files.ReadWrite');
    await expect(page.getByText(/permission requested|authorization required/i)).toBeVisible();
  });
});

test.describe('Microsoft 365 - Activity Logs', () => {
  test.beforeEach(async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.admin.email, testUsers.admin.password);

    // Mock Activity Logs API
    await page.route('**/api/v1/integrations/m365/activity/**', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          logs: [
            { id: '1', action: 'Sync Completed', timestamp: '2024-01-15T10:30:00Z', status: 'success' },
            { id: '2', action: 'OneDrive Upload', timestamp: '2024-01-15T09:15:00Z', status: 'success' },
            { id: '3', action: 'Email Processed', timestamp: '2024-01-15T08:45:00Z', status: 'success' },
          ],
        }),
      });
    });
  });

  test('should display activity logs section', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.navigate();
    await expect(m365Page.activityLogsSection).toBeVisible();
  });

  test('should display activity logs list', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.navigate();
    await expect(m365Page.activityLogsList).toBeVisible();
  });

  test('should get activity logs count', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.navigate();
    const count = await m365Page.getActivityLogsCount();
    expect(count).toBeGreaterThan(0);
  });

  test('should view all activity logs', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.navigate();
    await m365Page.viewAllActivityLogs();
    await expect(page).toHaveURL(/activity|logs/i);
  });
});

test.describe('Microsoft 365 - Error Handling', () => {
  test.beforeEach(async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.admin.email, testUsers.admin.password);
  });

  test('should display error on failed connection test', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);

    await page.route('**/api/v1/integrations/m365/test', (route) => {
      route.fulfill({
        status: 400,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Invalid credentials' }),
      });
    });

    await m365Page.navigate();
    await m365Page.testConnection();
    await m365Page.verifyTestConnectionResult('error');
  });

  test('should display error on sync failure', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);

    await page.route('**/api/v1/integrations/m365/sync', (route) => {
      route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Sync failed' }),
      });
    });

    await m365Page.navigate();
    await m365Page.syncNowButton.click();
    await expect(page.getByText(/sync failed|error/i)).toBeVisible();
  });

  test('should validate tenant ID format', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.navigate();
    await m365Page.tenantIdInput.fill('invalid-tenant');
    await m365Page.saveButton.click();
    await expect(page.getByText(/invalid tenant|validation error/i)).toBeVisible();
  });

  test('should validate domain format', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.navigate();
    await m365Page.domainInput.fill('invalid-domain');
    await m365Page.saveButton.click();
    await expect(page.getByText(/invalid domain|must end with/i)).toBeVisible();
  });
});

test.describe('Microsoft 365 - Overview Section', () => {
  test.beforeEach(async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.admin.email, testUsers.admin.password);
  });

  test('should display overview section', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.navigate();
    await expect(m365Page.overviewSection).toBeVisible();
  });

  test('should display integration status summary', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.navigate();
    await expect(m365Page.connectionStatusBadge).toBeVisible();
    await expect(m365Page.lastSyncTime).toBeVisible();
  });

  test('should display quick actions', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.navigate();
    await expect(m365Page.syncNowButton).toBeVisible();
    await expect(m365Page.testConnectionButton).toBeVisible();
  });
});

test.describe('Microsoft 365 - Access Control', () => {
  test('should restrict M365 settings to admin users', async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.user.email, testUsers.user.password);

    await page.goto('/settings/microsoft-365');

    const isForbidden = await page.getByText(/forbidden|not authorized|access denied/i).isVisible();
    const isRedirected = page.url().includes('/dashboard');

    expect(isForbidden || isRedirected).toBe(true);
  });

  test('should allow admin access to M365 settings', async ({ page }) => {
    const authPage = new AuthPage(page);
    await authPage.gotoLogin();
    await authPage.login(testUsers.admin.email, testUsers.admin.password);

    const m365Page = new Microsoft365Page(page);
    await m365Page.navigate();
    await expect(m365Page.heading).toBeVisible();
  });
});

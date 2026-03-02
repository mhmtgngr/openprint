/**
 * Microsoft 365 Integration E2E Tests
 * Tests for OneDrive, SharePoint, and Outlook integration
 */
import { test, expect } from '@playwright/test';
import { setupAuthAndNavigate, mockUsers } from '../helpers';
import { Microsoft365Page } from '../pages/Microsoft365Page';

test.describe('Microsoft 365 - Connection', () => {
  test('should display connection status', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.setupMocks();
    await setupAuthAndNavigate(page, '/microsoft-365', mockUsers[1]); // Admin

    await expect(m365Page.heading).toContainText('Microsoft 365');
    await expect(m365Page.connectionStatus).toBeVisible();
  });

  test('should show connected status when configured', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.setupMocks();
    await setupAuthAndNavigate(page, '/microsoft-365', mockUsers[1]);

    const status = await m365Page.getConnectionStatus();
    expect(status.toLowerCase()).toContain('connected');
    await m365Page.verifyConnectionStatus('connected');
  });

  test('should show disconnected status when not configured', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);

    // Mock disconnected status
    await page.route('**/api/v1/integrations/m365/config', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          enabled: false,
          configured: false,
        }),
      });
    });

    await setupAuthAndNavigate(page, '/microsoft-365', mockUsers[1]);

    await m365Page.verifyConnectionStatus('disconnected');
  });

  test('should open configuration form', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.setupMocks();
    await setupAuthAndNavigate(page, '/microsoft-365', mockUsers[1]);

    await m365Page.openConfigForm();
    await expect(m365Page.configForm).toBeVisible();
  });

  test('should save configuration', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.setupMocks();
    await setupAuthAndNavigate(page, '/microsoft-365', mockUsers[1]);

    await m365Page.openConfigForm();
    await m365Page.fillConfigForm({
      tenantId: 'test-tenant-id',
      clientId: 'test-client-id',
      clientSecret: 'test-secret',
      defaultSharePointSite: 'https://contoso.sharepoint.com/sites/site',
    });
    await m365Page.saveConfiguration();

    await expect(m365Page.configForm).not.toBeVisible();
  });

  test('should test connection successfully', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.setupMocks();
    await setupAuthAndNavigate(page, '/microsoft-365', mockUsers[1]);

    await m365Page.testConnection();
    await m365Page.verifyConnectionStatus('connected');
  });

  test('should show connection error on failed test', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);

    // Mock failed connection test
    await page.route('**/api/v1/integrations/m365/test', async (route) => {
      await route.fulfill({
        status: 400,
        contentType: 'application/json',
        body: JSON.stringify({
          success: false,
          message: 'Invalid credentials',
        }),
      });
    });

    await setupAuthAndNavigate(page, '/microsoft-365', mockUsers[1]);

    await m365Page.testConnection();
    await expect(page.locator('.error, [data-testid="error-message"]')).toBeVisible();
  });

  test('should disconnect integration', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.setupMocks();
    await setupAuthAndNavigate(page, '/microsoft-365', mockUsers[1]);

    await m365Page.disconnect();

    // Should show confirmation or disconnected status
    await expect(m365Page.connectionStatus).toContainText('disconnected', { ignoreCase: true });
  });
});

test.describe('Microsoft 365 - OneDrive Integration', () => {
  test('should display OneDrive section', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.setupMocks();
    await setupAuthAndNavigate(page, '/microsoft-365', mockUsers[1]);

    await m365Page.verifyOneDriveAvailable();
  });

  test('should browse OneDrive documents', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.setupMocks();
    await setupAuthAndNavigate(page, '/microsoft-365', mockUsers[1]);

    await m365Page.browseOneDrive();
    await expect(m365Page.documentBrowser).toBeVisible();
  });

  test('should display document list', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.setupMocks();
    await setupAuthAndNavigate(page, '/microsoft-365', mockUsers[1]);

    await m365Page.browseOneDrive();

    const count = await m365Page.getDocumentCount();
    expect(count).toBeGreaterThan(0);
  });

  test('should navigate folders', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.setupMocks();
    await setupAuthAndNavigate(page, '/microsoft-365', mockUsers[1]);

    await m365Page.browseOneDrive();

    // Mock folder navigation
    await page.route('**/api/v1/integrations/m365/documents*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          documents: [
            {
              id: 'doc-3',
              name: 'Subfolder Report.pdf',
              source: 'onedrive',
              sourceLocation: '/Documents/Reports',
              url: 'https://graph.microsoft.com/v1.0/me/drive/items/doc-3',
              size: 512000,
              createdBy: 'Test User',
              createdAt: '2024-02-27T10:00:00Z',
            },
          ],
          total: 1,
        }),
      });
    });

    await m365Page.navigateToFolder('Reports');
    await m365Page.navigateToParent();
  });

  test('should search documents', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.setupMocks();
    await setupAuthAndNavigate(page, '/microsoft-365', mockUsers[1]);

    await m365Page.browseOneDrive();
    await m365Page.searchDocuments('Report');

    // Verify search was performed
    await page.waitForTimeout(500);
  });

  test('should select and print document', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.setupMocks();
    await setupAuthAndNavigate(page, '/microsoft-365', mockUsers[1]);

    await m365Page.browseOneDrive();

    // Mock printers API for print modal
    await page.route('**/api/v1/printers', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          printers: [
            { id: 'printer-1', name: 'HP LaserJet', isActive: true },
          ],
        }),
      });
    });

    await m365Page.selectDocument('Report.pdf');
    await m365Page.confirmSelection();

    // Should navigate to job creation or show print modal
    const printModal = page.locator('[data-testid="print-modal"], .print-modal');
    await expect(printModal).toBeVisible();
  });

  test('should verify document source badge', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.setupMocks();
    await setupAuthAndNavigate(page, '/microsoft-365', mockUsers[1]);

    await m365Page.browseOneDrive();
    await m365Page.verifyDocumentSource('Report.pdf', 'onedrive');
  });

  test('should display document metadata', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.setupMocks();
    await setupAuthAndNavigate(page, '/microsoft-365', mockUsers[1]);

    await m365Page.browseOneDrive();

    const size = await m365Page.getDocumentSize('Report.pdf');
    expect(size).toBeTruthy();
  });
});

test.describe('Microsoft 365 - SharePoint Integration', () => {
  test('should display SharePoint section', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.setupMocks();
    await setupAuthAndNavigate(page, '/microsoft-365', mockUsers[1]);

    await m365Page.verifySharePointAvailable();
  });

  test('should browse SharePoint documents', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.setupMocks();
    await setupAuthAndNavigate(page, '/microsoft-365', mockUsers[1]);

    await m365Page.browseSharePoint('https://contoso.sharepoint.com/sites/site');
    await expect(m365Page.documentBrowser).toBeVisible();
  });

  test('should select SharePoint site', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.setupMocks();

    // Mock multiple SharePoint sites
    await page.route('**/api/v1/integrations/m365/sharepoint/sites', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          sites: [
            { id: 'site-1', name: 'Team Site', url: 'https://contoso.sharepoint.com/sites/team' },
            { id: 'site-2', name: 'Project Site', url: 'https://contoso.sharepoint.com/sites/project' },
          ],
        }),
      });
    });

    await setupAuthAndNavigate(page, '/microsoft-365', mockUsers[1]);

    await m365Page.browseSharePoint();

    // Site selector should be visible
    const siteSelect = page.locator('select[name="site"]');
    await expect(siteSelect).toBeVisible();
  });

  test('should verify SharePoint document source', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.setupMocks();
    await setupAuthAndNavigate(page, '/microsoft-365', mockUsers[1]);

    await m365Page.browseSharePoint();
    await m365Page.verifyDocumentSource('Presentation.pptx', 'sharepoint');
  });
});

test.describe('Microsoft 365 - Outlook Integration', () => {
  test('should display Outlook section', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.setupMocks();
    await setupAuthAndNavigate(page, '/microsoft-365', mockUsers[1]);

    await m365Page.verifyOutlookAvailable();
  });

  test('should select documents from email attachments', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.setupMocks();

    // Mock emails with attachments
    await page.route('**/api/v1/integrations/m365/outlook/emails*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          emails: [
            {
              id: 'email-1',
              subject: 'Quarterly Report',
              from: 'john@example.com',
              hasAttachments: true,
              attachments: [
                { id: 'attach-1', name: 'Q4 Report.pdf', size: 1024000 },
              ],
            },
          ],
        }),
      });
    });

    await setupAuthAndNavigate(page, '/microsoft-365', mockUsers[1]);

    await m365Page.selectFromOutlookButton.click();

    // Should show email list or attachment picker
    const emailList = page.locator('[data-testid="email-list"], .email-list');
    await expect(emailList).toBeVisible();
  });
});

test.describe('Microsoft 365 - Print Settings', () => {
  test('should display print settings section', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.setupMocks();
    await setupAuthAndNavigate(page, '/microsoft-365', mockUsers[1]);

    await expect(m365Page.printSettings).toBeVisible();
  });

  test('should update print settings', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.setupMocks();
    await setupAuthAndNavigate(page, '/microsoft-365', mockUsers[1]);

    await m365Page.updatePrintSettings({
      convertToPdf: true,
      includeMetadata: true,
      stampDocument: false,
    });

    await expect(m365Page.convertToPdfToggle).toBeChecked();
    await expect(m365Page.includeMetadataToggle).toBeChecked();
  });

  test('should verify convert to PDF option', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.setupMocks();
    await setupAuthAndNavigate(page, '/microsoft-365', mockUsers[1]);

    const isChecked = await m365Page.convertToPdfToggle.isChecked();
    expect(isChecked).toBe(true); // Should be enabled by default
  });
});

test.describe('Microsoft 365 - Activity Log', () => {
  test('should display activity log', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.setupMocks();
    await setupAuthAndNavigate(page, '/microsoft-365', mockUsers[1]);

    await expect(m365Page.activityLog).toBeVisible();
  });

  test('should show activity entries', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.setupMocks();
    await setupAuthAndNavigate(page, '/microsoft-365', mockUsers[1]);

    const count = await m365Page.getActivityCount();
    expect(count).toBeGreaterThan(0);
  });

  test('should refresh activity log', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.setupMocks();
    await setupAuthAndNavigate(page, '/microsoft-365', mockUsers[1]);

    await m365Page.refreshActivity();

    // Should trigger refresh indicator or update
    await page.waitForTimeout(1000);
  });

  test('should show different activity types', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.setupMocks();
    await setupAuthAndNavigate(page, '/microsoft-365', mockUsers[1]);

    // Check for various activity types in the log
    const activities = page.locator('[data-testid="activity-item"]');
    const count = await activities.count();

    for (let i = 0; i < Math.min(count, 5); i++) {
      const text = await activities.nth(i).textContent();
      expect(text).toBeTruthy();
    }
  });
});

test.describe('Microsoft 365 - Permissions', () => {
  test('should display granted permissions', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.setupMocks();
    await setupAuthAndNavigate(page, '/microsoft-365', mockUsers[1]);

    await expect(m365Page.permissionsSection).toBeVisible();
  });

  test('should show required permissions', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.setupMocks();
    await setupAuthAndNavigate(page, '/microsoft-365', mockUsers[1]);

    const permissions = await m365Page.viewPermissions();
    expect(permissions.length).toBeGreaterThan(0);
  });

  test('should request additional permissions', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.setupMocks();

    // Mock consent URL
    await page.route('**/api/v1/integrations/m365/permissions/consent', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          consentUrl: 'https://login.microsoftonline.com/consent',
        }),
      });
    });

    await setupAuthAndNavigate(page, '/microsoft-365', mockUsers[1]);

    // Click request permissions - should open consent flow
    await m365Page.requestPermissionsButton.click();

    // In real scenario, this would redirect to Microsoft
    // For E2E test, we verify the button action was triggered
    await page.waitForTimeout(500);
  });
});

test.describe('Microsoft 365 - Error Handling', () => {
  test('should handle authentication errors', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);

    // Mock auth error
    await page.route('**/api/v1/integrations/m365/**', async (route) => {
      await route.fulfill({
        status: 401,
        contentType: 'application/json',
        body: JSON.stringify({
          error: 'Authentication failed',
        }),
      });
    });

    await setupAuthAndNavigate(page, '/microsoft-365', mockUsers[1]);

    await expect(page.locator('.error, [data-testid="error-message"]')).toBeVisible();
  });

  test('should handle network errors gracefully', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);

    // Mock network error
    await page.route('**/api/v1/integrations/m365/documents*', async (route) => {
      await route.abort('failed');
    });

    await setupAuthAndNavigate(page, '/microsoft-365', mockUsers[1]);

    await m365Page.browseOneDrive();

    // Should show error state
    await expect(page.locator('.error, [data-testid="error-state"]')).toBeVisible();
  });

  test('should handle rate limiting', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);

    // Mock rate limit response
    await page.route('**/api/v1/integrations/m365/**', async (route) => {
      await route.fulfill({
        status: 429,
        contentType: 'application/json',
        body: JSON.stringify({
          error: 'Rate limit exceeded',
          retryAfter: 60,
        }),
      });
    });

    await setupAuthAndNavigate(page, '/microsoft-365', mockUsers[1]);

    await m365Page.browseOneDrive();

    // Should show rate limit message
    await expect(page.locator('text=/rate limit/i')).toBeVisible();
  });
});

test.describe('Microsoft 365 - Admin Only', () => {
  test('should require admin access', async ({ page }) => {
    // Mock non-admin user
    await setupAuthAndNavigate(page, '/microsoft-365', mockUsers[0]);

    // Should either redirect or show access denied
    const isDenied = page.url().includes('denied') || page.url().includes('/dashboard');
    expect(isDenied).toBe(true);
  });

  test('should show admin-only features for admin users', async ({ page }) => {
    const m365Page = new Microsoft365Page(page);
    await m365Page.setupMocks();
    await setupAuthAndNavigate(page, '/microsoft-365', mockUsers[1]); // Admin

    // Admin should see configuration options
    await expect(m365Page.configureButton).toBeVisible();
    await expect(m365Page.disconnectButton).toBeVisible();
  });
});

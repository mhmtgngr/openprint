/**
 * Microsoft 365 Page Object
 * Handles Microsoft 365 integration configuration and testing
 */
import { Page, Locator, expect } from '@playwright/test';
import { BasePage } from './BasePage';
import { mockApiResponse } from '../helpers';

export interface M365Config {
  enabled: boolean;
  tenantId: string;
  clientId: string;
  clientSecret: string;
  defaultSharePointSite?: string;
  defaultOneDriveFolder?: string;
  autoDiscovery: boolean;
}

export interface M365Document {
  id: string;
  name: string;
  source: 'onedrive' | 'sharepoint' | 'outlook';
  sourceLocation: string;
  url: string;
  size: number;
  createdBy: string;
  createdAt: string;
}

export class Microsoft365Page extends BasePage {
  // Page elements
  readonly heading: Locator;
  readonly connectionStatus: Locator;
  readonly configureButton: Locator;
  readonly testConnectionButton: Locator;
  readonly disconnectButton: Locator;

  // Configuration form
  readonly configForm: Locator;
  readonly tenantIdInput: Locator;
  readonly clientIdInput: Locator;
  readonly clientSecretInput: Locator;
  readonly sharePointSiteInput: Locator;
  readonly oneDriveFolderInput: Locator;
  readonly autoDiscoveryToggle: Locator;
  readonly saveConfigButton: Locator;
  readonly cancelButton: Locator;

  // Document sources
  readonly oneDriveSection: Locator;
  readonly sharePointSection: Locator;
  readonly outlookSection: Locator;
  readonly browseOneDriveButton: Locator;
  readonly browseSharePointButton: Locator;
  readonly selectFromOutlookButton: Locator;

  // Document browser
  readonly documentBrowser: Locator;
  readonly documentList: Locator;
  readonly documentItems: Locator;
  readonly breadcrumb: Locator;
  readonly parentFolderButton: Locator;
  readonly selectDocumentButton: Locator;
  readonly searchDocumentsInput: Locator;

  // Print settings
  readonly printSettings: Locator;
  readonly convertToPdfToggle: Locator;
  readonly includeMetadataToggle: Locator;
  readonly stampDocumentToggle: Locator;
  readonly savePrintSettingsButton: Locator;

  // Activity log
  readonly activityLog: Locator;
  readonly activityItems: Locator;
  readonly refreshActivityButton: Locator;

  // Permissions
  readonly permissionsSection: Locator;
  readonly grantedPermissions: Locator;
  readonly requestPermissionsButton: Locator;

  constructor(page: Page) {
    super(page);

    // Initialize locators
    this.heading = page.locator('h1, [data-testid="m365-heading"]');
    this.connectionStatus = page.locator('[data-testid="connection-status"], .connection-status');
    this.configureButton = page.locator('button:has-text("Configure"), [data-testid="configure-button"]');
    this.testConnectionButton = page.locator('button:has-text("Test Connection"), [data-testid="test-connection"]');
    this.disconnectButton = page.locator('button:has-text("Disconnect"), [data-testid="disconnect-button"]');

    // Configuration form
    this.configForm = page.locator('[data-testid="m365-config-form"], form[data-type="m365-config"]');
    this.tenantIdInput = page.locator('input[name="tenantId"], [data-testid="tenant-id"]');
    this.clientIdInput = page.locator('input[name="clientId"], [data-testid="client-id"]');
    this.clientSecretInput = page.locator('input[name="clientSecret"], [data-testid="client-secret"]');
    this.sharePointSiteInput = page.locator('input[name="sharePointSite"], [data-testid="sharepoint-site"]');
    this.oneDriveFolderInput = page.locator('input[name="oneDriveFolder"], [data-testid="onedrive-folder"]');
    this.autoDiscoveryToggle = page.locator('input[name="autoDiscovery"], [data-testid="auto-discovery"]');
    this.saveConfigButton = page.locator('button[type="submit"]:has-text("Save"), [data-testid="save-config"]');
    this.cancelButton = page.locator('button:has-text("Cancel")');

    // Document sources
    this.oneDriveSection = page.locator('[data-testid="onedrive-section"], .onedrive-section');
    this.sharePointSection = page.locator('[data-testid="sharepoint-section"], .sharepoint-section');
    this.outlookSection = page.locator('[data-testid="outlook-section"], .outlook-section');
    this.browseOneDriveButton = page.locator('button:has-text("Browse OneDrive"), [data-testid="browse-onedrive"]');
    this.browseSharePointButton = page.locator('button:has-text("Browse SharePoint"), [data-testid="browse-sharepoint"]');
    this.selectFromOutlookButton = page.locator('button:has-text("Select from Outlook"), [data-testid="select-outlook"]');

    // Document browser
    this.documentBrowser = page.locator('[data-testid="document-browser"], .document-browser');
    this.documentList = page.locator('[data-testid="document-list"], .document-list');
    this.documentItems = page.locator('[data-testid="document-item"], .document-item');
    this.breadcrumb = page.locator('[data-testid="breadcrumb"], .breadcrumb');
    this.parentFolderButton = page.locator('button:has-text("Parent"), [data-testid="parent-folder"]');
    this.selectDocumentButton = page.locator('button:has-text("Select"), [data-testid="select-document"]');
    this.searchDocumentsInput = page.locator('input[type="search"], [data-testid="search-documents"]');

    // Print settings
    this.printSettings = page.locator('[data-testid="print-settings"], .print-settings');
    this.convertToPdfToggle = page.locator('input[name="convertToPdf"], [data-testid="convert-pdf"]');
    this.includeMetadataToggle = page.locator('input[name="includeMetadata"], [data-testid="include-metadata"]');
    this.stampDocumentToggle = page.locator('input[name="stampDocument"], [data-testid="stamp-document"]');
    this.savePrintSettingsButton = page.locator('button:has-text("Save Settings"), [data-testid="save-print-settings"]');

    // Activity log
    this.activityLog = page.locator('[data-testid="activity-log"], .activity-log');
    this.activityItems = page.locator('[data-testid="activity-item"], .activity-item');
    this.refreshActivityButton = page.locator('button:has-text("Refresh"), [data-testid="refresh-activity"]');

    // Permissions
    this.permissionsSection = page.locator('[data-testid="permissions-section"], .permissions-section');
    this.grantedPermissions = page.locator('[data-testid="granted-permissions"], .granted-permissions');
    this.requestPermissionsButton = page.locator('button:has-text("Request Permissions"), [data-testid="request-permissions"]');
  }

  /**
   * Navigate to Microsoft 365 page
   */
  async goto() {
    await this.goto('/microsoft-365');
    await this.waitForPageLoad();
  }

  /**
   * Setup API mocks for Microsoft 365 page
   */
  async setupMocks() {
    // Mock config status
    await this.page.route('**/api/v1/integrations/m365/config', async (route) => {
      if (route.request().method() === 'GET') {
        await mockApiResponse(route, {
          enabled: true,
          tenantId: 'mock-tenant-id',
          clientId: 'mock-client-id',
          configured: true,
        } as Partial<M365Config>);
      } else if (route.request().method() === 'PUT' || route.request().method() === 'POST') {
        await mockApiResponse(route, {
          message: 'Configuration saved',
        });
      }
    });

    // Mock connection test
    await this.page.route('**/api/v1/integrations/m365/test', async (route) => {
      await mockApiResponse(route, {
        success: true,
        message: 'Connection successful',
        details: {
          tenantName: 'Mock Tenant',
          user: 'test@example.com',
        },
      });
    });

    // Mock document list
    await this.page.route('**/api/v1/integrations/m365/documents*', async (route) => {
      await mockApiResponse(route, {
        documents: [
          {
            id: 'doc-1',
            name: 'Report.pdf',
            source: 'onedrive',
            sourceLocation: '/Documents',
            url: 'https://graph.microsoft.com/v1.0/me/drive/items/doc-1',
            size: 1024000,
            createdBy: 'Test User',
            createdAt: '2024-02-27T10:00:00Z',
          },
          {
            id: 'doc-2',
            name: 'Presentation.pptx',
            source: 'sharepoint',
            sourceLocation: '/Shared Documents',
            url: 'https://graph.microsoft.com/v1.0/sites/site/drive/items/doc-2',
            size: 5120000,
            createdBy: 'Jane Smith',
            createdAt: '2024-02-26T14:00:00Z',
          },
        ],
        total: 2,
      });
    });

    // Mock print settings
    await this.page.route('**/api/v1/integrations/m365/print-settings', async (route) => {
      if (route.request().method() === 'GET') {
        await mockApiResponse(route, {
          convertToPdf: true,
          includeMetadata: false,
          stampDocument: false,
        });
      } else if (route.request().method() === 'PUT' || route.request().method() === 'PATCH') {
        await mockApiResponse(route, {
          message: 'Settings saved',
        });
      }
    });

    // Mock activity log
    await this.page.route('**/api/v1/integrations/m365/activity*', async (route) => {
      await mockApiResponse(route, {
        activities: [
          {
            id: 'act-1',
            action: 'document_printed',
            documentName: 'Report.pdf',
            source: 'onedrive',
            user: 'Test User',
            timestamp: '2024-02-27T10:00:00Z',
          },
          {
            id: 'act-2',
            action: 'connection_tested',
            user: 'Test User',
            timestamp: '2024-02-27T09:00:00Z',
          },
        ],
        total: 2,
      });
    });

    // Mock permissions
    await this.page.route('**/api/v1/integrations/m365/permissions', async (route) => {
      await mockApiResponse(route, {
        permissions: ['Files.Read', 'Files.Read.All', 'Sites.Read.All'],
        consentUrl: 'https://login.microsoftonline.com/consent',
      });
    });

    // Mock disconnect
    await this.page.route('**/api/v1/integrations/m365/disconnect', async (route) => {
      await mockApiResponse(route, {
        message: 'Disconnected successfully',
      });
    });
  }

  /**
   * Verify Microsoft 365 page is loaded
   */
  async isLoaded(): Promise<boolean> {
    await this.heading.waitFor({ state: 'visible', timeout: 5000 });
    return await this.heading.isVisible();
  }

  /**
   * Get connection status
   */
  async getConnectionStatus(): Promise<string> {
    const statusText = await this.connectionStatus.textContent();
    return statusText || 'unknown';
  }

  /**
   * Verify connection status
   */
  async verifyConnectionStatus(expectedStatus: 'connected' | 'disconnected' | 'error') {
    const statusClass = await this.connectionStatus.getAttribute('data-status');
    expect(statusClass).toBe(expectedStatus);
  }

  /**
   * Open configuration form
   */
  async openConfigForm() {
    await this.configureButton.click();
    await expect(this.configForm).toBeVisible();
  }

  /**
   * Fill configuration form
   */
  async fillConfigForm(config: Partial<M365Config>) {
    if (config.tenantId) {
      await this.tenantIdInput.fill(config.tenantId);
    }

    if (config.clientId) {
      await this.clientIdInput.fill(config.clientId);
    }

    if (config.clientSecret) {
      await this.clientSecretInput.fill(config.clientSecret);
    }

    if (config.defaultSharePointSite) {
      await this.sharePointSiteInput.fill(config.defaultSharePointSite);
    }

    if (config.defaultOneDriveFolder) {
      await this.oneDriveFolderInput.fill(config.defaultOneDriveFolder);
    }

    if (config.autoDiscovery !== undefined) {
      if (config.autoDiscovery) {
        await this.autoDiscoveryToggle.check();
      } else {
        await this.autoDiscoveryToggle.uncheck();
      }
    }
  }

  /**
   * Save configuration
   */
  async saveConfiguration() {
    await this.saveConfigButton.click();
    await expect(this.configForm).not.toBeVisible();
    await this.verifyToast('Configuration saved', 'success');
  }

  /**
   * Test connection
   */
  async testConnection() {
    await this.testConnectionButton.click();
    await this.verifyToast('Connection successful', 'success');
  }

  /**
   * Disconnect integration
   */
  async disconnect() {
    await this.disconnectButton.click();

    const confirmButton = this.page.locator('button:has-text("Confirm"), button:has-text("Disconnect")');
    if (await confirmButton.isVisible()) {
      await confirmButton.click();
    }

    await this.verifyToast('Disconnected', 'success');
  }

  /**
   * Browse OneDrive documents
   */
  async browseOneDrive(folderPath?: string) {
    await this.browseOneDriveButton.click();
    await expect(this.documentBrowser).toBeVisible();

    if (folderPath) {
      // Navigate to specific folder
      await this.navigateToFolder(folderPath);
    }
  }

  /**
   * Browse SharePoint documents
   */
  async browseSharePoint(site?: string, folderPath?: string) {
    await this.browseSharePointButton.click();
    await expect(this.documentBrowser).toBeVisible();

    if (site) {
      // Select site first
      const siteSelect = this.page.locator('select[name="site"]');
      await siteSelect.selectOption(site);
    }

    if (folderPath) {
      await this.navigateToFolder(folderPath);
    }
  }

  /**
   * Navigate to folder in document browser
   */
  async navigateToFolder(folderName: string) {
    const folder = this.documentItems.filter({ hasText: folderName });
    await folder.dblclick();
  }

  /**
   * Navigate to parent folder
   */
  async navigateToParent() {
    await this.parentFolderButton.click();
  }

  /**
   * Select document for printing
   */
  async selectDocument(documentName: string) {
    const document = this.documentItems.filter({ hasText: documentName });
    await document.locator('input[type="checkbox"]').check();
  }

  /**
   * Confirm document selection
   */
  async confirmSelection() {
    await this.selectDocumentButton.click();
    await expect(this.documentBrowser).not.toBeVisible();
  }

  /**
   * Search documents
   */
  async searchDocuments(query: string) {
    await this.searchDocumentsInput.fill(query);
    await this.page.waitForTimeout(500);
  }

  /**
   * Get document count
   */
  async getDocumentCount(): Promise<number> {
    return await this.documentItems.count();
  }

  /**
   * Update print settings
   */
  async updatePrintSettings(settings: {
    convertToPdf?: boolean;
    includeMetadata?: boolean;
    stampDocument?: boolean;
  }) {
    if (settings.convertToPdf !== undefined) {
      if (settings.convertToPdf) {
        await this.convertToPdfToggle.check();
      } else {
        await this.convertToPdfToggle.uncheck();
      }
    }

    if (settings.includeMetadata !== undefined) {
      if (settings.includeMetadata) {
        await this.includeMetadataToggle.check();
      } else {
        await this.includeMetadataToggle.uncheck();
      }
    }

    if (settings.stampDocument !== undefined) {
      if (settings.stampDocument) {
        await this.stampDocumentToggle.check();
      } else {
        await this.stampDocumentToggle.uncheck();
      }
    }

    await this.savePrintSettingsButton.click();
    await this.verifyToast('Settings saved', 'success');
  }

  /**
   * Get activity log count
   */
  async getActivityCount(): Promise<number> {
    return await this.activityItems.count();
  }

  /**
   * Refresh activity log
   */
  async refreshActivity() {
    await this.refreshActivityButton.click();
    await this.page.waitForTimeout(1000);
  }

  /**
   * View granted permissions
   */
  async viewPermissions(): Promise<string[]> {
    const permissionText = await this.grantedPermissions.textContent();
    return permissionText?.split(',').map((p) => p.trim()) || [];
  }

  /**
   * Request additional permissions
   */
  async requestPermissions() {
    await this.requestPermissionsButton.click();

    // Should open Microsoft consent URL in new window or redirect
    // For E2E testing, we'll mock this behavior
  }

  /**
   * Verify OneDrive section is available
   */
  async verifyOneDriveAvailable() {
    await expect(this.oneDriveSection).toBeVisible();
  }

  /**
   * Verify SharePoint section is available
   */
  async verifySharePointAvailable() {
    await expect(this.sharePointSection).toBeVisible();
  }

  /**
   * Verify Outlook section is available
   */
  async verifyOutlookAvailable() {
    await expect(this.outlookSection).toBeVisible();
  }

  /**
   * Get breadcrumb path
   */
  async getBreadcrumbPath(): Promise<string[]> {
    const breadcrumbItems = this.breadcrumb.locator('span, a');
    const paths: string[] = [];
    const count = await breadcrumbItems.count();

    for (let i = 0; i < count; i++) {
      const text = await breadcrumbItems.nth(i).textContent();
      if (text) paths.push(text.trim());
    }

    return paths;
  }

  /**
   * Verify document source badge
   */
  async verifyDocumentSource(documentName: string, source: 'onedrive' | 'sharepoint' | 'outlook') {
    const document = this.documentItems.filter({ hasText: documentName });
    const sourceBadge = document.locator('[data-testid="source-badge"], .source-badge');
    await expect(sourceBadge).toHaveText(new RegExp(source, 'i'));
  }

  /**
   * Get document size
   */
  async getDocumentSize(documentName: string): Promise<string> {
    const document = this.documentItems.filter({ hasText: documentName });
    const sizeElement = document.locator('[data-testid="document-size"], .document-size');
    return await sizeElement.textContent() || '';
  }

  /**
   * Print selected document
   */
  async printDocument(documentName: string) {
    await this.selectDocument(documentName);
    await this.confirmSelection();

    // Should open print job modal or navigate to job creation
    const printModal = this.page.locator('[data-testid="print-modal"], .print-modal');
    await expect(printModal).toBeVisible();
  }
}

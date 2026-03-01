import { type Page, type Locator, expect } from '@playwright/test';
import { BasePage } from './BasePage';

/**
 * Microsoft 365 Integration Page Object
 * Handles M365 integration configuration, testing, and management
 */
export class Microsoft365Page extends BasePage {
  // Page heading and sections
  readonly heading: Locator;
  readonly overviewSection: Locator;
  readonly configurationSection: Locator;
  readonly testSection: Locator;
  readonly permissionsSection: Locator;
  readonly syncStatusSection: Locator;

  // Configuration form elements
  readonly tenantIdInput: Locator;
  readonly clientIdInput: Locator;
  readonly clientSecretInput: Locator;
  readonly domainInput: Locator;
  readonly enableToggle: Locator;

  // Connection status
  readonly connectionStatusBadge: Locator;
  readonly connectionStatusText: Locator;
  readonly lastSyncTime: Locator;

  // Action buttons
  readonly saveButton: Locator;
  readonly testConnectionButton: Locator;
  readonly syncNowButton: Locator;
  readonly disconnectButton: Locator;
  readonly reauthorizeButton: Locator;

  // Permissions management
  readonly permissionsList: Locator;
  readonly requestPermissionButton: Locator;
  readonly grantPermissionButton: Locator;
  readonly revokePermissionButton: Locator;

  // OneDrive integration
  readonly oneDriveSection: Locator;
  readonly oneDriveToggle: Locator;
  readonly oneDriveFolderInput: Locator;
  readonly oneDriveTestButton: Locator;

  // SharePoint integration
  readonly sharePointSection: Locator;
  readonly sharePointToggle: Locator;
  readonly sharePointSiteInput: Locator;
  readonly sharePointLibraryInput: Locator;
  readonly sharePointTestButton: Locator;

  // Email to Print (M365)
  readonly emailToPrintSection: Locator;
  readonly emailToPrintToggle: Locator;
  readonly emailAddressDisplay: Locator;
  readonly allowedSendersInput: Locator;
  readonly addAllowedSenderButton: Locator;
  readonly allowedSendersList: Locator;

  // User and group sync
  readonly userSyncSection: Locator;
  readonly userSyncToggle: Locator;
  readonly syncIntervalSelect: Locator;
  readonly syncGroupsInput: Locator;
  readonly addSyncGroupButton: Locator;
  readonly syncGroupsList: Locator;

  // Activity logs
  readonly activityLogsSection: Locator;
  readonly activityLogsList: Locator;
  readonly viewAllLogsButton: Locator;

  constructor(page: Page) {
    super(page);

    // Page heading and sections
    this.heading = page.getByRole('heading', { name: /microsoft 365|m365|office 365/i });
    this.overviewSection = page.locator('[data-testid="m365-overview"], section:has-text("Overview")');
    this.configurationSection = page.locator('[data-testid="m365-configuration"], section:has-text("Configuration")');
    this.testSection = page.locator('[data-testid="m365-test"], section:has-text("Test Connection")');
    this.permissionsSection = page.locator('[data-testid="m365-permissions"], section:has-text("Permissions")');
    this.syncStatusSection = page.locator('[data-testid="m365-sync-status"], section:has-text("Sync Status")');

    // Configuration form elements
    this.tenantIdInput = page.getByLabel(/tenant id/i);
    this.clientIdInput = page.getByLabel(/client id|application id/i);
    this.clientSecretInput = page.getByLabel(/client secret/i);
    this.domainInput = page.getByLabel(/domain|onmicrosoft/i);
    this.enableToggle = page.getByRole('switch', { name: /enable microsoft 365|enable m365/i });

    // Connection status
    this.connectionStatusBadge = page.locator('[data-testid="connection-status-badge"]');
    this.connectionStatusText = page.locator('[data-testid="connection-status-text"]');
    this.lastSyncTime = page.locator('[data-testid="last-sync-time"]');

    // Action buttons
    this.saveButton = page.getByRole('button', { name: /save|save configuration/i });
    this.testConnectionButton = page.getByRole('button', { name: /test connection|test/i });
    this.syncNowButton = page.getByRole('button', { name: /sync now|sync/i });
    this.disconnectButton = page.getByRole('button', { name: /disconnect|remove/i });
    this.reauthorizeButton = page.getByRole('button', { name: /reauthorize|re-authenticate/i });

    // Permissions management
    this.permissionsList = page.locator('[data-testid="permissions-list"]');
    this.requestPermissionButton = page.getByRole('button', { name: /request permission|grant/i });
    this.grantPermissionButton = page.getByRole('button', { name: /grant/i });
    this.revokePermissionButton = page.getByRole('button', { name: /revoke/i });

    // OneDrive integration
    this.oneDriveSection = page.locator('[data-testid="onedrive-section"], section:has-text("OneDrive")');
    this.oneDriveToggle = page.locator('[data-testid="onedrive-toggle"]').or(page.getByRole('switch', { name: /onedrive/i }));
    this.oneDriveFolderInput = page.getByLabel(/folder|destination folder/i);
    this.oneDriveTestButton = page.locator('[data-testid="onedrive-test"]').or(page.getByRole('button', { name: /test onedrive/i }));

    // SharePoint integration
    this.sharePointSection = page.locator('[data-testid="sharepoint-section"], section:has-text("SharePoint")');
    this.sharePointToggle = page.locator('[data-testid="sharepoint-toggle"]').or(page.getByRole('switch', { name: /sharepoint/i }));
    this.sharePointSiteInput = page.getByLabel(/sharepoint site|site url/i);
    this.sharePointLibraryInput = page.getByLabel(/document library|library/i);
    this.sharePointTestButton = page.locator('[data-testid="sharepoint-test"]').or(page.getByRole('button', { name: /test sharepoint/i }));

    // Email to Print (M365)
    this.emailToPrintSection = page.locator('[data-testid="email-to-print-section"], section:has-text("Email to Print")');
    this.emailToPrintToggle = page.locator('[data-testid="email-to-print-toggle"]');
    this.emailAddressDisplay = page.locator('[data-testid="email-address-display"]');
    this.allowedSendersInput = page.getByLabel(/allowed senders|allowed domains/i);
    this.addAllowedSenderButton = page.getByRole('button', { name: /add|allow/i });
    this.allowedSendersList = page.locator('[data-testid="allowed-senders-list"]');

    // User and group sync
    this.userSyncSection = page.locator('[data-testid="user-sync-section"], section:has-text("User Sync")');
    this.userSyncToggle = page.locator('[data-testid="user-sync-toggle"]');
    this.syncIntervalSelect = page.getByLabel(/sync interval|sync frequency/i);
    this.syncGroupsInput = page.getByLabel(/groups|security groups/i);
    this.addSyncGroupButton = page.getByRole('button', { name: /add group/i });
    this.syncGroupsList = page.locator('[data-testid="sync-groups-list"]');

    // Activity logs
    this.activityLogsSection = page.locator('[data-testid="activity-logs-section"], section:has-text("Activity")');
    this.activityLogsList = page.locator('[data-testid="activity-logs-list"]');
    this.viewAllLogsButton = page.getByRole('button', { name: /view all logs/i });
  }

  /**
   * Navigate to Microsoft 365 settings page
   */
  async navigate(): Promise<void> {
    await this.goto('/settings/microsoft-365');
  }

  /**
   * Verify Microsoft 365 page is loaded
   */
  async isLoaded(): Promise<boolean> {
    await this.page.waitForLoadState('networkidle');
    return await this.heading.isVisible();
  }

  /**
   * Configure Microsoft 365 integration
   */
  async configure(config: {
    tenantId: string;
    clientId: string;
    clientSecret: string;
    domain: string;
  }): Promise<void> {
    await this.tenantIdInput.fill(config.tenantId);
    await this.clientIdInput.fill(config.clientId);
    await this.clientSecretInput.fill(config.clientSecret);
    await this.domainInput.fill(config.domain);
    await this.saveButton.click();
  }

  /**
   * Enable Microsoft 365 integration
   */
  async enableIntegration(): Promise<void> {
    await this.enableToggle.click();
    await this.saveButton.click();
  }

  /**
   * Disable Microsoft 365 integration
   */
  async disableIntegration(): Promise<void> {
    if (await this.enableToggle.isChecked()) {
      await this.enableToggle.click();
    }
    await this.saveButton.click();
  }

  /**
   * Test connection to Microsoft 365
   */
  async testConnection(): Promise<void> {
    await this.testConnectionButton.click();
    // Wait for test result
    await this.page.waitForTimeout(2000);
  }

  /**
   * Get connection status
   */
  async getConnectionStatus(): Promise<string> {
    return await this.connectionStatusText.textContent() || '';
  }

  /**
   * Check if connection is healthy
   */
  async isConnectionHealthy(): Promise<boolean> {
    const status = await this.getConnectionStatus();
    return status.toLowerCase().includes('connected') || status.toLowerCase().includes('healthy');
  }

  /**
   * Trigger manual sync
   */
  async triggerSync(): Promise<void> {
    await this.syncNowButton.click();
    // Wait for sync to complete
    await expect(this.page.getByText(/sync complete|synced/i)).toBeVisible({ timeout: 30000 });
  }

  /**
   * Disconnect Microsoft 365 integration
   */
  async disconnect(): Promise<void> {
    await this.disconnectButton.click();
    // Confirm disconnection
    const confirmButton = this.page.getByRole('button', { name: /confirm|disconnect/i });
    if (await confirmButton.isVisible()) {
      await confirmButton.click();
    }
  }

  /**
   * Reauthorize Microsoft 365 access
   */
  async reauthorize(): Promise<void> {
    await this.reauthorizeButton.click();
    // This would typically open a new window for OAuth
    // Implementation depends on how the app handles reauthorization
  }

  /**
   * Enable OneDrive integration
   */
  async enableOneDrive(folder: string): Promise<void> {
    await this.oneDriveToggle.click();
    await this.oneDriveFolderInput.fill(folder);
    await this.saveButton.click();
  }

  /**
   * Disable OneDrive integration
   */
  async disableOneDrive(): Promise<void> {
    if (await this.oneDriveToggle.isChecked()) {
      await this.oneDriveToggle.click();
    }
    await this.saveButton.click();
  }

  /**
   * Test OneDrive connection
   */
  async testOneDriveConnection(): Promise<void> {
    await this.oneDriveTestButton.click();
    await this.page.waitForTimeout(2000);
  }

  /**
   * Enable SharePoint integration
   */
  async enableSharePoint(config: { siteUrl: string; library: string }): Promise<void> {
    await this.sharePointToggle.click();
    await this.sharePointSiteInput.fill(config.siteUrl);
    await this.sharePointLibraryInput.fill(config.library);
    await this.saveButton.click();
  }

  /**
   * Disable SharePoint integration
   */
  async disableSharePoint(): Promise<void> {
    if (await this.sharePointToggle.isChecked()) {
      await this.sharePointToggle.click();
    }
    await this.saveButton.click();
  }

  /**
   * Test SharePoint connection
   */
  async testSharePointConnection(): Promise<void> {
    await this.sharePointTestButton.click();
    await this.page.waitForTimeout(2000);
  }

  /**
   * Enable Email to Print
   */
  async enableEmailToPrint(): Promise<void> {
    await this.emailToPrintToggle.click();
    await this.saveButton.click();
  }

  /**
   * Get the Email to Print email address
   */
  async getEmailToPrintAddress(): Promise<string> {
    return await this.emailAddressDisplay.textContent() || '';
  }

  /**
   * Add allowed sender for Email to Print
   */
  async addAllowedSender(sender: string): Promise<void> {
    await this.allowedSendersInput.fill(sender);
    await this.addAllowedSenderButton.click();
  }

  /**
   * Remove allowed sender for Email to Print
   */
  async removeAllowedSender(sender: string): Promise<void> {
    const senderElement = this.allowedSendersList.getByText(sender);
    const removeButton = senderElement.locator('../..').getByRole('button', { name: /remove|delete/i });
    await removeButton.click();
  }

  /**
   * Enable user sync
   */
  async enableUserSync(interval: string): Promise<void> {
    await this.userSyncToggle.click();
    await this.syncIntervalSelect.selectOption(interval);
    await this.saveButton.click();
  }

  /**
   * Add sync group
   */
  async addSyncGroup(groupId: string): Promise<void> {
    await this.syncGroupsInput.fill(groupId);
    await this.addSyncGroupButton.click();
  }

  /**
   * Remove sync group
   */
  async removeSyncGroup(groupId: string): Promise<void> {
    const groupElement = this.syncGroupsList.getByText(groupId);
    const removeButton = groupElement.locator('../..').getByRole('button', { name: /remove|delete/i });
    await removeButton.click();
  }

  /**
   * Get sync groups list
   */
  async getSyncGroups(): Promise<string[]> {
    const groups: string[] = [];
    const groupElements = await this.syncGroupsList.locator('[data-testid="sync-group-item"]').all();
    for (const element of groupElements) {
      groups.push(await element.textContent() || '');
    }
    return groups;
  }

  /**
   * Get last sync time
   */
  async getLastSyncTime(): Promise<string> {
    return await this.lastSyncTime.textContent() || '';
  }

  /**
   * View all activity logs
   */
  async viewAllActivityLogs(): Promise<void> {
    await this.viewAllLogsButton.click();
    await this.waitForNavigation();
  }

  /**
   * Get activity logs count
   */
  async getActivityLogsCount(): Promise<number> {
    return await this.activityLogsList.locator('> div, > tr').count();
  }

  /**
   * Verify permission is granted
   */
  async hasPermission(permissionName: string): Promise<boolean> {
    const permissionElement = this.permissionsList.getByText(permissionName);
    const count = await permissionElement.count();
    return count > 0;
  }

  /**
   * Request/grant permission
   */
  async requestPermission(permissionName: string): Promise<void> {
    const permissionElement = this.permissionsList.getByText(permissionName);
    const grantButton = permissionElement.locator('../..').getByRole('button', { name: /grant|request/i });
    await grantButton.click();
  }

  /**
   * Verify configuration is saved
   */
  async verifyConfigurationSaved(): Promise<void> {
    await expect(this.page.getByText(/saved|updated|configuration saved/i)).toBeVisible();
  }

  /**
   * Verify test connection result
   */
  async verifyTestConnectionResult(expectedStatus: 'success' | 'error'): Promise<void> {
    if (expectedStatus === 'success') {
      await expect(this.page.getByText(/connection successful|connected|test passed/i)).toBeVisible();
    } else {
      await expect(this.page.getByText(/connection failed|error|unable to connect/i)).toBeVisible();
    }
  }
}

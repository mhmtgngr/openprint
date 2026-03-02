/**
 * Settings Page Object
 * Handles user profile, security, and preferences
 */
import { Page, Locator, expect } from '@playwright/test';
import { BasePage } from './BasePage';
import { mockApiResponse, mockUsers } from '../helpers';

export class SettingsPage extends BasePage {
  // Page elements
  readonly heading: Locator;
  readonly tabs: Locator;
  readonly profileTab: Locator;
  readonly securityTab: Locator;
  readonly sessionsTab: Locator;
  readonly preferencesTab: Locator;

  // Profile section
  readonly profileForm: Locator;
  readonly nameInput: Locator;
  readonly emailInput: Locator;
  readonly avatarUpload: Locator;
  readonly avatarImage: Locator;
  readonly saveProfileButton: Locator;
  readonly changePasswordButton: Locator;

  // Password change form
  readonly currentPasswordInput: Locator;
  readonly newPasswordInput: Locator;
  readonly confirmNewPasswordInput: Locator;
  readonly updatePasswordButton: Locator;

  // Security section
  readonly mfaSection: Locator;
  readonly enableMfaButton: Locator;
  readonly disableMfaButton: Locator;
  readonly mfaQrCode: Locator;
  readonly mfaBackupCodes: Locator;
  readonly mfaVerifyInput: Locator;
  readonly mfaConfirmButton: Locator;
  readonly apiKeysSection: Locator;
  readonly createApiKeyButton: Locator;
  readonly apiKeysList: Locator;

  // Sessions section
  readonly sessionsList: Locator;
  readonly currentSession: Locator;
  readonly revokeSessionButton: Locator;
  readonly revokeAllSessionsButton: Locator;

  // Preferences section
  readonly themeSelect: Locator;
  readonly languageSelect: Locator;
  readonly timezoneSelect: Locator;
  readonly dateFormatSelect: Locator;
  readonly timezoneDisplay: Locator;
  readonly emailNotificationsToggle: Locator;
  readonly pushNotificationsToggle: Locator;
  readonly weeklyDigestToggle: Locator;
  readonly savePreferencesButton: Locator;

  // Notification settings by type
  readonly jobCompletedNotification: Locator;
  readonly jobFailedNotification: Locator;
  readonly printerOfflineNotification: Locator;
  readonly securityAlertNotification: Locator;

  constructor(page: Page) {
    super(page);

    // Initialize locators
    this.heading = page.locator('h1, [data-testid="settings-heading"]');
    this.tabs = page.locator('[data-testid="settings-tabs"], .settings-tabs');
    this.profileTab = page.locator('button:has-text("Profile"), [data-testid="tab-profile"]');
    this.securityTab = page.locator('button:has-text("Security"), [data-testid="tab-security"]');
    this.sessionsTab = page.locator('button:has-text("Sessions"), [data-testid="tab-sessions"]');
    this.preferencesTab = page.locator('button:has-text("Preferences"), [data-testid="tab-preferences"]');

    // Profile
    this.profileForm = page.locator('[data-testid="profile-form"], form[data-type="profile"]');
    this.nameInput = page.locator('input[name="name"], [data-testid="name-input"]');
    this.emailInput = page.locator('input[name="email"], [data-testid="email-input"]');
    this.avatarUpload = page.locator('input[type="file"][accept*="image"], [data-testid="avatar-upload"]');
    this.avatarImage = page.locator('[data-testid="avatar"], .avatar-image');
    this.saveProfileButton = page.locator('button:has-text("Save Profile"), [data-testid="save-profile"]');
    this.changePasswordButton = page.locator('button:has-text("Change Password"), [data-testid="change-password"]');

    // Password change
    this.currentPasswordInput = page.locator('input[name="currentPassword"], [data-testid="current-password"]');
    this.newPasswordInput = page.locator('input[name="newPassword"], [data-testid="new-password"]');
    this.confirmNewPasswordInput = page.locator('input[name="confirmPassword"], [data-testid="confirm-new-password"]');
    this.updatePasswordButton = page.locator('button:has-text("Update Password"), [data-testid="update-password"]');

    // Security
    this.mfaSection = page.locator('[data-testid="mfa-section"], .mfa-section');
    this.enableMfaButton = page.locator('button:has-text("Enable"), [data-testid="enable-mfa"]');
    this.disableMfaButton = page.locator('button:has-text("Disable"), [data-testid="disable-mfa"]');
    this.mfaQrCode = page.locator('[data-testid="mfa-qr"], .mfa-qr-code');
    this.mfaBackupCodes = page.locator('[data-testid="backup-codes"], .backup-codes');
    this.mfaVerifyInput = page.locator('input[name="mfaCode"], [data-testid="mfa-verify-input"]');
    this.mfaConfirmButton = page.locator('button:has-text("Verify"), [data-testid="mfa-confirm"]');
    this.apiKeysSection = page.locator('[data-testid="api-keys-section"], .api-keys-section');
    this.createApiKeyButton = page.locator('button:has-text("Create API Key"), [data-testid="create-api-key"]');
    this.apiKeysList = page.locator('[data-testid="api-keys-list"], .api-keys-list');

    // Sessions
    this.sessionsList = page.locator('[data-testid="sessions-list"], .sessions-list');
    this.currentSession = page.locator('[data-testid="current-session"], .session-item.current');
    this.revokeSessionButton = page.locator('button:has-text("Revoke"), [data-testid="revoke-session"]');
    this.revokeAllSessionsButton = page.locator('button:has-text("Revoke All"), [data-testid="revoke-all-sessions"]');

    // Preferences
    this.themeSelect = page.locator('select[name="theme"], [data-testid="theme-select"]');
    this.languageSelect = page.locator('select[name="language"], [data-testid="language-select"]');
    this.timezoneSelect = page.locator('select[name="timezone"], [data-testid="timezone-select"]');
    this.dateFormatSelect = page.locator('select[name="dateFormat"], [data-testid="date-format-select"]');
    this.timezoneDisplay = page.locator('[data-testid="timezone-display"], .timezone-display');
    this.emailNotificationsToggle = page.locator('input[name="emailNotifications"], [data-testid="email-notifications"]');
    this.pushNotificationsToggle = page.locator('input[name="pushNotifications"], [data-testid="push-notifications"]');
    this.weeklyDigestToggle = page.locator('input[name="weeklyDigest"], [data-testid="weekly-digest"]');
    this.savePreferencesButton = page.locator('button:has-text("Save Preferences"), [data-testid="save-preferences"]');

    // Notification settings by type
    this.jobCompletedNotification = page.locator('input[name="notifyJobCompleted"], [data-testid="notify-job-completed"]');
    this.jobFailedNotification = page.locator('input[name="notifyJobFailed"], [data-testid="notify-job-failed"]');
    this.printerOfflineNotification = page.locator('input[name="notifyPrinterOffline"], [data-testid="notify-printer-offline"]');
    this.securityAlertNotification = page.locator('input[name="notifySecurityAlert"], [data-testid="notify-security-alert"]');
  }

  /**
   * Navigate to settings page
   */
  async goto() {
    await this.goto('/settings');
    await this.waitForPageLoad();
  }

  /**
   * Setup API mocks for settings page
   */
  async setupMocks() {
    // Mock user profile
    await this.page.route('**/api/v1/users/me', async (route) => {
      await mockApiResponse(route, {
        ...mockUsers[0],
        avatar: '/avatar/default.png',
        preferences: {
          theme: 'light',
          language: 'en',
          timezone: 'America/New_York',
          dateFormat: 'MM/DD/YYYY',
        },
      });
    });

    // Mock profile update
    await this.page.route('**/api/v1/users/me', async (route) => {
      if (route.request().method() === 'PATCH' || route.request().method() === 'PUT') {
        await mockApiResponse(route, {
          ...mockUsers[0],
          message: 'Profile updated successfully',
        });
      }
    });

    // Mock password change
    await this.page.route('**/api/v1/users/change-password', async (route) => {
      await mockApiResponse(route, {
        message: 'Password changed successfully',
      });
    });

    // Mock MFA setup
    await this.page.route('**/api/v1/users/mfa/setup', async (route) => {
      await mockApiResponse(route, {
        secret: 'JBSWY3DPEHPK3PXP',
        qrCode: 'data:image/png;base64,mockqr',
        backupCodes: ['123456', '234567', '345678', '456789', '567890'],
      });
    });

    // Mock MFA verify
    await this.page.route('**/api/v1/users/mfa/verify', async (route) => {
      await mockApiResponse(route, {
        message: 'MFA enabled successfully',
      });
    });

    // Mock MFA disable
    await this.page.route('**/api/v1/users/mfa/disable', async (route) => {
      await mockApiResponse(route, {
        message: 'MFA disabled',
      });
    });

    // Mock sessions list
    await this.page.route('**/api/v1/users/sessions', async (route) => {
      await mockApiResponse(route, {
        sessions: [
          {
            id: 'current-session',
            device: 'Chrome on Windows',
            ip: '192.168.1.100',
            current: true,
            createdAt: new Date().toISOString(),
            lastActive: new Date().toISOString(),
          },
          {
            id: 'other-session',
            device: 'Safari on iPhone',
            ip: '192.168.1.101',
            current: false,
            createdAt: new Date(Date.now() - 86400000).toISOString(),
            lastActive: new Date(Date.now() - 3600000).toISOString(),
          },
        ],
      });
    });

    // Mock revoke session
    await this.page.route('**/api/v1/users/sessions/*/revoke', async (route) => {
      await mockApiResponse(route, {
        message: 'Session revoked',
      });
    });

    // Mock API keys list
    await this.page.route('**/api/v1/users/api-keys', async (route) => {
      await mockApiResponse(route, {
        keys: [
          {
            id: 'key-1',
            name: 'Test Key',
            createdAt: '2024-01-01T00:00:00Z',
            lastUsed: '2024-02-27T00:00:00Z',
            prefix: 'op_test_...',
          },
        ],
      });
    });

    // Mock create API key
    await this.page.route('**/api/v1/users/api-keys', async (route) => {
      if (route.request().method() === 'POST') {
        await mockApiResponse(route, {
          key: 'op_test_' + Math.random().toString(36).substring(7),
        });
      }
    });

    // Mock avatar upload
    await this.page.route('**/api/v1/users/avatar', async (route) => {
      if (route.request().method() === 'POST') {
        await mockApiResponse(route, {
          avatar: '/avatar/new-avatar.png',
        });
      }
    });
  }

  /**
   * Verify settings page is loaded
   */
  async isLoaded(): Promise<boolean> {
    await this.heading.waitFor({ state: 'visible', timeout: 5000 });
    return await this.heading.isVisible();
  }

  /**
   * Navigate to specific tab
   */
  async navigateToTab(tab: 'profile' | 'security' | 'sessions' | 'preferences') {
    const tabButton = this.tabs.locator(`button:has-text("${tab.charAt(0).toUpperCase() + tab.slice(1)}"), [data-testid="tab-${tab}"]`);
    await tabButton.click();
  }

  /**
   * Update profile information
   */
  async updateProfile(data: { name?: string; email?: string }) {
    await this.navigateToTab('profile');

    if (data.name) {
      await this.nameInput.fill(data.name);
    }

    if (data.email) {
      await this.emailInput.fill(data.email);
    }

    await this.saveProfileButton.click();
    await this.verifyToast('Profile updated', 'success');
  }

  /**
   * Upload avatar
   */
  async uploadAvatar(filePath: string) {
    await this.navigateToTab('profile');
    await this.avatarUpload.setInputFiles(filePath);
    await this.verifyToast('Avatar uploaded', 'success');
  }

  /**
   * Change password
   */
  async changePassword(current: string, newPass: string, confirm: string) {
    await this.navigateToTab('profile');

    await this.changePasswordButton.click();

    await this.currentPasswordInput.fill(current);
    await this.newPasswordInput.fill(newPass);
    await this.confirmNewPasswordInput.fill(confirm);

    await this.updatePasswordButton.click();
    await this.verifyToast('Password changed', 'success');
  }

  /**
   * Enable MFA
   */
  async enableMFA(verificationCode: string) {
    await this.navigateToTab('security');

    await this.enableMfaButton.click();

    // Wait for QR code to be displayed
    await expect(this.mfaQrCode).toBeVisible();

    // Enter verification code
    await this.mfaVerifyInput.fill(verificationCode);
    await this.mfaConfirmButton.click();

    await this.verifyToast('MFA enabled', 'success');
  }

  /**
   * Disable MFA
   */
  async disableMFA() {
    await this.navigateToTab('security');
    await this.disableMfaButton.click();

    // Confirm if prompted
    const confirmButton = this.page.locator('button:has-text("Confirm")');
    if (await confirmButton.isVisible()) {
      await confirmButton.click();
    }

    await this.verifyToast('MFA disabled', 'success');
  }

  /**
   * Get MFA backup codes
   */
  async getBackupCodes(): Promise<string[]> {
    await this.navigateToTab('security');

    // View or regenerate backup codes
    const viewCodesButton = this.page.locator('button:has-text("View Codes"), button:has-text("Generate Codes")');
    await viewCodesButton.click();

    await expect(this.mfaBackupCodes).toBeVisible();

    const codesText = await this.mfaBackupCodes.textContent();
    return codesText?.split('\n').map((c) => c.trim()).filter(Boolean) || [];
  }

  /**
   * Create API key
   */
  async createAPIKey(name: string): Promise<string> {
    await this.navigateToTab('security');

    await this.createApiKeyButton.click();

    const nameInput = this.page.locator('input[name="keyName"], [data-testid="key-name-input"]');
    await nameInput.fill(name);

    const createButton = this.page.locator('button:has-text("Create"), button:has-text("Generate")');
    await createButton.click();

    // Get the generated key
    const keyDisplay = this.page.locator('[data-testid="api-key-display"], .api-key-display');
    const key = await keyDisplay.textContent();

    return key || '';
  }

  /**
   * Revoke API key
   */
  async revokeAPIKey(keyName: string) {
    await this.navigateToTab('security');

    const keyRow = this.apiKeysList.locator('tr').filter({ hasText: keyName });
    const revokeButton = keyRow.locator('button:has-text("Revoke"), button:has-text("Delete")');
    await revokeButton.click();

    // Confirm
    const confirmButton = this.page.locator('button:has-text("Confirm")');
    if (await confirmButton.isVisible()) {
      await confirmButton.click();
    }
  }

  /**
   * Get active sessions count
   */
  async getSessionsCount(): Promise<number> {
    await this.navigateToTab('sessions');
    const sessions = this.sessionsList.locator('[data-testid="session-item"], .session-item');
    return await sessions.count();
  }

  /**
   * Revoke a specific session
   */
  async revokeSession(sessionId: string) {
    await this.navigateToTab('sessions');

    const sessionItem = this.sessionsList.locator(`[data-session-id="${sessionId}"], .session-item:has-text("${sessionId}")`);
    const revokeButton = sessionItem.locator('button:has-text("Revoke"), [data-testid="revoke-session"]');
    await revokeButton.click();
  }

  /**
   * Revoke all other sessions
   */
  async revokeAllOtherSessions() {
    await this.navigateToTab('sessions');
    await this.revokeAllSessionsButton.click();

    // Confirm
    const confirmButton = this.page.locator('button:has-text("Confirm")');
    if (await confirmButton.isVisible()) {
      await confirmButton.click();
    }

    await this.verifyToast('Sessions revoked', 'success');
  }

  /**
   * Update preferences
   */
  async updatePreferences(preferences: {
    theme?: string;
    language?: string;
    timezone?: string;
    dateFormat?: string;
  }) {
    await this.navigateToTab('preferences');

    if (preferences.theme) {
      await this.themeSelect.selectOption(preferences.theme);
    }

    if (preferences.language) {
      await this.languageSelect.selectOption(preferences.language);
    }

    if (preferences.timezone) {
      await this.timezoneSelect.selectOption(preferences.timezone);
    }

    if (preferences.dateFormat) {
      await this.dateFormatSelect.selectOption(preferences.dateFormat);
    }

    await this.savePreferencesButton.click();
    await this.verifyToast('Preferences saved', 'success');
  }

  /**
   * Toggle notification setting
   */
  async toggleNotification(setting: 'email' | 'push' | 'weeklyDigest' | 'jobCompleted' | 'jobFailed' | 'printerOffline' | 'securityAlert') {
    await this.navigateToTab('preferences');

    const toggle = this.page.locator(`input[name="${setting}"], [data-testid="toggle-${setting}"]`);
    await toggle.check();
  }

  /**
   * Verify theme is applied
   */
  async verifyTheme(theme: 'light' | 'dark') {
    const html = this.page.locator('html');
    const dataTheme = await html.getAttribute('data-theme');
    expect(dataTheme).toBe(theme);
  }

  /**
   * Verify profile form values
   */
  async verifyProfileValues(values: { name: string; email: string }) {
    await this.navigateToTab('profile');

    const nameValue = await this.nameInput.inputValue();
    const emailValue = await this.emailInput.inputValue();

    expect(nameValue).toBe(values.name);
    expect(emailValue).toBe(values.email);
  }

  /**
   * Verify current session is marked
   */
  async verifyCurrentSessionMarked() {
    await this.navigateToTab('sessions');

    const currentSession = this.sessionsList.locator('[data-current="true"], .session-item.current');
    await expect(currentSession).toBeVisible();
    await expect(currentSession).toContainText('Current');
  }

  /**
   * Get session info
   */
  async getSessionInfo(sessionId: string): Promise<{
    device: string;
    ip: string;
    lastActive: string;
  }> {
    const sessionItem = this.sessionsList.locator(`[data-session-id="${sessionId}"]`);

    const device = await sessionItem.locator('[data-testid="session-device"], .session-device').textContent() || '';
    const ip = await sessionItem.locator('[data-testid="session-ip"], .session-ip').textContent() || '';
    const lastActive = await sessionItem.locator('[data-testid="session-last-active"], .session-last-active').textContent() || '';

    return { device, ip, lastActive };
  }

  /**
   * Verify timezone display
   */
  async verifyTimezone(expectedTimezone: string) {
    const timezoneText = await this.timezoneDisplay.textContent();
    expect(timezoneText).toContain(expectedTimezone);
  }

  /**
   * Verify email validation on change
   */
  async verifyEmailValidation() {
    await this.navigateToTab('profile');

    await this.emailInput.fill('invalid-email');
    await this.emailInput.blur();

    const error = this.emailInput.locator('..').locator('.error, [data-testid="validation-error"]');
    await expect(error).toBeVisible();
  }

  /**
   * Verify password strength indicator
   */
  async verifyPasswordStrength(password: string) {
    await this.navigateToTab('profile');
    await this.changePasswordButton.click();

    await this.newPasswordInput.fill(password);

    const strengthIndicator = this.page.locator('[data-testid="password-strength"], .password-strength');
    await expect(strengthIndicator).toBeVisible();
  }

  /**
   * Verify settings tab navigation
   */
  async verifyTabNavigation() {
    const tabs = ['Profile', 'Security', 'Sessions', 'Preferences'];

    for (const tab of tabs) {
      await this.navigateToTab(tab.toLowerCase() as 'profile' | 'security' | 'sessions' | 'preferences');

      const activeTab = this.tabs.locator('button[aria-selected="true"], .active');
      await expect(activeTab).toContainText(tab);
    }
  }
}

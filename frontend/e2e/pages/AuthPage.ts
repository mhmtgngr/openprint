import { type Page, type Locator, expect } from '@playwright/test';
import { BasePage } from './BasePage';

/**
 * Authentication Page Object
 * Handles login, registration, OIDC, and SAML authentication flows
 */
export class AuthPage extends BasePage {
  // Login form elements
  readonly loginEmailInput: Locator;
  readonly loginPasswordInput: Locator;
  readonly loginSubmitButton: Locator;
  readonly rememberMeCheckbox: Locator;
  readonly forgotPasswordLink: Locator;

  // Registration form elements
  readonly registerLink: Locator;
  readonly registerNameInput: Locator;
  readonly registerEmailInput: Locator;
  readonly registerPasswordInput: Locator;
  readonly registerConfirmPasswordInput: Locator;
  readonly registerSubmitButton: Locator;

  // OIDC/SAML buttons
  readonly microsoftLoginButton: Locator;
  readonly googleLoginButton: Locator;
  readonly oidcLoginButton: Locator;
  readonly samlLoginButton: Locator;

  // Error and success messages
  readonly errorMessage: Locator;
  readonly successMessage: Locator;
  readonly validationErrors: Locator;

  // Password recovery
  readonly resetPasswordEmailInput: Locator;
  readonly resetPasswordSubmitButton: Locator;
  readonly resetPasswordBackLink: Locator;

  // Two-factor authentication
  readonly twoFactorCodeInput: Locator;
  readonly twoFactorSubmitButton: Locator;
  readonly twoFactorBackupCodeLink: Locator;
  readonly twoFactorRememberDeviceCheckbox: Locator;

  // Organization selection (for SSO users)
  readonly organizationSelect: Locator;
  readonly organizationContinueButton: Locator;

  constructor(page: Page) {
    super(page);

    // Login form
    this.loginEmailInput = page.getByLabel(/email|username|sign in/i);
    this.loginPasswordInput = page.getByLabel(/password/i);
    this.loginSubmitButton = page.getByRole('button', { name: /sign in|login|log in/i });
    this.rememberMeCheckbox = page.getByLabel(/remember me|keep me signed in/i);
    this.forgotPasswordLink = page.getByRole('link', { name: /forgot password/i });

    // Registration
    this.registerLink = page.getByRole('link', { name: /register|sign up|create account/i });
    this.registerNameInput = page.getByLabel(/full name|name/i);
    this.registerEmailInput = page.locator('input[type="email"]').nth(1);
    this.registerPasswordInput = page.locator('(//input[@type="password"])[1]');
    this.registerConfirmPasswordInput = page.locator('(//input[@type="password"])[2]');
    this.registerSubmitButton = page.getByRole('button', { name: /register|create account|sign up/i });

    // OIDC/SAML
    this.microsoftLoginButton = page.getByRole('button', { name: /microsoft|office 365|m365/i });
    this.googleLoginButton = page.getByRole('button', { name: /google/i });
    this.oidcLoginButton = page.getByRole('button', { name: /openid|oidc|single sign-on/i });
    this.samlLoginButton = page.getByRole('button', { name: /saml/i });

    // Messages
    this.errorMessage = page.locator('[data-testid="error"], .error, .alert-error');
    this.successMessage = page.locator('[data-testid="success"], .success, .alert-success');
    this.validationErrors = page.locator('.validation-error, .error-message');

    // Password recovery
    this.resetPasswordEmailInput = page.getByLabel(/email/i);
    this.resetPasswordSubmitButton = page.getByRole('button', { name: /send|reset|submit/i });
    this.resetPasswordBackLink = page.getByRole('link', { name: /back|sign in/i });

    // Two-factor
    this.twoFactorCodeInput = page.getByLabel(/code|verification code/i);
    this.twoFactorSubmitButton = page.getByRole('button', { name: /verify|confirm/i });
    this.twoFactorBackupCodeLink = page.getByRole('link', { name: /backup code|use backup/i });
    this.twoFactorRememberDeviceCheckbox = page.getByLabel(/remember|trust/i);

    // Organization selection
    this.organizationSelect = page.getByLabel(/organization|tenant/i);
    this.organizationContinueButton = page.getByRole('button', { name: /continue|select/i });
  }

  /**
   * Navigate to login page
   */
  async gotoLogin(): Promise<void> {
    await this.goto('/login');
    await this.isLoginVisible();
  }

  /**
   * Navigate to registration page
   */
  async gotoRegister(): Promise<void> {
    await this.goto('/register');
    await this.isRegisterVisible();
  }

  /**
   * Check if login form is visible
   */
  async isLoginVisible(): Promise<boolean> {
    return await this.loginEmailInput.isVisible();
  }

  /**
   * Check if registration form is visible
   */
  async isRegisterVisible(): Promise<boolean> {
    return await this.registerNameInput.isVisible();
  }

  /**
   * Login with email and password
   */
  async login(email: string, password: string, rememberMe = false): Promise<void> {
    await this.loginEmailInput.fill(email);
    await this.loginPasswordInput.fill(password);
    if (rememberMe) {
      await this.rememberMeCheckbox.check();
    }
    await this.loginSubmitButton.click();
    await this.page.waitForURL('/dashboard');
  }

  /**
   * Login with OIDC provider (mock for E2E)
   */
  async loginWithOIDC(provider: 'microsoft' | 'google'): Promise<void> {
    const button = provider === 'microsoft' ? this.microsoftLoginButton : this.googleLoginButton;
    await button.click();
    // In E2E tests, we would mock the OIDC flow
    // or handle the redirect to the provider
    await this.page.waitForURL('/dashboard');
  }

  /**
   * Login with SAML provider
   */
  async loginWithSAML(): Promise<void> {
    await this.samlLoginButton.click();
    // Handle SAML redirect
    await this.page.waitForURL('/dashboard');
  }

  /**
   * Register a new user
   */
  async register(userData: {
    name: string;
    email: string;
    password: string;
    confirmPassword?: string;
  }): Promise<void> {
    await this.registerNameInput.fill(userData.name);
    await this.registerEmailInput.fill(userData.email);
    await this.registerPasswordInput.fill(userData.password);
    await this.registerConfirmPasswordInput.fill(userData.confirmPassword || userData.password);
    await this.registerSubmitButton.click();
  }

  /**
   * Navigate to registration from login
   */
  async goToRegistration(): Promise<void> {
    await this.registerLink.click();
    await this.isRegisterVisible();
  }

  /**
   * Navigate to forgot password
   */
  async goToForgotPassword(): Promise<void> {
    await this.forgotPasswordLink.click();
    await expect(this.resetPasswordEmailInput).toBeVisible();
  }

  /**
   * Request password reset
   */
  async requestPasswordReset(email: string): Promise<void> {
    await this.resetPasswordEmailInput.fill(email);
    await this.resetPasswordSubmitButton.click();
  }

  /**
   * Submit two-factor authentication code
   */
  async submitTwoFactorCode(code: string, rememberDevice = false): Promise<void> {
    await this.twoFactorCodeInput.fill(code);
    if (rememberDevice) {
      await this.twoFactorRememberDeviceCheckbox.check();
    }
    await this.twoFactorSubmitButton.click();
    await this.page.waitForURL('/dashboard');
  }

  /**
   * Use backup code for two-factor
   */
  async useBackupCode(code: string): Promise<void> {
    await this.twoFactorBackupCodeLink.click();
    await this.twoFactorCodeInput.fill(code);
    await this.twoFactorSubmitButton.click();
  }

  /**
   * Select organization (for SSO users)
   */
  async selectOrganization(organization: string): Promise<void> {
    await this.organizationSelect.selectOption(organization);
    await this.organizationContinueButton.click();
  }

  /**
   * Get error message text
   */
  async getErrorMessage(): Promise<string> {
    await expect(this.errorMessage).toBeVisible({ timeout: 5000 });
    return await this.errorMessage.textContent() || '';
  }

  /**
   * Get success message text
   */
  async getSuccessMessage(): Promise<string> {
    await expect(this.successMessage).toBeVisible({ timeout: 5000 });
    return await this.successMessage.textContent() || '';
  }

  /**
   * Check if error is displayed
   */
  async hasError(): Promise<boolean> {
    return await this.errorMessage.isVisible();
  }

  /**
   * Verify user is logged in
   */
  async isLoggedIn(): Promise<boolean> {
    return this.page.url().includes('/dashboard') || !(await this.loginEmailInput.isVisible());
  }

  /**
   * Verify user is logged out
   */
  async isLoggedOut(): Promise<boolean> {
    return await this.loginEmailInput.isVisible();
  }

  /**
   * Logout from the application
   */
  async logout(): Promise<void> {
    await super.logout();
    await this.isLoginVisible();
  }

  /**
   * Mock OIDC authentication for E2E testing
   * This intercepts the OIDC flow and returns a mock response
   */
  async mockOIDCAuthentication(): Promise<void> {
    await this.page.route('**/auth/oidc/callback', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          access_token: 'mock-access-token',
          refresh_token: 'mock-refresh-token',
          user: {
            id: 'mock-user-id',
            email: 'mockuser@oidc.test',
            name: 'Mock OIDC User',
          },
        }),
      });
    });
  }

  /**
   * Mock SAML authentication for E2E testing
   */
  async mockSAMLAuthentication(): Promise<void> {
    await this.page.route('**/auth/saml/callback', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          access_token: 'mock-saml-token',
          refresh_token: 'mock-saml-refresh-token',
          user: {
            id: 'mock-saml-user-id',
            email: 'mockuser@saml.test',
            name: 'Mock SAML User',
          },
        }),
      });
    });
  }

  /**
   * Mock login API for E2E testing
   */
  async mockLoginAPI(response: {
    status: number;
    body?: any;
  }): Promise<void> {
    await this.page.route('**/auth/login', (route) => {
      route.fulfill({
        status: response.status,
        contentType: 'application/json',
        body: JSON.stringify(response.body || {}),
      });
    });
  }

  /**
   * Verify login form is displayed correctly
   */
  async verifyLoginForm(): Promise<void> {
    await expect(this.loginEmailInput).toBeVisible();
    await expect(this.loginPasswordInput).toBeVisible();
    await expect(this.loginSubmitButton).toBeVisible();
  }

  /**
   * Verify registration form is displayed correctly
   */
  async verifyRegistrationForm(): Promise<void> {
    await expect(this.registerNameInput).toBeVisible();
    await expect(this.registerEmailInput).toBeVisible();
    await expect(this.registerPasswordInput).toBeVisible();
    await expect(this.registerConfirmPasswordInput).toBeVisible();
    await expect(this.registerSubmitButton).toBeVisible();
  }

  /**
   * Verify social login buttons are visible
   */
  async verifySocialLoginButtons(): Promise<void> {
    await expect(this.microsoftLoginButton).toBeVisible();
    await expect(this.googleLoginButton).toBeVisible();
  }

  /**
   * Check if OIDC button is visible
   */
  async hasOIDCButton(): Promise<boolean> {
    return await this.oidcLoginButton.isVisible();
  }

  /**
   * Check if SAML button is visible
   */
  async hasSAMLButton(): Promise<boolean> {
    return await this.samlLoginButton.isVisible();
  }

  /**
   * Get validation errors
   */
  async getValidationErrors(): Promise<string[]> {
    const errors: string[] = [];
    const errorElements = await this.validationErrors.all();
    for (const error of errorElements) {
      errors.push(await error.textContent() || '');
    }
    return errors;
  }

  /**
   * Clear login form
   */
  async clearLoginForm(): Promise<void> {
    await this.loginEmailInput.clear();
    await this.loginPasswordInput.clear();
  }

  /**
   * Check if password field is masked
   */
  async isPasswordMasked(): Promise<boolean> {
    const inputType = await this.loginPasswordInput.getAttribute('type');
    return inputType === 'password';
  }

  /**
   * Toggle password visibility
   */
  async togglePasswordVisibility(): Promise<void> {
    const toggleButton = this.loginPasswordInput.locator('..').getByRole('button', { name: /show|hide/i });
    if (await toggleButton.isVisible()) {
      await toggleButton.click();
    }
  }
}

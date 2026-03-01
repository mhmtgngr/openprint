/**
 * Authentication Page Object
 * Handles login, registration, OIDC, and SAML authentication flows
 */
import { Page, Locator, expect } from '@playwright/test';
import { BasePage } from './BasePage';
import { mockApiResponse } from '../helpers';

export class AuthPage extends BasePage {
  // Form elements
  readonly loginForm: Locator;
  readonly registerForm: Locator;
  readonly emailInput: Locator;
  readonly passwordInput: Locator;
  readonly nameInput: Locator;
  readonly confirmPasswordInput: Locator;
  readonly submitButton: Locator;
  readonly toggleFormButton: Locator;
  readonly errorMessage: Locator;
  readonly successMessage: Locator;

  // OIDC/SAML buttons
  readonly microsoftButton: Locator;
  readonly googleButton: Locator;
  readonly samlButton: Locator;
  readonly oidcButton: Locator;

  // Password recovery
  readonly forgotPasswordLink: Locator;
  readonly resetPasswordForm: Locator;
  readonly resetEmailInput: Locator;
  readonly resetSubmitButton: Locator;
  readonly backToLoginLink: Locator;

  // Email verification
  readonly verificationForm: Locator;
  readonly verificationCodeInput: Locator;
  readonly resendCodeButton: Locator;
  readonly verifiedMessage: Locator;

  // MFA
  readonly mfaCodeInput: Locator;
  readonly mfaVerifyButton: Locator;
  readonly rememberDeviceCheckbox: Locator;

  constructor(page: Page) {
    super(page);

    // Initialize form locators
    this.loginForm = page.locator('[data-testid="login-form"], form[data-type="login"]');
    this.registerForm = page.locator('[data-testid="register-form"], form[data-type="register"]');
    this.emailInput = page.locator('input[type="email"], input[name="email"]');
    this.passwordInput = page.locator('input[type="password"], input[name="password"]');
    this.nameInput = page.locator('input[name="name"], input[type="text"].first');
    this.confirmPasswordInput = page.locator('input[name="confirmPassword"], input[data-type="confirm-password"]');
    this.submitButton = page.locator('button[type="submit"], [data-testid="submit-button"]');
    this.toggleFormButton = page.locator('button:has-text("Sign up"), button:has-text("Log in"), a:has-text("Register")');
    this.errorMessage = page.locator('[data-testid="error-message"], .error-message, [role="alert"].error');
    this.successMessage = page.locator('[data-testid="success-message"], .success-message, [role="alert"].success');

    // OIDC/SAML buttons
    this.microsoftButton = page.locator('button:has-text("Microsoft"), [data-provider="microsoft"]');
    this.googleButton = page.locator('button:has-text("Google"), [data-provider="google"]');
    this.samlButton = page.locator('button:has-text("SAML"), [data-provider="saml"]');
    this.oidcButton = page.locator('button:has-text("OIDC"), [data-provider="oidc"]');

    // Password recovery
    this.forgotPasswordLink = page.locator('a:has-text("Forgot password")');
    this.resetPasswordForm = page.locator('[data-testid="reset-password-form"]');
    this.resetEmailInput = page.locator('input[name="resetEmail"], [data-testid="reset-email"]');
    this.resetSubmitButton = page.locator('button:has-text("Send reset link"), [data-testid="reset-submit"]');
    this.backToLoginLink = page.locator('a:has-text("Back to login"), [data-testid="back-to-login"]');

    // Email verification
    this.verificationForm = page.locator('[data-testid="verification-form"]');
    this.verificationCodeInput = page.locator('input[name="verificationCode"], [data-testid="verification-code"]');
    this.resendCodeButton = page.locator('button:has-text("Resend"), [data-testid="resend-code"]');
    this.verifiedMessage = page.locator('[data-testid="verified-message"]');

    // MFA
    this.mfaCodeInput = page.locator('input[name="mfaCode"], [data-testid="mfa-code"]');
    this.mfaVerifyButton = page.locator('button:has-text("Verify"), [data-testid="mfa-verify"]');
    this.rememberDeviceCheckbox = page.locator('input[name="rememberDevice"], [data-testid="remember-device"]');
  }

  /**
   * Navigate to login page
   */
  async goto() {
    await this.goto('/login');
    await this.waitForPageLoad();
  }

  /**
   * Setup auth API mocks
   */
  async setupMocks() {
    // Mock login endpoint
    await this.page.route('**/api/v1/auth/login', async (route) => {
      const request = route.request();
      const body = request.postDataJSON();

      if (body.email === 'test@example.com' && body.password === 'TestPassword123!') {
        await mockApiResponse(route, {
          userId: '1',
          access_token: 'mock-access-token',
          refresh_token: 'mock-refresh-token',
          org: { id: 'org-1', name: 'Test Org' },
        });
      } else if (body.email === 'admin@example.com' && body.password === 'AdminPassword123!') {
        await mockApiResponse(route, {
          userId: '2',
          access_token: 'admin-access-token',
          refresh_token: 'admin-refresh-token',
          org: { id: 'org-1', name: 'Test Org' },
        });
      } else {
        await mockApiResponse(route, { error: 'Invalid credentials' }, 401);
      }
    });

    // Mock register endpoint
    await this.page.route('**/api/v1/auth/register', async (route) => {
      const request = route.request();
      const body = request.postDataJSON();

      if (body.email && body.password && body.name) {
        await mockApiResponse(route, {
          userId: 'new-user-id',
          message: 'Registration successful. Please check your email to verify your account.',
        });
      } else {
        await mockApiResponse(route, { error: 'Invalid registration data' }, 400);
      }
    });

    // Mock me endpoint
    await this.page.route('**/api/v1/auth/me', async (route) => {
      await mockApiResponse(route, {
        id: '1',
        email: 'test@example.com',
        name: 'Test User',
        role: 'user',
        orgId: 'org-1',
        isActive: true,
        emailVerified: true,
      });
    });

    // Mock OIDC endpoints
    await this.page.route('**/api/v1/auth/oidc/microsoft/url', async (route) => {
      await mockApiResponse(route, {
        url: 'https://login.microsoftonline.com/oidc-mock',
      });
    });

    await this.page.route('**/api/v1/auth/oidc/microsoft/callback', async (route) => {
      await mockApiResponse(route, {
        userId: '1',
        access_token: 'oidc-access-token',
        refresh_token: 'oidc-refresh-token',
        org: { id: 'org-1', name: 'Test Org' },
      });
    });

    // Mock SAML endpoints
    await this.page.route('**/api/v1/auth/saml/url', async (route) => {
      await mockApiResponse(route, {
        url: 'https://saml-idp-mock.com/sso',
      });
    });

    // Mock password reset
    await this.page.route('**/api/v1/auth/forgot-password', async (route) => {
      await mockApiResponse(route, {
        message: 'Password reset email sent',
      });
    });

    await this.page.route('**/api/v1/auth/reset-password', async (route) => {
      await mockApiResponse(route, {
        message: 'Password reset successful',
      });
    });

    // Mock email verification
    await this.page.route('**/api/v1/auth/verify-email', async (route) => {
      await mockApiResponse(route, {
        message: 'Email verified successfully',
      });
    });

    // Mock MFA verification
    await this.page.route('**/api/v1/auth/mfa/verify', async (route) => {
      await mockApiResponse(route, {
        access_token: 'mfa-verified-token',
        refresh_token: 'mfa-refresh-token',
      });
    });
  }

  /**
   * Fill and submit login form
   */
  async login(email: string, password: string) {
    await this.emailInput.fill(email);
    await this.passwordInput.fill(password);
    await this.submitButton.click();
  }

  /**
   * Login as regular user
   */
  async loginAsUser() {
    await this.login('test@example.com', 'TestPassword123!');
    await this.page.waitForURL('**/dashboard', { timeout: 10000 });
  }

  /**
   * Login as admin user
   */
  async loginAsAdmin() {
    await this.login('admin@example.com', 'AdminPassword123!');
    await this.page.waitForURL('**/dashboard', { timeout: 10000 });
  }

  /**
   * Switch to registration form
   */
  async switchToRegister() {
    await this.toggleFormButton.filter({ hasText: /sign up|register/i }).click();
    await expect(this.registerForm).toBeVisible();
  }

  /**
   * Switch to login form
   */
  async switchToLogin() {
    await this.toggleFormButton.filter({ hasText: /log in|sign in/i }).click();
    await expect(this.loginForm).toBeVisible();
  }

  /**
   * Fill and submit registration form
   */
  async register(name: string, email: string, password: string, confirmPassword?: string) {
    await this.switchToRegister();
    await this.nameInput.fill(name);
    await this.emailInput.fill(email);
    await this.passwordInput.fill(password);

    if (confirmPassword) {
      await this.confirmPasswordInput.fill(confirmPassword);
    }

    await this.submitButton.click();
  }

  /**
   * Verify error message is displayed
   */
  async verifyErrorMessage(expectedMessage: string) {
    await expect(this.errorMessage).toBeVisible();
    await expect(this.errorMessage).toContainText(expectedMessage);
  }

  /**
   * Verify success message is displayed
   */
  async verifySuccessMessage(expectedMessage: string) {
    await expect(this.successMessage).toBeVisible();
    await expect(this.successMessage).toContainText(expectedMessage);
  }

  /**
   * Click Microsoft OAuth button
   */
  async clickMicrosoftLogin() {
    await this.microsoftButton.click();
  }

  /**
   * Click Google OAuth button
   */
  async clickGoogleLogin() {
    await this.googleButton.click();
  }

  /**
   * Click SAML login button
   */
  async clickSamlLogin() {
    await this.samlButton.click();
  }

  /**
   * Initiate password reset flow
   */
  async initiatePasswordReset(email: string) {
    await this.forgotPasswordLink.click();
    await expect(this.resetPasswordForm).toBeVisible();
    await this.resetEmailInput.fill(email);
    await this.resetSubmitButton.click();
  }

  /**
   * Go back to login from reset password
   */
  async backToLogin() {
    await this.backToLoginLink.click();
    await expect(this.loginForm).toBeVisible();
  }

  /**
   * Verify email input validation
   */
  async verifyEmailValidation() {
    await this.emailInput.fill('invalid-email');
    await this.emailInput.blur();

    // Check for validation error
    const validationError = this.emailInput.locator('..').locator('.error, [data-testid="validation-error"]');
    await expect(validationError).toBeVisible();
  }

  /**
   * Verify password validation
   */
  async verifyPasswordValidation() {
    await this.passwordInput.fill('123');
    await this.passwordInput.blur();

    // Check for validation error
    const validationError = this.passwordInput.locator('..').locator('.error, [data-testid="validation-error"]');
    await expect(validationError).toBeVisible();
  }

  /**
   * Verify password match validation
   */
  async verifyPasswordMatchValidation() {
    await this.switchToRegister();
    await this.passwordInput.fill('Password123!');
    await this.confirmPasswordInput.fill('Different123!');
    await this.confirmPasswordInput.blur();

    // Check for validation error
    const validationError = this.confirmPasswordInput.locator('..').locator('.error, [data-testid="validation-error"]');
    await expect(validationError).toBeVisible();
  }

  /**
   * Submit verification code
   */
  async submitVerificationCode(code: string) {
    await this.verificationCodeInput.fill(code);
    await this.submitButton.click();
  }

  /**
   * Resend verification code
   */
  async resendVerificationCode() {
    await this.resendCodeButton.click();
  }

  /**
   * Submit MFA code
   */
  async submitMfaCode(code: string, rememberDevice: boolean = false) {
    await this.mfaCodeInput.fill(code);
    if (rememberDevice) {
      await this.rememberDeviceCheckbox.check();
    }
    await this.mfaVerifyButton.click();
  }

  /**
   * Verify login form is visible
   */
  async isLoginFormVisible(): Promise<boolean> {
    return await this.loginForm.isVisible();
  }

  /**
   * Verify register form is visible
   */
  async isRegisterFormVisible(): Promise<boolean> {
    return await this.registerForm.isVisible();
  }

  /**
   * Get current URL path
   */
  async getCurrentPath(): Promise<string> {
    return new URL(this.page.url()).pathname;
  }

  /**
   * Wait for redirect to dashboard
   */
  async waitForDashboardRedirect() {
    await this.page.waitForURL('**/dashboard', { timeout: 10000 });
  }

  /**
   * Verify all form fields are present
   */
  async verifyFormFieldsPresent() {
    await expect(this.emailInput).toBeVisible();
    await expect(this.passwordInput).toBeVisible();
    await expect(this.submitButton).toBeVisible();
  }

  /**
   * Verify OIDC buttons are available
   */
  async verifyOidcButtonsAvailable() {
    await expect(this.microsoftButton).toBeVisible();
    await expect(this.googleButton).isVisible().catch(() => {});
    await expect(this.samlButton).isVisible().catch(() => {});
  }

  /**
   * Clear form inputs
   */
  async clearForm() {
    await this.emailInput.clear();
    await this.passwordInput.clear();
  }

  /**
   * Verify form is in loading state
   */
  async verifyFormLoading() {
    await expect(this.submitButton).toBeDisabled();
    await expect(this.spinner).toBeVisible();
  }

  /**
   * Verify password is masked
   */
  async verifyPasswordMasked() {
    await expect(this.passwordInput).toHaveAttribute('type', 'password');
  }

  /**
   * Toggle password visibility
   */
  async togglePasswordVisibility() {
    const toggleButton = this.passwordInput.locator('..').locator('button[type="button"], .toggle-password');
    await toggleButton.click();
  }

  /**
   * Verify page title
   */
  async verifyPageTitle() {
    await expect(this.page).toHaveTitle(/login|sign in|authentication/i);
  }
}

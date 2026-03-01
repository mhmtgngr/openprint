import { test, expect } from '@playwright/test';
import { LoginPage } from '../helpers/page-objects';
import { testUsers } from '../helpers/test-data';

test.describe('Authentication', () => {
  let loginPage: LoginPage;

  test.beforeEach(async ({ page }) => {
    loginPage = new LoginPage(page);
    await loginPage.goto();
  });

  test('should display login form', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /sign in/i })).toBeVisible();
    await expect(loginPage.emailInput).toBeVisible();
    await expect(loginPage.passwordInput).toBeVisible();
    await expect(loginPage.submitButton).toBeVisible();
  });

  test('should show validation errors for empty fields', async ({ page }) => {
    await loginPage.submitButton.click();

    // Check for validation messages
    await expect(page.getByText(/email is required/i)).toBeVisible();
  });

  test('should show error for invalid credentials', async ({ page }) => {
    await loginPage.emailInput.fill('invalid@test.com');
    await loginPage.passwordInput.fill('wrongpassword');
    await loginPage.submitButton.click();

    // Should show error message
    await expect(loginPage.errorMessage).toBeVisible();
    await expect(page.getByText(/invalid credentials/i)).toBeVisible();
  });

  test('should redirect to dashboard after successful login', async ({ page }) => {
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    // Should redirect to dashboard
    await expect(page).toHaveURL('/dashboard');
    await expect(page.getByRole('heading', { name: /welcome/i })).toBeVisible();
  });

  test('should persist authentication across page reloads', async ({ page, context }) => {
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    // Reload page
    await page.reload();

    // Should still be authenticated
    await expect(page).toHaveURL('/dashboard');
    await expect(page.getByRole('heading', { name: /welcome/i })).toBeVisible();
  });

  test('should logout and redirect to login', async ({ page }) => {
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    // Click logout
    await page.getByRole('button', { name: /logout/i }).click();

    // Should redirect to login
    await expect(page).toHaveURL('/login');
  });

  test('should protect routes and redirect to login', async ({ page }) => {
    // Try to access protected route without authentication
    await page.goto('/dashboard');

    // Should redirect to login
    await expect(page).toHaveURL(/\/login/);
  });

  test('should remember email for convenience', async ({ page }) => {
    await loginPage.emailInput.fill(testUsers.admin.email);
    await loginPage.passwordInput.fill(testUsers.admin.password);
    await loginPage.submitButton.click();

    // Logout
    await page.getByRole('button', { name: /logout/i }).click();

    // Go back to login
    await loginPage.goto();

    // Email should be remembered (check localStorage or form value)
    const emailValue = await loginPage.emailInput.inputValue();
    expect(emailValue).toBe(testUsers.admin.email);
  });
});

test.describe('Authentication - Password Reset', () => {
  test('should show forgot password link', async ({ page }) => {
    await page.goto('/login');
    const forgotLink = page.getByRole('link', { name: /forgot password/i });

    await expect(forgotLink).toBeVisible();
  });

  test('should navigate to password reset page', async ({ page }) => {
    await page.goto('/login');
    await page.getByRole('link', { name: /forgot password/i }).click();

    await expect(page).toHaveURL(/\/forgot-password/);
    await expect(page.getByRole('heading', { name: /reset password/i })).toBeVisible();
  });
});

test.describe('Authentication - Registration', () => {
  test('should show registration link', async ({ page }) => {
    await page.goto('/login');
    const registerLink = page.getByRole('link', { name: /register|sign up/i });

    await expect(registerLink).toBeVisible();
  });

  test('should navigate to registration page', async ({ page }) => {
    await page.goto('/login');
    await page.getByRole('link', { name: /register|sign up/i }).click();

    await expect(page).toHaveURL(/\/register/);
    await expect(page.getByRole('heading', { name: /create account/i })).toBeVisible();
  });

  test('should validate registration form', async ({ page }) => {
    await page.goto('/register');

    // Submit empty form
    await page.getByRole('button', { name: /create account|register/i }).click();

    // Should show validation errors
    await expect(page.getByText(/name is required/i)).toBeVisible();
    await expect(page.getByText(/email is required/i)).toBeVisible();
    await expect(page.getByText(/password is required/i)).toBeVisible();
  });

  test('should validate password strength', async ({ page }) => {
    await page.goto('/register');

    await page.getByLabel(/name/i).fill('Test User');
    await page.getByLabel(/email/i).fill('test@example.com');
    await page.getByLabel(/password/i).fill('weak');

    await page.getByRole('button', { name: /create account|register/i }).click();

    // Should show password strength error
    await expect(page.getByText(/password is too weak/i)).toBeVisible();
  });
});

test.describe('Authentication - Session Management', () => {
  test('should handle token refresh automatically', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    // Wait a bit and make an authenticated request
    await page.waitForTimeout(2000);

    // Navigate to a protected route
    await page.goto('/settings');

    // Should still be authenticated
    await expect(page.getByRole('heading', { name: /settings/i })).toBeVisible();
  });

  test('should clear session on logout', async ({ page, context }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    // Logout
    await page.getByRole('button', { name: /logout/i }).click();

    // Check localStorage is cleared
    const tokens = await page.evaluate(() =>
      localStorage.getItem('auth_tokens')
    );
    expect(tokens).toBeNull();
  });

  test('should handle multiple tabs', async ({ browser }) => {
    const context = await browser.newContext();
    const page1 = await context.newPage();
    const page2 = await context.newPage();

    // Login in first tab
    const loginPage1 = new LoginPage(page1);
    await loginPage1.goto();
    await loginPage1.login(testUsers.admin.email, testUsers.admin.password);

    // Navigate in second tab
    await page2.goto('/dashboard');

    // Should be authenticated in second tab too
    await expect(page2.getByRole('heading', { name: /welcome/i })).toBeVisible();

    // Logout in first tab
    await page1.getByRole('button', { name: /logout/i }).click();

    // Reload second tab
    await page2.reload();

    // Second tab should also be logged out
    await expect(page2).toHaveURL(/\/login/);

    await context.close();
  });
});

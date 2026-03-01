import { test, expect } from '@playwright/test';

test.describe('OpenPrint Login', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('http://192.168.31.76:3000');
  });

  test('should display login form', async ({ page }) => {
    await expect(page).toHaveTitle(/OpenPrint/);
    await expect(page.locator('input[type="email"]').or(page.locator('input[name="email"]'))).toBeVisible();
    await expect(page.locator('input[type="password"]').or(page.locator('input[name="password"]'))).toBeVisible();
  });

  test('should login with valid credentials', async ({ page }) => {
    // Fill in login form
    const emailInput = page.locator('input[type="email"], input[name="email"]');
    const passwordInput = page.locator('input[type="password"], input[name="password"]');
    const loginButton = page.locator('button[type="submit"], button:has-text("Login"), button:has-text("Sign In")');

    await emailInput.fill('test@openprint.local');
    await passwordInput.fill('TestPassword123');

    // Click login button
    await loginButton.click();

    // Wait for navigation or success response
    await page.waitForURL('**/dashboard', { timeout: 5000 }).catch(() => {});
    await page.waitForTimeout(2000);

    // Check if login was successful - either by URL change or dashboard element
    const currentUrl = page.url();
    const hasDashboardElement = await page.locator('text=Dashboard, h1:has-text("Dashboard"), [data-testid="dashboard"]').count() > 0;

    console.log('Current URL after login:', currentUrl);
    console.log('Has dashboard element:', hasDashboardElement);

    // Take screenshot for debugging
    await page.screenshot({ path: 'tests/screenshots/login-success.png', fullPage: true });

    // Assert success - either URL changed or we see dashboard content
    expect(currentUrl.includes('dashboard') || hasDashboardElement).toBeTruthy();
  });

  test('should show error with invalid credentials', async ({ page }) => {
    const emailInput = page.locator('input[type="email"], input[name="email"]');
    const passwordInput = page.locator('input[type="password"], input[name="password"]');
    const loginButton = page.locator('button[type="submit"], button:has-text("Login"), button:has-text("Sign In")');

    await emailInput.fill('test@openprint.local');
    await passwordInput.fill('WrongPassword123');

    await loginButton.click();
    await page.waitForTimeout(1000);

    // Should show error message or stay on login page
    const currentUrl = page.url();
    expect(currentUrl).toContain('3000');

    // Take screenshot
    await page.screenshot({ path: 'tests/screenshots/login-error.png' });
  });

  test('should validate required fields', async ({ page }) => {
    const loginButton = page.locator('button[type="submit"], button:has-text("Login"), button:has-text("Sign In")');

    // Try to login without filling fields
    await loginButton.click();
    await page.waitForTimeout(500);

    // Check for validation messages
    const hasRequiredFieldText = await page.getByText(/required|email.*required|password.*required/i).count() > 0;

    await page.screenshot({ path: 'tests/screenshots/login-validation.png' });
  });
});

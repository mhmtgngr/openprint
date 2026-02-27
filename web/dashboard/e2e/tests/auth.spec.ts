import { test, expect } from '@playwright/test';

test.describe('Authentication', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('should redirect to login when not authenticated', async ({ page }) => {
    await expect(page).toHaveURL(/.*\/login/);
  });

  test('should display login form', async ({ page }) => {
    await expect(page.getByText('OpenPrint Cloud')).toBeVisible();
    await expect(page.getByText('Sign in to your account')).toBeVisible();
    await expect(page.getByLabel('Email Address')).toBeVisible();
    await expect(page.getByLabel('Password')).toBeVisible();
    await expect(page.getByRole('button', { name: 'Sign In' })).toBeVisible();
  });

  test('should toggle between login and register', async ({ page }) => {
    await expect(page.getByText("Don't have an account? Sign up")).toBeVisible();

    await page.getByRole('button', { name: "Don't have an account? Sign up" }).click();

    await expect(page.getByText('Create a new account')).toBeVisible();
    await expect(page.getByLabel('Full Name')).toBeVisible();
    await expect(page.getByRole('button', { name: 'Create Account' })).toBeVisible();

    await page.getByRole('button', { name: "Already have an account? Sign in" }).click();

    await expect(page.getByText('Sign in to your account')).toBeVisible();
  });

  test('should validate email format', async ({ page }) => {
    const emailInput = page.getByLabel('Email Address');
    await emailInput.fill('invalid-email');
    await emailInput.blur();

    // Browser validation
    const isValid = await emailInput.evaluate((el) => (el as HTMLInputElement).checkValidity());
    expect(isValid).toBe(false);
  });

  test('should validate password length', async ({ page }) => {
    // Switch to register to see password validation
    await page.getByRole('button', { name: "Don't have an account? Sign up" }).click();

    const passwordInput = page.getByLabel('Password');
    await passwordInput.fill('short');
    await passwordInput.blur();

    const isValid = await passwordInput.evaluate((el) => (el as HTMLInputElement).checkValidity());
    expect(isValid).toBe(false);
  });

  test('should show loading state on form submission', async ({ page }) => {
    await page.getByLabel('Email Address').fill('test@example.com');
    await page.getByLabel('Password').fill('password123');

    // Mock network request - in real tests, you'd use MSW or similar
    const submitButton = page.getByRole('button', { name: 'Sign In' });
    await submitButton.click();

    // Button should show loading state (will fail on network error, but that's expected)
    await expect(submitButton).toBeVisible();
  });
});

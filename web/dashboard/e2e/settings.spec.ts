import { test, expect } from '@playwright/test';
import { login, mockApiResponse, mockUsers, mockOrganization } from './helpers';

test.describe('Settings Page', () => {
  test.beforeEach(async ({ page }) => {
    // Setup auth mock
    await page.route('**/api/v1/auth/me', async (route) => {
      await mockApiResponse(route, mockUsers[0]);
    });

    // Setup organization mock
    await page.route('**/api/v1/organizations', async (route) => {
      await mockApiResponse(route, mockOrganization);
    });

    // Setup user update mock
    await page.route('**/api/v1/users/me', async (route) => {
      if (route.request().method() === 'PATCH') {
        await mockApiResponse(route, {
          ...mockUsers[0],
          name: 'Updated Name',
        });
      } else {
        await mockApiResponse(route, mockUsers[0]);
      }
    });

    // Setup password change mock
    await page.route('**/api/v1/users/me/password', async (route) => {
      await mockApiResponse(route, {}, 204);
    });

    await login(page);
    await page.goto('/settings');
    await page.waitForURL('**/settings');
  });

  test('should display settings page header', async ({ page }) => {
    await expect(page.locator('h1')).toContainText('Settings');
    await expect(page.locator('text=Manage your account and preferences')).toBeVisible();
  });

  test('should display tabs', async ({ page }) => {
    await expect(page.locator('text=Profile')).toBeVisible();
    await expect(page.locator('text=Security')).toBeVisible();
    await expect(page.locator('text=Preferences')).toBeVisible();
  });

  test('should switch between tabs', async ({ page }) => {
    // Click Security tab
    await page.click('button:has-text("Security")');
    await expect(page.locator('text=Change Password')).toBeVisible();

    // Click Preferences tab
    await page.click('button:has-text("Preferences")');
    await expect(page.locator('text=App Preferences')).toBeVisible();

    // Click Profile tab
    await page.click('button:has-text("Profile")');
    await expect(page.locator('text=Profile Information')).toBeVisible();
  });

  test('should display profile form', async ({ page }) => {
    await expect(page.locator('text=Profile Information')).toBeVisible();
    await expect(page.locator('label[for="name"]')).toBeVisible();
    await expect(page.locator('label[for="email"]')).toBeVisible();
    await expect(page.locator('button:has-text("Save Changes")')).toBeVisible();
  });

  test('should show user avatar with initial', async ({ page }) => {
    const avatar = page.locator('.w-20.h-20.bg-gradient-to-br');
    await expect(avatar).toBeVisible();
    await expect(avatar).toContainText(mockUsers[0].name.charAt(0).toUpperCase());
  });

  test('should pre-fill user data in form', async ({ page }) => {
    const nameInput = page.locator('input#name');
    const emailInput = page.locator('input#email');

    await expect(nameInput).toHaveValue(mockUsers[0].name);
    await expect(emailInput).toHaveValue(mockUsers[0].email);
  });

  test('should display organization info', async ({ page }) => {
    await expect(page.locator('text=Organization:')).toBeVisible();
    await expect(page.locator(`text=${mockOrganization.name}`)).toBeVisible();
    await expect(page.locator('text=Role:')).toBeVisible();
    await expect(page.locator(`text=${mockUsers[0].role}`)).toBeVisible();
  });

  test('should update user profile', async ({ page }) => {
    const nameInput = page.locator('input#name');
    await nameInput.clear();
    await nameInput.fill('Updated Name');

    const saveButton = page.locator('button:has-text("Save Changes")');
    await saveButton.click();

    // Should show success message
    await expect(page.locator('text=Profile updated successfully')).toBeVisible();
  });

  test('should have change avatar button', async ({ page }) => {
    const changeAvatarButton = page.locator('button:has-text("Change Avatar")');
    await expect(changeAvatarButton).toBeVisible();
  });

  test('should display security tab with password form', async ({ page }) => {
    await page.click('button:has-text("Security")');

    await expect(page.locator('text=Change Password')).toBeVisible();
    await expect(page.locator('label[for="current-password"]')).toBeVisible();
    await expect(page.locator('label[for="new-password"]')).toBeVisible();
    await expect(page.locator('label[for="confirm-password"]')).toBeVisible();
    await expect(page.locator('button:has-text("Change Password")')).toBeVisible();
  });

  test('should validate password confirmation', async ({ page }) => {
    await page.click('button:has-text("Security")');

    await page.fill('input#current-password', 'CurrentPass123!');
    await page.fill('input#new-password', 'NewPass123!');
    await page.fill('input#confirm-password', 'DifferentPass123!');

    const changeButton = page.locator('button:has-text("Change Password")');
    await changeButton.click();

    // Should show error message
    await expect(page.locator('text=Passwords do not match')).toBeVisible();
  });

  test('should validate minimum password length', async ({ page }) => {
    await page.click('button:has-text("Security")');

    await page.fill('input#current-password', 'CurrentPass123!');
    await page.fill('input#new-password', 'short');
    await page.fill('input#confirm-password', 'short');

    const changeButton = page.locator('button:has-text("Change Password")');
    await changeButton.click();

    // Should show error message
    await expect(page.locator('text=Password must be at least 8 characters')).toBeVisible();
  });

  test('should change password successfully', async ({ page }) => {
    await page.click('button:has-text("Security")');

    await page.fill('input#current-password', 'CurrentPass123!');
    await page.fill('input#new-password', 'NewPass123!');
    await page.fill('input#confirm-password', 'NewPass123!');

    const changeButton = page.locator('button:has-text("Change Password")');
    await changeButton.click();

    // Should show success message
    await expect(page.locator('text=Password changed successfully')).toBeVisible();
  });

  test('should display active sessions section', async ({ page }) => {
    await page.click('button:has-text("Security")');

    await expect(page.locator('text=Active Sessions')).toBeVisible();
    await expect(page.locator('text=Current Session')).toBeVisible();
    await expect(page.locator('text=Current')).toBeVisible();
  });

  test('should display preferences tab', async ({ page }) => {
    await page.click('button:has-text("Preferences")');

    await expect(page.locator('text=App Preferences')).toBeVisible();
    await expect(page.locator('text=Dark Mode')).toBeVisible();
    await expect(page.locator('text=Email Notifications')).toBeVisible();
    await expect(page.locator('text=Language')).toBeVisible();
    await expect(page.locator('text=Timezone')).toBeVisible();
  });

  test('should have language selector', async ({ page }) => {
    await page.click('button:has-text("Preferences")');

    const languageSelect = page.locator('select').filter({ hasText: 'English' });
    await expect(languageSelect).toBeVisible();

    const options = await languageSelect.locator('option').allTextContents();
    expect(options).toContain('English (US)');
  });

  test('should have timezone selector', async ({ page }) => {
    await page.click('button:has-text("Preferences")');

    await expect(page.locator('select')).toBeVisible();
    await expect(page.locator('text=Pacific Time')).toBeVisible();
  });

  test('should show success message then dismiss it', async ({ page }) => {
    const nameInput = page.locator('input#name');
    await nameInput.clear();
    await nameInput.fill('Updated Name');

    const saveButton = page.locator('button:has-text("Save Changes")');
    await saveButton.click();

    // Should show success message
    await expect(page.locator('text=Profile updated successfully')).toBeVisible();

    // Message should disappear after timeout
    await page.waitForTimeout(3500);
    await expect(page.locator('text=Profile updated successfully')).not.toBeVisible();
  });

  test('should highlight active tab', async ({ page }) => {
    // Profile tab should be active initially
    const profileTab = page.locator('button').filter({ hasText: 'Profile' });
    await expect(profileTab).toHaveClass(/border-blue-500/);
  });

  test('should navigate to dashboard via sidebar', async ({ page }) => {
    await page.click('a[href="/dashboard"]');
    await page.waitForURL('**/dashboard');

    await expect(page.locator('h1')).toContainText('Welcome back');
  });

  test('should highlight Settings in navigation', async ({ page }) => {
    const settingsLink = page.locator('a[href="/settings"]');
    await expect(settingsLink).toHaveClass(/bg-blue-100/);
  });

  test('should handle API errors gracefully', async ({ page }) => {
    // Mock error response
    await page.route('**/api/v1/users/me', async (route) => {
      if (route.request().method() === 'PATCH') {
        await route.abort('failed');
      }
    });

    const nameInput = page.locator('input#name');
    await nameInput.clear();
    await nameInput.fill('Updated Name');

    const saveButton = page.locator('button:has-text("Save Changes")');
    await saveButton.click();

    // Should show error message
    await expect(page.locator('text=Failed to update profile')).toBeVisible();
  });

  test('should be responsive on mobile', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });

    // Check that main content is still visible
    await expect(page.locator('h1')).toBeVisible();
    await expect(page.locator('text=Profile Information')).toBeVisible();
  });
});

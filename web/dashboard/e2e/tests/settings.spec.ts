import { test, expect } from '@playwright/test';
import { login, mockApiResponse, mockUsers } from '../helpers';

test.describe('Settings Page', () => {
  test.beforeEach(async ({ page }) => {
    // Setup auth mock
    await page.route('**/api/v1/auth/me', async (route) => {
      await mockApiResponse(route, mockUsers[0]);
    });

    await login(page);
    await page.goto('/settings');
  });

  test('should display settings page header', async ({ page }) => {
    await expect(page.locator('h1')).toContainText('Settings');
  });

  test('should display settings tabs', async ({ page }) => {
    await expect(page.locator('button:has-text("Profile")')).toBeVisible();
    await expect(page.locator('button:has-text("Security")')).toBeVisible();
    await expect(page.locator('button:has-text("Preferences")')).toBeVisible();
  });

  test('should display profile settings by default', async ({ page }) => {
    await expect(page.locator('text=Profile Settings')).toBeVisible();
    await expect(page.locator('input[name="name"]')).toBeVisible();
    await expect(page.locator('input[name="email"]')).toBeVisible();
  });

  test('should display user information', async ({ page }) => {
    const user = mockUsers[0];

    await expect(page.locator('input[name="name"]')).toHaveValue(user.name);
    await expect(page.locator('input[name="email"]')).toHaveValue(user.email);
  });

  test('should update profile information', async ({ page }) => {
    // Mock update API
    await page.route('**/api/v1/users/profile', async (route) => {
      if (route.request().method() === 'PATCH' || route.request().method() === 'PUT') {
        await mockApiResponse(route, {
          ...mockUsers[0],
          name: 'Updated Name',
        });
      }
    });

    await page.fill('input[name="name"]', 'Updated Name');
    await page.click('button:has-text("Save Changes")');

    // Should show success message
    await expect(page.locator('text=Profile updated')).toBeVisible();
  });

  test('should validate email format', async ({ page }) => {
    const emailInput = page.locator('input[name="email"]');
    await emailInput.fill('invalid-email');
    await emailInput.blur();

    const isValid = await emailInput.evaluate((el) => (el as HTMLInputElement).checkValidity());
    expect(isValid).toBeFalsy();
  });
});

test.describe('Security Settings', () => {
  test.beforeEach(async ({ page }) => {
    await page.route('**/api/v1/auth/me', async (route) => {
      await mockApiResponse(route, mockUsers[0]);
    });

    await login(page);
    await page.goto('/settings');
  });

  test('should display security settings tab', async ({ page }) => {
    await page.click('button:has-text("Security")');

    await expect(page.locator('text=Security Settings')).toBeVisible();
  });

  test('should display change password form', async ({ page }) => {
    await page.click('button:has-text("Security")');

    await expect(page.locator('input[name="currentPassword"]')).toBeVisible();
    await expect(page.locator('input[name="newPassword"]')).toBeVisible();
    await expect(page.locator('input[name="confirmPassword"]')).toBeVisible();
  });

  test('should change password', async ({ page }) => {
    await page.click('button:has-text("Security")');

    // Mock change password API
    await page.route('**/api/v1/auth/change-password', async (route) => {
      await mockApiResponse(route, { success: true });
    });

    await page.fill('input[name="currentPassword"]', 'oldpassword');
    await page.fill('input[name="newPassword"]', 'NewPassword123!');
    await page.fill('input[name="confirmPassword"]', 'NewPassword123!');
    await page.click('button:has-text("Change Password")');

    // Should show success message
    await expect(page.locator('text=Password changed')).toBeVisible();
  });

  test('should validate password confirmation', async ({ page }) => {
    await page.click('button:has-text("Security")');

    await page.fill('input[name="currentPassword"]', 'oldpassword');
    await page.fill('input[name="newPassword"]', 'NewPassword123!');
    await page.fill('input[name="confirmPassword"]', 'DifferentPassword123!');
    await page.click('button:has-text("Change Password")');

    // Should show error
    await expect(page.locator('text=Passwords do not match')).toBeVisible();
  });

  test('should display active sessions', async ({ page }) => {
    await page.click('button:has-text("Security")');

    await expect(page.locator('text=Active Sessions')).toBeVisible();
  });

  test('should revoke session', async ({ page }) => {
    await page.click('button:has-text("Security")');

    const revokeButton = page.locator('button:has-text("Revoke")').first();

    if (await revokeButton.isVisible()) {
      // Mock revoke API
      await page.route('**/api/v1/auth/sessions/**', async (route) => {
        if (route.request().method() === 'DELETE') {
          await mockApiResponse(route, { success: true });
        }
      });

      await revokeButton.click();
      await page.waitForTimeout(500);
    }
  });
});

test.describe('Preferences Settings', () => {
  test.beforeEach(async ({ page }) => {
    await page.route('**/api/v1/auth/me', async (route) => {
      await mockApiResponse(route, mockUsers[0]);
    });

    await login(page);
    await page.goto('/settings');
  });

  test('should display preferences tab', async ({ page }) => {
    await page.click('button:has-text("Preferences")');

    await expect(page.locator('text=Preferences')).toBeVisible();
  });

  test('should display theme toggle', async ({ page }) => {
    await page.click('button:has-text("Preferences")');

    await expect(page.locator('text=Theme')).toBeVisible();
    await expect(page.locator('text=Light')).toBeVisible();
    await expect(page.locator('text=Dark')).toBeVisible();
    await expect(page.locator('text=System')).toBeVisible();
  });

  test('should switch theme', async ({ page }) => {
    await page.click('button:has-text("Preferences")');

    // Mock update API
    await page.route('**/api/v1/users/preferences', async (route) => {
      await mockApiResponse(route, { theme: 'dark' });
    });

    await page.click('button:has-text("Dark")');
    await page.waitForTimeout(500);

    // Check if dark mode is applied
    const hasDarkMode = await page.evaluate(() => {
      return document.documentElement.classList.contains('dark');
    });

    expect(hasDarkMode).toBeTruthy();
  });

  test('should display language selector', async ({ page }) => {
    await page.click('button:has-text("Preferences")');

    await expect(page.locator('text=Language')).toBeVisible();
    await expect(page.locator('select[name="language"]')).toBeVisible();
  });

  test('should display email notification settings', async ({ page }) => {
    await page.click('button:has-text("Preferences")');

    await expect(page.locator('text=Email Notifications')).toBeVisible();
    await expect(page.locator('text=Job completed')).toBeVisible();
    await expect(page.locator('text=Job failed')).toBeVisible();
  });

  test('should toggle email notifications', async ({ page }) => {
    await page.click('button:has-text("Preferences")');

    // Mock update API
    await page.route('**/api/v1/users/preferences', async (route) => {
      await mockApiResponse(route, {
        emailNotifications: {
          jobCompleted: false,
          jobFailed: true,
        },
      });
    });

    const jobCompletedCheckbox = page.locator('input[name="jobCompleted"]');
    await jobCompletedCheckbox.check();
    await page.waitForTimeout(500);
  });
});

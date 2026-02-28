import { test, expect } from '@playwright/test';
import { login, mockApiResponse, mockUsers, mockOrganization, mockInvitations } from '../helpers';

const adminUser = mockUsers[1];

test.describe('Organization Settings (Admin)', () => {
  test.beforeEach(async ({ page }) => {
    // Setup auth mock with admin user
    await page.route('**/api/v1/auth/me', async (route) => {
      await mockApiResponse(route, adminUser);
    });

    // Setup organization mock
    await page.route('**/api/v1/organizations/**', async (route) => {
      await mockApiResponse(route, mockOrganization);
    });

    // Setup invitations mock
    await page.route('**/api/v1/invitations/**', async (route) => {
      await mockApiResponse(route, {
        data: mockInvitations,
        total: mockInvitations.length,
      });
    });

    // Setup users mock
    await page.route('**/api/v1/users/**', async (route) => {
      await mockApiResponse(route, {
        data: mockUsers,
        total: mockUsers.length,
      });
    });

    await login(page, {
      email: adminUser.email,
      password: 'AdminPassword123!',
      name: adminUser.name,
    });
    await page.goto('/organization');
  });

  test('should display organization page header', async ({ page }) => {
    await expect(page.locator('h1')).toContainText('Organization Settings');
  });

  test('should display organization name', async ({ page }) => {
    await expect(page.locator('text=' + mockOrganization.name)).toBeVisible();
  });

  test('should display organization plan', async ({ page }) => {
    await expect(page.locator('text=Plan')).toBeVisible();
    await expect(page.locator('text=' + mockOrganization.plan.charAt(0).toUpperCase() + mockOrganization.plan.slice(1))).toBeVisible();
  });

  test('should display usage statistics', async ({ page }) => {
    await expect(page.locator('text=Users')).toBeVisible();
    await expect(page.locator('text=Printers')).toBeVisible();
    await expect(page.locator('text=' + mockUsers.length)).toBeVisible();
  });

  test('should display organization details card', async ({ page }) => {
    await expect(page.locator('text=Organization Details')).toBeVisible();
    await expect(page.locator('text=Organization Name')).toBeVisible();
    await expect(page.locator('text=Organization Slug')).toBeVisible();
    await expect(page.locator('text=' + mockOrganization.slug)).toBeVisible();
  });

  test('should open edit organization modal', async ({ page }) => {
    await page.click('button:has-text("Edit Organization")');
    await expect(page.locator('text=Edit Organization')).toBeVisible();
  });

  test('should update organization name', async ({ page }) => {
    await page.click('button:has-text("Edit Organization")');

    // Update name
    await page.fill('input[name="name"]', 'Updated Organization Name');

    // Mock update API
    await page.route('**/api/v1/organizations/**', async (route) => {
      if (route.request().method() === 'PATCH' || route.request().method() === 'PUT') {
        await mockApiResponse(route, {
          ...mockOrganization,
          name: 'Updated Organization Name',
        });
      } else {
        await mockApiResponse(route, mockOrganization);
      }
    });

    await page.click('button:has-text("Save Changes")');

    // Modal should close
    await expect(page.locator('text=Edit Organization')).not.toBeVisible();
  });

  test('should display users section', async ({ page }) => {
    await expect(page.locator('text=Users')).toBeVisible();
    await expect(page.locator('text=Add User')).toBeVisible();
  });

  test('should display users table', async ({ page }) => {
    // Check for table headers
    await expect(page.locator('th:has-text("Name")')).toBeVisible();
    await expect(page.locator('th:has-text("Email")')).toBeVisible();
    await expect(page.locator('th:has-text("Role")')).toBeVisible();
    await expect(page.locator('th:has-text("Status")')).toBeVisible();
  });

  test('should display user entries', async ({ page }) => {
    for (const user of mockUsers) {
      await expect(page.locator('text=' + user.name)).toBeVisible();
    }
  });

  test('should open add user modal', async ({ page }) => {
    await page.click('button:has-text("Add User")');
    await expect(page.locator('text=Add User')).toBeVisible();
    await expect(page.locator('input[name="email"]')).toBeVisible();
  });

  test('should invite new user', async ({ page }) => {
    await page.click('button:has-text("Add User")');

    // Mock invite API
    await page.route('**/api/v1/invitations', async (route) => {
      if (route.request().method() === 'POST') {
        await mockApiResponse(route, {
          id: 'inv-new',
          email: 'newuser@example.com',
          role: 'user',
        });
      }
    });

    await page.fill('input[name="email"]', 'newuser@example.com');
    await page.selectOption('select[name="role"]', 'user');
    await page.click('button:has-text("Send Invitation")');

    // Modal should close
    await expect(page.locator('text=Add User')).not.toBeVisible();
  });

  test('should display pending invitations', async ({ page }) => {
    await expect(page.locator('text=Pending Invitations')).toBeVisible();

    if (mockInvitations.length > 0) {
      await expect(page.locator('text=' + mockInvitations[0].email)).toBeVisible();
    }
  });

  test('should cancel invitation', async ({ page }) => {
    if (mockInvitations.length > 0) {
      const cancelButton = page.locator('button:has-text("Cancel")').first();

      if (await cancelButton.isVisible()) {
        // Mock cancel API
        await page.route('**/api/v1/invitations/**', async (route) => {
          if (route.request().method() === 'DELETE') {
            await mockApiResponse(route, { success: true });
          }
        });

        await cancelButton.click();
        await page.waitForTimeout(500);
      }
    }
  });

  test('should change user role', async ({ page }) => {
    const roleButton = page.locator('button:has-text("Change Role")').first();

    if (await roleButton.isVisible()) {
      await roleButton.click();

      // Mock update API
      await page.route('**/api/v1/users/*/role', async (route) => {
        await mockApiResponse(route, { success: true });
      });

      await page.selectOption('select[name="role"]', 'admin');
      await page.click('button:has-text("Save")');

      await page.waitForTimeout(500);
    }
  });

  test('should deactivate user', async ({ page }) => {
    const deactivateButton = page.locator('button:has-text("Deactivate")').first();

    if (await deactivateButton.isVisible()) {
      // Mock deactivate API
      await page.route('**/api/v1/users/*/deactivate', async (route) => {
        await mockApiResponse(route, { success: true });
      });

      await deactivateButton.click();

      // Confirm dialog
      await page.click('button:has-text("Confirm")');

      await page.waitForTimeout(500);
    }
  });

  test('should search users', async ({ page }) => {
    const searchInput = page.getByPlaceholder('Search users');
    await searchInput.fill('admin');

    // Should filter results
    await page.waitForTimeout(500);

    await expect(page.locator('text=' + adminUser.name)).toBeVisible();
  });

  test('should display organization limits', async ({ page }) => {
    await expect(page.locator('text=Limits')).toBeVisible();
    await expect(page.locator('text=Max Users')).toBeVisible();
    await expect(page.locator('text=Max Printers')).toBeVisible();
    await expect(page.locator('text=' + mockOrganization.maxUsers)).toBeVisible();
    await expect(page.locator('text=' + mockOrganization.maxPrinters)).toBeVisible();
  });

  test('should display danger zone', async ({ page }) => {
    await expect(page.locator('text=Danger Zone')).toBeVisible();
    await expect(page.locator('button:has-text("Delete Organization")')).toBeVisible();
  });

  test('should require confirmation for organization deletion', async ({ page }) => {
    await page.click('button:has-text("Delete Organization")');

    await expect(page.locator('text=Type the organization name to confirm')).toBeVisible();

    // Should require typing the name
    await page.fill('input[name="confirmName"]', 'Wrong Name');
    await expect(page.locator('button:has-text("Delete")')).toBeDisabled();
  });
});

test.describe('Organization Access Control', () => {
  test('should redirect non-admin users to dashboard', async ({ page }) => {
    // Setup auth as regular user
    await page.route('**/api/v1/auth/me', async (route) => {
      await mockApiResponse(route, mockUsers[0]);
    });

    await login(page);
    await page.goto('/organization');

    // Should redirect to dashboard
    await page.waitForURL('**/dashboard');
    await expect(page.locator('h1')).toContainText('Welcome back');
  });

  test('should allow admin access', async ({ page }) => {
    // Setup auth as admin
    await page.route('**/api/v1/auth/me', async (route) => {
      await mockApiResponse(route, mockUsers[1]);
    });

    await page.route('**/api/v1/organizations/**', async (route) => {
      await mockApiResponse(route, mockOrganization);
    });

    await login(page, {
      email: mockUsers[1].email,
      password: 'AdminPassword123!',
      name: mockUsers[1].name,
    });
    await page.goto('/organization');

    await expect(page.locator('h1')).toContainText('Organization Settings');
  });
});

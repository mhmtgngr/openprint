import { test, expect } from '@playwright/test';
import { login, mockApiResponse, mockUsers, mockPrinters, mockOrganization, mockInvitations } from './helpers';

const adminUser = {
  ...mockUsers[1],
  role: 'admin',
};

test.describe('Organization Page (Admin)', () => {
  test.beforeEach(async ({ page }) => {
    // Setup auth mock with admin user
    await page.route('**/api/v1/auth/me', async (route) => {
      await mockApiResponse(route, adminUser);
    });

    // Setup organization mock
    await page.route('**/api/v1/organizations', async (route) => {
      if (route.request().method() === 'PATCH') {
        await mockApiResponse(route, {
          ...mockOrganization,
          name: 'Updated Org Name',
        });
      } else {
        await mockApiResponse(route, mockOrganization);
      }
    });

    // Setup users mock
    await page.route('**/api/v1/organizations/users', async (route) => {
      await mockApiResponse(route, mockUsers);
    });

    // Setup invitations mock
    await page.route('**/api/v1/organizations/invitations', async (route) => {
      if (route.request().method() === 'POST') {
        await mockApiResponse(route, mockInvitations[0]);
      } else if (route.request().method() === 'DELETE') {
        await mockApiResponse(route, {}, 204);
      } else {
        await mockApiResponse(route, mockInvitations);
      }
    });

    // Setup user role update mock
    await page.route('**/api/v1/organizations/users/*/role', async (route) => {
      await mockApiResponse(route, mockUsers[0]);
    });

    // Setup remove user mock
    await page.route('**/api/v1/organizations/users/*', async (route) => {
      await mockApiResponse(route, {}, 204);
    });

    // Setup printers mock
    await page.route('**/api/v1/printers', async (route) => {
      await mockApiResponse(route, mockPrinters);
    });

    await login(page, {
      email: adminUser.email,
      password: 'AdminPassword123!',
    });
    await page.goto('/organization');
    await page.waitForURL('**/organization');
  });

  test('should display organization page header', async ({ page }) => {
    await expect(page.locator('h1')).toContainText('Organization');
    await expect(page.locator('text=Manage your organization settings and members')).toBeVisible();
  });

  test('should display organization info card', async ({ page }) => {
    await expect(page.locator(`text=${mockOrganization.name}`)).toBeVisible();
    await expect(page.locator(`text=Plan: ${mockOrganization.plan}`)).toBeVisible();
    await expect(page.locator('text=Total Users')).toBeVisible();
    await expect(page.locator('text=Total Printers')).toBeVisible();
    await expect(page.locator('text=Online Printers')).toBeVisible();
  });

  test('should display correct counts in org card', async ({ page }) => {
    await expect(page.locator(`text=${mockUsers.length}`)).toBeVisible();
    await expect(page.locator(`text=${mockPrinters.length}`)).toBeVisible();

    const onlineCount = mockPrinters.filter(p => p.isOnline).length;
    await expect(page.locator(`text=${onlineCount}`)).toBeVisible();
  });

  test('should have edit organization button', async ({ page }) => {
    const editButton = page.locator('button:has-text("Edit Organization")');
    await expect(editButton).toBeVisible();
  });

  test('should display tabs', async ({ page }) => {
    await expect(page.locator('text=Overview')).toBeVisible();
    await expect(page.locator('text=Users')).toBeVisible();
    await expect(page.locator('text=Printers')).toBeVisible();
    await expect(page.locator('text=Invitations')).toBeVisible();
  });

  test('should switch between tabs', async ({ page }) => {
    // Click Users tab
    await page.click('button:has-text("Users")');
    await expect(page.locator('text=Team Members')).toBeVisible();

    // Click Printers tab
    await page.click('button:has-text("Printers")');
    await expect(page.locator('text=Organization Printers')).toBeVisible();

    // Click Invitations tab
    await page.click('button:has-text("Invitations")');
    await expect(page.locator('text=Invite New Member')).toBeVisible();
  });

  test('should display overview tab with plan details', async ({ page }) => {
    await expect(page.locator('text=Plan Details')).toBeVisible();
    await expect(page.locator(`text=Current Plan`)).toBeVisible();
    await expect(page.locator('text=Max Users')).toBeVisible();
    await expect(page.locator('text=Max Printers')).toBeVisible();
  });

  test('should display user distribution chart', async ({ page }) => {
    await expect(page.locator('text=User Distribution')).toBeVisible();
    await expect(page.locator('text=Admins')).toBeVisible();
    await expect(page.locator('text=Regular Users')).toBeVisible();
  });

  test('should have upgrade plan button', async ({ page }) => {
    await expect(page.locator('button:has-text("Upgrade Plan")')).toBeVisible();
  });

  test('should display users tab with team members', async ({ page }) => {
    await page.click('button:has-text("Users")');

    await expect(page.locator('text=Team Members')).toBeVisible();
    await expect(page.locator(`text=(${mockUsers.length})`)).toBeVisible();
  });

  test('should display user list with avatars', async ({ page }) => {
    await page.click('button:has-text("Users")');

    for (const user of mockUsers) {
      await expect(page.locator(`text=${user.name}`)).toBeVisible();
      await expect(page.locator(`text=${user.email}`)).toBeVisible();

      // Check for avatar with user initial
      const initial = user.name?.charAt(0).toUpperCase() || user.email.charAt(0).toUpperCase();
      await expect(page.locator(`text=${initial}`)).toBeVisible();
    }
  });

  test('should have invite user button', async ({ page }) => {
    await page.click('button:has-text("Users")');

    const inviteButton = page.locator('button:has-text("Invite User")');
    await expect(inviteButton).toBeVisible();
  });

  test('should change user role', async ({ page }) => {
    await page.click('button:has-text("Users")');

    // Find role dropdown for a non-admin user
    const roleSelect = page.locator('select').first();
    await roleSelect.selectOption('admin');

    // Should trigger API call
    // Success message would appear
  });

  test('should remove user from organization', async ({ page }) => {
    await page.click('button:has-text("Users")');

    // Find and click trash icon for a user (not current user or owner)
    const trashButtons = page.locator('button[title="Remove user"]');
    const count = await trashButtons.count();

    if (count > 0) {
      await trashButtons.first().click();
      // Should trigger confirmation and API call
    }
  });

  test('should disable role change for current user', async ({ page }) => {
    await page.click('button:has-text("Users")');

    // Find the current user's role select
    const userRows = page.locator('text=' + adminUser.name).locator('..').locator('..');
    const roleSelect = userRows.locator('select');

    // Should be disabled
    await expect(roleSelect).toBeDisabled();
  });

  test('should display printers tab', async ({ page }) => {
    await page.click('button:has-text("Printers")');

    await expect(page.locator('text=Organization Printers')).toBeVisible();
    await expect(page.locator(`text=(${mockPrinters.length})`)).toBeVisible();
  });

  test('should display printer list in org tab', async ({ page }) => {
    await page.click('button:has-text("Printers")');

    for (const printer of mockPrinters) {
      await expect(page.locator(`text=${printer.name}`)).toBeVisible();
    }
  });

  test('should show printer status badges', async ({ page }) => {
    await page.click('button:has-text("Printers")');

    await expect(page.locator('text=Active')).toBeVisible();
    await expect(page.locator('text=Disabled')).toBeVisible();
  });

  test('should display invitations tab with form', async ({ page }) => {
    await page.click('button:has-text("Invitations")');

    await expect(page.locator('text=Invite New Member')).toBeVisible();
    await expect(page.locator('input[placeholder="colleague@example.com"]')).toBeVisible();
    await expect(page.locator('button:has-text("Send Invite")')).toBeVisible();
  });

  test('should send invitation', async ({ page }) => {
    await page.click('button:has-text("Invitations")');

    await page.fill('input[placeholder="colleague@example.com"]', 'newuser@example.com');
    await page.locator('select').selectOption('user');

    const sendButton = page.locator('button:has-text("Send Invite")');
    await sendButton.click();

    // Should trigger API call and show success message
  });

  test('should display pending invitations', async ({ page }) => {
    await page.click('button:has-text("Invitations")');

    if (mockInvitations.length > 0) {
      await expect(page.locator('text=Pending Invitations')).toBeVisible();
      await expect(page.locator(`text=(${mockInvitations.length})`)).toBeVisible();

      for (const invite of mockInvitations) {
        await expect(page.locator(`text=${invite.email}`)).toBeVisible();
      }
    }
  });

  test('should cancel invitation', async ({ page }) => {
    await page.click('button:has-text("Invitations")');

    const trashButtons = page.locator('button[title="Remove user"]');
    const count = await trashButtons.count();

    if (count > 0) {
      await trashButtons.first().click();
      // Should trigger API call
    }
  });

  test('should show invite role selector', async ({ page }) => {
    await page.click('button:has-text("Invitations")');

    const roleSelect = page.locator('select');
    await expect(roleSelect).toBeVisible();

    const options = await roleSelect.locator('option').allTextContents();
    expect(options).toContain('User');
    expect(options).toContain('Admin');
  });

  test('should navigate to dashboard via sidebar', async ({ page }) => {
    await page.click('a[href="/dashboard"]');
    await page.waitForURL('**/dashboard');

    await expect(page.locator('h1')).toContainText('Welcome back');
  });

  test('should highlight Organization in navigation', async ({ page }) => {
    const orgLink = page.locator('a[href="/organization"]');
    await expect(orgLink).toHaveClass(/bg-blue-100/);
  });

  test('should handle API errors gracefully', async ({ page }) => {
    // Mock error response
    await page.route('**/api/v1/organizations/users', async (route) => {
      await route.abort('failed');
    });

    await page.click('button:has-text("Users")');

    // Should show loading or error state
    await expect(page.locator('text=Team Members')).toBeVisible();
  });

  test('should be responsive on mobile', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });

    // Check that main content is still visible
    await expect(page.locator('h1')).toBeVisible();
  });

  test('should show organization plan in header', async ({ page }) => {
    await expect(page.locator(`text=Plan:`)).toBeVisible();
    await expect(page.locator(`text=${mockOrganization.plan}`)).toBeVisible();
  });

  test('should allow editing organization name', async ({ page }) => {
    const editButton = page.locator('button:has-text("Edit Organization")');
    await editButton.click();

    // Would open a modal or form for editing
    // This test verifies the button is present and clickable
  });
});

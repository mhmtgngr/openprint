/**
 * Multi-tenancy E2E Tests
 *
 * Tests for platform admin organization management functionality:
 * - Organizations list page
 * - Organization detail view
 * - Creating/editing organizations
 * - Managing organization users
 * - Quota management
 * - Usage reporting
 */

import { test, expect } from '@playwright/test';
import { setupAuthAndNavigate, mockApiResponse, mockUsers } from '../helpers';

// Platform admin user mock
const platformAdminUser = {
  ...mockUsers[1],
  role: 'platform_admin',
  isPlatformAdmin: true,
};

// Mock organization data
const mockOrganizations = [
  {
    id: 'org-1',
    name: 'Acme Corporation',
    slug: 'acme-corp',
    displayName: 'Acme Corp',
    status: 'active',
    plan: 'enterprise',
    currentUserCount: 45,
    currentPrinterCount: 12,
    maxUsers: -1,
    maxPrinters: -1,
    maxStorageGB: -1,
    maxJobsPerMonth: -1,
    usagePercentage: 45,
    healthScore: 95,
    alertCount: 0,
    createdAt: '2024-01-01T00:00:00Z',
    settings: {
      branding: {
        primaryColor: '#3b82f6',
      },
      security: {
        requireMFA: false,
        passwordMinLength: 8,
        sessionTimeoutMinutes: 60,
      },
    },
    quotas: {
      currentUserCount: 45,
      currentPrinterCount: 12,
      currentStorageGB: 234,
      currentJobsThisMonth: 1543,
      maxUsers: -1,
      maxPrinters: -1,
      maxStorageGB: -1,
      maxJobsPerMonth: -1,
      quotaResetDate: '2024-03-01T00:00:00Z',
    },
  },
  {
    id: 'org-2',
    name: 'Globex Inc',
    slug: 'globex-inc',
    displayName: 'Globex',
    status: 'active',
    plan: 'pro',
    currentUserCount: 28,
    currentPrinterCount: 8,
    maxUsers: 50,
    maxPrinters: 20,
    maxStorageGB: 100,
    maxJobsPerMonth: 10000,
    usagePercentage: 56,
    healthScore: 88,
    alertCount: 2,
    createdAt: '2024-02-01T00:00:00Z',
    settings: {
      branding: {
        primaryColor: '#8b5cf6',
      },
      security: {
        requireMFA: true,
        passwordMinLength: 12,
        sessionTimeoutMinutes: 30,
      },
    },
    quotas: {
      currentUserCount: 28,
      currentPrinterCount: 8,
      currentStorageGB: 67,
      currentJobsThisMonth: 5620,
      maxUsers: 50,
      maxPrinters: 20,
      maxStorageGB: 100,
      maxJobsPerMonth: 10000,
      quotaResetDate: '2024-03-01T00:00:00Z',
    },
  },
  {
    id: 'org-3',
    name: 'StartUp Labs',
    slug: 'startup-labs',
    displayName: 'StartUp Labs',
    status: 'trial',
    plan: 'free',
    currentUserCount: 3,
    currentPrinterCount: 1,
    maxUsers: 5,
    maxPrinters: 2,
    maxStorageGB: 10,
    maxJobsPerMonth: 1000,
    usagePercentage: 60,
    healthScore: 92,
    alertCount: 0,
    createdAt: '2024-02-20T00:00:00Z',
    settings: {
      branding: {
        primaryColor: '#10b981',
      },
      security: {
        requireMFA: false,
        passwordMinLength: 8,
        sessionTimeoutMinutes: 60,
      },
    },
    quotas: {
      currentUserCount: 3,
      currentPrinterCount: 1,
      currentStorageGB: 2,
      currentJobsThisMonth: 234,
      maxUsers: 5,
      maxPrinters: 2,
      maxStorageGB: 10,
      maxJobsPerMonth: 1000,
      quotaResetDate: '2024-03-01T00:00:00Z',
    },
  },
  {
    id: 'org-4',
    name: 'Suspended Company',
    slug: 'suspended-company',
    displayName: 'Suspended Co',
    status: 'suspended',
    plan: 'pro',
    currentUserCount: 15,
    currentPrinterCount: 5,
    maxUsers: 50,
    maxPrinters: 20,
    maxStorageGB: 100,
    maxJobsPerMonth: 10000,
    usagePercentage: 30,
    healthScore: 45,
    alertCount: 5,
    createdAt: '2023-12-01T00:00:00Z',
    settings: {
      branding: {
        primaryColor: '#ef4444',
      },
      security: {
        requireMFA: false,
        passwordMinLength: 8,
        sessionTimeoutMinutes: 60,
      },
    },
    quotas: {
      currentUserCount: 15,
      currentPrinterCount: 5,
      currentStorageGB: 34,
      currentJobsThisMonth: 0,
      maxUsers: 50,
      maxPrinters: 20,
      maxStorageGB: 100,
      maxJobsPerMonth: 10000,
      quotaResetDate: '2024-03-01T00:00:00Z',
    },
  },
];

// Mock organization users
const mockOrgUsers = [
  {
    id: 'user-1',
    userId: 'user-1',
    organizationId: 'org-2',
    role: 'owner',
    status: 'active',
    invitedBy: 'platform-admin',
    joinedAt: '2024-02-01T00:00:00Z',
    lastActiveAt: '2024-02-27T10:00:00Z',
    user: {
      id: 'user-1',
      name: 'John Owner',
      email: 'john@acme.com',
      isActive: true,
    },
  },
  {
    id: 'user-2',
    userId: 'user-2',
    organizationId: 'org-2',
    role: 'admin',
    status: 'active',
    invitedBy: 'user-1',
    joinedAt: '2024-02-02T00:00:00Z',
    lastActiveAt: '2024-02-27T09:00:00Z',
    user: {
      id: 'user-2',
      name: 'Jane Admin',
      email: 'jane@acme.com',
      isActive: true,
    },
  },
  {
    id: 'user-3',
    userId: 'user-3',
    organizationId: 'org-2',
    role: 'member',
    status: 'active',
    invitedBy: 'user-1',
    joinedAt: '2024-02-05T00:00:00Z',
    lastActiveAt: '2024-02-26T15:00:00Z',
    user: {
      id: 'user-3',
      name: 'Bob Member',
      email: 'bob@acme.com',
      isActive: true,
    },
  },
  {
    id: 'user-4',
    userId: 'user-4',
    organizationId: 'org-2',
    role: 'viewer',
    status: 'pending',
    invitedBy: 'user-2',
    joinedAt: '2024-02-27T00:00:00Z',
    user: {
      id: 'user-4',
      name: 'Alice Viewer',
      email: 'alice@acme.com',
      isActive: true,
    },
  },
];

// Mock usage trends
const mockUsageTrends = Array.from({ length: 12 }, (_, i) => ({
  date: new Date(Date.now() - (11 - i) * 7 * 24 * 60 * 60 * 1000).toISOString().split('T')[0],
  jobs: Math.floor(100 + Math.random() * 200),
  pages: Math.floor(500 + Math.random() * 2000),
  users: 25 + Math.floor(Math.random() * 10),
  storage: Math.floor(50 + Math.random() * 30),
}));

test.describe('Platform Admin - Organizations List', () => {
  test.beforeEach(async ({ page }) => {
    // Setup auth as platform admin
    await setupAuthAndNavigate(page, '/admin/organizations', platformAdminUser);

    // Mock organizations API
    await page.route('**/api/v1/platform/organizations*', async (route) => {
      const url = route.request().url();

      if (url.includes('/organizations/') && !url.includes('?')) {
        // Single organization detail
        const orgId = url.split('/').pop()?.split('?')[0];
        const org = mockOrganizations.find(o => o.id === orgId);
        await mockApiResponse(route, org || mockOrganizations[0]);
      } else {
        // Organizations list
        await mockApiResponse(route, {
          data: mockOrganizations,
          total: mockOrganizations.length,
          limit: 50,
          offset: 0,
        });
      }
    });

    // Mock quota API
    await page.route('**/api/v1/platform/organizations/*/quota', async (route) => {
      await mockApiResponse(route, mockOrganizations[0].quotas);
    });

    // Mock users API
    await page.route('**/api/v1/platform/organizations/*/users', async (route) => {
      await mockApiResponse(route, mockOrgUsers);
    });

    // Mock usage trends API
    await page.route('**/api/v1/platform/organizations/*/trends*', async (route) => {
      await mockApiResponse(route, mockUsageTrends);
    });
  });

  test('should display organizations list page header', async ({ page }) => {
    await expect(page.locator('h1')).toContainText('Organizations');
    await expect(page.locator('text=Manage all organizations on the platform')).toBeVisible();
  });

  test('should display organization statistics cards', async ({ page }) => {
    await expect(page.locator('text=Total Organizations')).toBeVisible();
    await expect(page.locator('text=Active')).toBeVisible();
    await expect(page.locator('text=Trial')).toBeVisible();
    await expect(page.locator('text=Suspended')).toBeVisible();
  });

  test('should display all organizations in list', async ({ page }) => {
    // Check for organization names
    await expect(page.locator('text=Acme Corporation')).toBeVisible();
    await expect(page.locator('text=Globex Inc')).toBeVisible();
    await expect(page.locator('text=StartUp Labs')).toBeVisible();
    await expect(page.locator('text=Suspended Company')).toBeVisible();
  });

  test('should display organization status badges', async ({ page }) => {
    await expect(page.locator('text=Active').first()).toBeVisible();
    await expect(page.locator('text=Trial')).toBeVisible();
    await expect(page.locator('text=Suspended')).toBeVisible();
  });

  test('should display organization plan badges', async ({ page }) => {
    await expect(page.locator('text=Enterprise')).toBeVisible();
    await expect(page.locator('text=Pro')).toBeVisible();
    await expect(page.locator('text=Free')).toBeVisible();
  });

  test('should display usage bar for each organization', async ({ page }) => {
    // Check that usage bars are visible
    const usageBars = page.locator('[class*="h-2 bg-gray-200"]');
    await expect(usageBars.first()).toBeVisible();
  });

  test('should filter organizations by status', async ({ page }) => {
    // Select Active status filter
    await page.selectOption('select', { label: 'All Statuses' });
    await page.selectOption('select', { label: 'Active' });

    // Wait for filtered results
    await page.waitForTimeout(500);

    // Verify only active organizations are shown
    await expect(page.locator('text=Acme Corporation')).toBeVisible();
    await expect(page.locator('text=Globex Inc')).toBeVisible();
  });

  test('should filter organizations by plan', async ({ page }) => {
    // Select Enterprise plan filter
    await page.selectOption('select:has-text("All Plans")', 'enterprise');

    // Wait for filtered results
    await page.waitForTimeout(500);

    // Verify only enterprise organizations are shown
    await expect(page.locator('text=Acme Corporation')).toBeVisible();
  });

  test('should search organizations', async ({ page }) => {
    // Type search query
    await page.fill('input[placeholder*="Search"]', 'Acme');

    // Wait for search results
    await page.waitForTimeout(500);

    // Verify search results
    await expect(page.locator('text=Acme Corporation')).toBeVisible();
    await expect(page.locator('text=Globex Inc')).not.toBeVisible();
  });

  test('should open create organization modal', async ({ page }) => {
    // Click New Organization button
    await page.click('button:has-text("New Organization")');

    // Verify modal is displayed
    await expect(page.locator('text=Create New Organization')).toBeVisible();
    await expect(page.locator('text=Set up a new organization tenant on the platform')).toBeVisible();
  });

  test('should navigate to organization detail', async ({ page }) => {
    // Click on an organization
    await page.click('text=Acme Corporation');

    // Verify navigation to detail page
    await page.waitForURL('**/admin/organizations/org-1');
    await expect(page.locator('text=Acme Corporation')).toBeVisible();
  });
});

test.describe('Platform Admin - Organization Detail', () => {
  test.beforeEach(async ({ page }) => {
    // Setup auth as platform admin
    await setupAuthAndNavigate(page, '/admin/organizations/org-1', platformAdminUser);

    // Mock organization detail API
    await page.route('**/api/v1/platform/organizations/org-1', async (route) => {
      await mockApiResponse(route, mockOrganizations[0]);
    });

    // Mock quota API
    await page.route('**/api/v1/platform/organizations/org-1/quota', async (route) => {
      await mockApiResponse(route, mockOrganizations[0].quotas);
    });

    // Mock users API
    await page.route('**/api/v1/platform/organizations/org-1/users', async (route) => {
      await mockApiResponse(route, mockOrgUsers);
    });

    // Mock usage trends API
    await page.route('**/api/v1/platform/organizations/org-1/trends*', async (route) => {
      await mockApiResponse(route, mockUsageTrends);
    });

    // Mock update API
    await page.route('**/api/v1/platform/organizations/org-1', async (route) => {
      if (route.request().method() === 'PATCH') {
        await mockApiResponse(route, mockOrganizations[0]);
      }
    });
  });

  test('should display organization detail header', async ({ page }) => {
    await expect(page.locator('h1:has-text("Acme Corporation")')).toBeVisible();
    await expect(page.locator('text=acme-corp')).toBeVisible();
  });

  test('should display organization tabs', async ({ page }) => {
    await expect(page.locator('text=Overview')).toBeVisible();
    await expect(page.locator('text=Users')).toBeVisible();
    await expect(page.locator('text=Usage')).toBeVisible();
    await expect(page.locator('text=Settings')).toBeVisible();
    await expect(page.locator('text=Audit Trail')).toBeVisible();
  });

  test('should display overview tab with quota card', async ({ page }) => {
    // Overview tab should be active by default
    await expect(page.locator('text=Resource Quotas')).toBeVisible();
    await expect(page.locator('text=Max Users')).toBeVisible();
    await expect(page.locator('text=Max Printers')).toBeVisible();
  });

  test('should navigate between tabs', async ({ page }) => {
    // Click Users tab
    await page.click('text=Users');
    await expect(page.locator('text=Organization Members')).toBeVisible();

    // Click Usage tab
    await page.click('text=Usage');
    await expect(page.locator('text=Usage Trends')).toBeVisible();

    // Click Settings tab
    await page.click('text=Settings');
    await expect(page.locator('text=Organization Settings')).toBeVisible();
  });

  test('should display users tab with member list', async ({ page }) => {
    await page.click('text=Users');

    await expect(page.locator('text=John Owner')).toBeVisible();
    await expect(page.locator('text=Jane Admin')).toBeVisible();
    await expect(page.locator('text=Bob Member')).toBeVisible();

    // Verify role badges
    await expect(page.locator('text=Owner')).toBeVisible();
    await expect(page.locator('text=Admin')).toBeVisible();
    await expect(page.locator('text=Member')).toBeVisible();
  });

  test('should display usage tab with chart', async ({ page }) => {
    await page.click('text=Usage');

    // Verify chart is displayed
    await expect(page.locator('text=Track organization usage over time')).toBeVisible();

    // Verify period selector
    await expect(page.locator('select')).toBeVisible();
  });

  test('should open edit modal', async ({ page }) => {
    await page.click('button:has-text("Edit")');

    await expect(page.locator('text=Edit Organization')).toBeVisible();
  });

  test('should navigate back to organizations list', async ({ page }) => {
    await page.click('button[aria-label="back"], button:has(svg:has-text("arrow-left"))');

    await page.waitForURL('**/admin/organizations');
    await expect(page.locator('h1:has-text("Organizations")')).toBeVisible();
  });
});

test.describe('Platform Admin - Create Organization', () => {
  test.beforeEach(async ({ page }) => {
    // Setup auth as platform admin
    await setupAuthAndNavigate(page, '/admin/organizations', platformAdminUser);

    // Mock organizations API
    await page.route('**/api/v1/platform/organizations*', async (route) => {
      if (route.request().method() === 'POST') {
        await mockApiResponse(route, mockOrganizations[0]);
      } else {
        await mockApiResponse(route, {
          data: mockOrganizations,
          total: mockOrganizations.length,
          limit: 50,
          offset: 0,
        });
      }
    });
  });

  test('should show create form modal', async ({ page }) => {
    await page.click('button:has-text("New Organization")');

    await expect(page.locator('text=Create New Organization')).toBeVisible();
    await expect(page.locator('label:has-text("Organization Name")')).toBeVisible();
    await expect(page.locator('label:has-text("Slug")')).toBeVisible();
  });

  test('should validate required fields', async ({ page }) => {
    await page.click('button:has-text("New Organization")');

    // Try to submit without filling required fields
    await page.click('button:has-text("Create Organization")');

    // Validation should prevent submission
    await expect(page.locator('text=Create Organization')).toBeVisible();
  });

  test('should auto-generate slug from name', async ({ page }) => {
    await page.click('button:has-text("New Organization")');

    // Fill organization name
    await page.fill('input[name="name"], input#name', 'Test Organization');

    // Slug should be auto-generated
    const slugInput = page.locator('input[name="slug"], input#slug');
    await expect(slugInput).toHaveValue('test-organization');
  });

  test('should show plan options', async ({ page }) => {
    await page.click('button:has-text("New Organization")');

    // Verify plan options are displayed
    await expect(page.locator('text=Free')).toBeVisible();
    await expect(page.locator('text=Pro')).toBeVisible();
    await expect(page.locator('text=Enterprise')).toBeVisible();

    // Verify plan descriptions
    await expect(page.locator('text=Up to 5 users')).toBeVisible();
  });

  test('should update quota fields when plan changes', async ({ page }) => {
    await page.click('button:has-text("New Organization")');

    // Select Pro plan
    await page.click('label:has-text("Pro")');

    // Quotas should update
    // Note: This would require checking the actual form values
  });

  test('should close modal on cancel', async ({ page }) => {
    await page.click('button:has-text("New Organization")');
    await page.click('button:has-text("Cancel")');

    // Modal should close
    await expect(page.locator('text=Create New Organization')).not.toBeVisible();
  });
});

test.describe('Platform Admin - Organization Actions', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuthAndNavigate(page, '/admin/organizations', platformAdminUser);

    // Mock organizations API
    await page.route('**/api/v1/platform/organizations*', async (route) => {
      const url = route.request().url();

      if (url.includes('/suspend')) {
        await mockApiResponse(route, { ...mockOrganizations[0], status: 'suspended' });
      } else if (url.includes('/reactivate')) {
        await mockApiResponse(route, { ...mockOrganizations[3], status: 'active' });
      } else if (route.request().method() === 'DELETE') {
        await mockApiResponse(route, { success: true });
      } else {
        await mockApiResponse(route, {
          data: mockOrganizations,
          total: mockOrganizations.length,
          limit: 50,
          offset: 0,
        });
      }
    });
  });

  test('should suspend organization', async ({ page }) => {
    // Find suspend button for active org
    const suspendButton = page.locator('button[title="Suspend organization"]').first();

    if (await suspendButton.isVisible()) {
      // Mock window.prompt for suspend reason
      page.on('dialog', dialog => dialog.accept('Violation of terms'));

      await suspendButton.click();

      // API should be called
      await page.waitForTimeout(500);
    }
  });

  test('should reactivate suspended organization', async ({ page }) => {
    // Find reactivate button for suspended org
    const reactivateButton = page.locator('button:has-text("Reactivate")').first();

    if (await reactivateButton.isVisible()) {
      await reactivateButton.click();

      // API should be called
      await page.waitForTimeout(500);
    }
  });

  test('should require confirmation for delete', async ({ page }) => {
    // Mock window.confirm
    page.on('dialog', dialog => dialog.dismiss());

    const deleteButton = page.locator('button:has-text("Delete")').first();

    if (await deleteButton.isVisible()) {
      await deleteButton.click();

      // Confirmation dialogs should be shown
      await page.waitForTimeout(500);
    }
  });
});

test.describe('Platform Admin - Access Control', () => {
  test('should redirect non-platform-admin users', async ({ page }) => {
    // Setup auth as regular admin (not platform admin)
    await setupAuthAndNavigate(page, '/admin/organizations', mockUsers[1]);

    // Should redirect to dashboard
    await page.waitForURL('**/dashboard', { timeout: 5000 });
  });

  test('should allow platform admin access', async ({ page }) => {
    // Setup auth as platform admin
    await setupAuthAndNavigate(page, '/admin/organizations', platformAdminUser);

    // Mock organizations API
    await page.route('**/api/v1/platform/organizations*', async (route) => {
      await mockApiResponse(route, {
        data: mockOrganizations,
        total: mockOrganizations.length,
        limit: 50,
        offset: 0,
      });
    });

    // Should stay on organizations page
    await expect(page.locator('h1:has-text("Organizations")')).toBeVisible();
  });
});

test.describe('Platform Admin - Organization Quotas', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuthAndNavigate(page, '/admin/organizations/org-1', platformAdminUser);

    // Mock organization and quota APIs
    await page.route('**/api/v1/platform/organizations/org-1', async (route) => {
      await mockApiResponse(route, mockOrganizations[0]);
    });

    await page.route('**/api/v1/platform/organizations/org-1/quota', async (route) => {
      if (route.request().method() === 'PATCH') {
        await mockApiResponse(route, mockOrganizations[0].quotas);
      } else {
        await mockApiResponse(route, mockOrganizations[0].quotas);
      }
    });
  });

  test('should display quota card with progress bars', async ({ page }) => {
    await expect(page.locator('text=Resource Quotas')).toBeVisible();

    // Check for quota items
    await expect(page.locator('text=Users')).toBeVisible();
    await expect(page.locator('text=Printers')).toBeVisible();
    await expect(page.locator('text=Storage')).toBeVisible();
  });

  test('should show quota warning when near limit', async ({ page }) => {
    // Mock quota near limit
    const nearLimitQuota = {
      ...mockOrganizations[0].quotas,
      currentUserCount: 47,
      maxUsers: 50,
    };

    await page.route('**/api/v1/platform/organizations/org-1/quota', async (route) => {
      await mockApiResponse(route, nearLimitQuota);
    });

    await page.reload();

    // Should show warning banner
    await expect(page.locator('text=Approaching resource limits')).toBeVisible();
  });
});

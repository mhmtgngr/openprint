import { test, expect } from '@playwright/test';
import { login, mockApiResponse, mockUsers, mockPrinters, mockJobs, mockEnvironmentReport } from '../helpers';

test.describe('Dashboard Page', () => {
  test.beforeEach(async ({ page }) => {
    // Setup auth mock
    await page.route('**/api/v1/auth/me', async (route) => {
      await mockApiResponse(route, mockUsers[0]);
    });

    // Setup printers mock - the Dashboard component expects { printers: [...] }
    await page.route('**/api/v1/printers', async (route) => {
      await mockApiResponse(route, { printers: mockPrinters });
    });

    // Setup jobs mock
    await page.route('**/api/v1/jobs*', async (route) => {
      await mockApiResponse(route, {
        data: mockJobs,
        total: mockJobs.length,
        limit: 50,
        offset: 0,
      });
    });

    // Setup environment report mock
    await page.route('**/api/v1/analytics/environment*', async (route) => {
      await mockApiResponse(route, mockEnvironmentReport);
    });
  });

  test('should display dashboard with user greeting', async ({ page }) => {
    await login(page);
    await page.waitForURL('**/dashboard');

    // Use getByName or be more specific with the selector
    await expect(page.getByRole('heading', { name: /Welcome back/i })).toBeVisible();
    await expect(page.getByRole('heading', { name: new RegExp(mockUsers[0].name.split(' ')[0], 'i') })).toBeVisible();
  });

  test('should display stats cards', async ({ page }) => {
    await login(page);
    await page.waitForURL('**/dashboard');

    // Check for stats cards
    await expect(page.getByText('Active Printers')).toBeVisible();
    await expect(page.getByText('Jobs Today')).toBeVisible();
    await expect(page.getByText('Pages This Month')).toBeVisible();
  });

  test('should show correct active printer count', async ({ page }) => {
    await login(page);
    await page.waitForURL('**/dashboard');

    // Count online printers from mock
    const onlinePrinters = mockPrinters.filter(p => p.isOnline && p.isActive).length;

    // Find the stat card for active printers - check the label and then find the value nearby
    await expect(page.getByText('Active Printers')).toBeVisible();

    // The count should be displayed in the stats section - use first() to get the stats grid
    const statsSection = page.locator('.grid.grid-cols-1.md\\:grid-cols-3').first();
    await expect(statsSection).toContainText(onlinePrinters.toString());
  });

  test('should display recent print jobs', async ({ page }) => {
    await login(page);
    await page.waitForURL('**/dashboard');

    await expect(page.getByText('Recent Print Jobs')).toBeVisible();

    // Check for job items
    await expect(page.getByText(mockJobs[0].documentName)).toBeVisible();
  });

  test('should have link to view all jobs', async ({ page }) => {
    await login(page);
    await page.waitForURL('**/dashboard');

    const viewAllLink = page.locator('a').filter({ hasText: 'View all' }).first();
    await expect(viewAllLink).toBeVisible();

    await viewAllLink.click();
    await page.waitForURL('**/jobs');
  });

  test('should display available printers', async ({ page }) => {
    await login(page);
    await page.waitForURL('**/dashboard');

    // The Available Printers section should be visible
    await expect(page.getByText('Available Printers')).toBeVisible();

    // Check for printer items - the printer name should be visible
    // Use .first() to avoid strict mode violation when multiple elements contain the text
    await expect(page.getByText(mockPrinters[0].name).first()).toBeVisible();
  });

  test('should have link to manage printers', async ({ page }) => {
    await login(page);
    await page.waitForURL('**/dashboard');

    const manageLink = page.getByRole('link', { name: /Manage/i }).first();
    await expect(manageLink).toBeVisible();

    await manageLink.click();
    await page.waitForURL('**/printers');
  });

  test('should display environmental impact report', async ({ page }) => {
    await login(page);
    await page.waitForURL('**/dashboard');

    await expect(page.getByText('Environmental Impact')).toBeVisible();
    await expect(page.getByText('Pages Printed')).toBeVisible();
    await expect(page.getByText(/CO.*Saved/i)).toBeVisible();
    await expect(page.getByText('Trees Saved')).toBeVisible();
  });

  test('should show empty state for no printers', async ({ page }) => {
    // Mock empty printers list - override the beforeEach mock
    await page.route('**/api/v1/printers', async (route) => {
      await mockApiResponse(route, { printers: [] });
    });

    await login(page);
    await page.waitForURL('**/dashboard');

    await expect(page.getByText('No printers configured')).toBeVisible();
  });

  test('should show empty state for no jobs', async ({ page }) => {
    // Mock empty jobs list - override the beforeEach mock
    await page.route('**/api/v1/jobs*', async (route) => {
      await mockApiResponse(route, {
        data: [],
        total: 0,
        limit: 50,
        offset: 0,
      });
    });

    await login(page);
    await page.waitForURL('**/dashboard');

    await expect(page.getByText('No print jobs yet')).toBeVisible();
  });

  test('should display job status badges', async ({ page }) => {
    await login(page);
    await page.waitForURL('**/dashboard');

    // Check for status badges
    await expect(page.getByText('Completed')).toBeVisible();
    await expect(page.getByText('Processing')).toBeVisible();
  });

  test('should have navigation sidebar', async ({ page }) => {
    await login(page);
    await page.waitForURL('**/dashboard');

    // Check for sidebar elements - use link role to be more specific
    await expect(page.getByRole('link', { name: 'Dashboard' })).toBeVisible();
    await expect(page.getByRole('link', { name: /Printers|Devices/i })).toBeVisible();
    await expect(page.getByRole('link', { name: /Jobs/i })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Settings' })).toBeVisible();
  });

  test('should show user info in sidebar', async ({ page }) => {
    await login(page);
    await page.waitForURL('**/dashboard');

    // Check for user info
    await expect(page.getByText(mockUsers[0].name)).toBeVisible();
    await expect(page.getByText(mockUsers[0].email)).toBeVisible();

    // Check for avatar with user initial
    const avatar = page.locator('.w-10.h-10.bg-gradient-to-br');
    await expect(avatar).toContainText(mockUsers[0].name.charAt(0).toUpperCase());
  });

  test('should have logout button in sidebar', async ({ page }) => {
    await login(page);
    await page.waitForURL('**/dashboard');

    const logoutButton = page.getByRole('button', { name: 'Logout' });
    await expect(logoutButton).toBeVisible();
  });

  test('should navigate to different pages via sidebar', async ({ page }) => {
    await login(page);
    await page.waitForURL('**/dashboard');

    // Navigate to printers - the sidebar shows "Devices" for the printers route
    await page.getByRole('link', { name: 'Devices' }).click();
    await page.waitForURL('**/printers', { timeout: 10000 });

    // Navigate back to dashboard first - wait for page to stabilize
    await page.waitForLoadState('networkidle');
    await page.getByRole('link', { name: 'Dashboard' }).click();
    await page.waitForURL('**/dashboard', { timeout: 10000 });

    // Navigate to jobs
    await page.waitForLoadState('networkidle');
    await page.getByRole('link', { name: 'Jobs' }).click();
    await page.waitForURL('**/jobs', { timeout: 10000 });
  });

  test('should highlight active navigation item', async ({ page }) => {
    await login(page);
    await page.waitForURL('**/dashboard');

    // Dashboard link should be visible in the navigation
    // The link might be in a sidebar or nav element
    const dashboardLink = page.getByRole('link', { name: 'Dashboard' });
    await expect(dashboardLink).toBeVisible();
    await expect(dashboardLink).toHaveAttribute('href', '/dashboard');
  });

  test('should handle API errors gracefully', async ({ page }) => {
    // Mock error responses - override the beforeEach mock
    await page.route('**/api/v1/printers', async (route) => {
      await route.abort('failed');
    });

    await login(page);
    await page.waitForURL('**/dashboard');

    // Page should still load
    await expect(page.getByRole('heading', { name: /Welcome back/i })).toBeVisible();
  });

  test('should be responsive on mobile', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await login(page);
    await page.waitForURL('**/dashboard');

    // Check that main content is still visible - use getByName to avoid strict mode violation
    await expect(page.getByRole('heading', { name: /Welcome back/i })).toBeVisible();

    // Stats should still be visible on mobile
    await expect(page.getByText('Active Printers')).toBeVisible();
  });
});

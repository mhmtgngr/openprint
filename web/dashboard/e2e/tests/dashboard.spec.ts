import { test, expect } from '@playwright/test';
import { login, mockApiResponse, mockUsers, mockPrinters, mockJobs, mockEnvironmentReport } from '../helpers';

test.describe('Dashboard Page', () => {
  test.beforeEach(async ({ page }) => {
    // Setup auth mock
    await page.route('**/api/v1/auth/me', async (route) => {
      await mockApiResponse(route, mockUsers[0]);
    });

    // Setup printers mock
    await page.route('**/api/v1/printers', async (route) => {
      await mockApiResponse(route, mockPrinters);
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

    await expect(page.locator('h1')).toContainText('Welcome back');
    await expect(page.locator('h1')).toContainText(mockUsers[0].name.split(' ')[0]);
  });

  test('should display stats cards', async ({ page }) => {
    await login(page);
    await page.waitForURL('**/dashboard');

    // Check for stats cards
    await expect(page.locator('text=Active Printers')).toBeVisible();
    await expect(page.locator('text=Jobs Today')).toBeVisible();
    await expect(page.locator('text=Pages This Month')).toBeVisible();
  });

  test('should show correct active printer count', async ({ page }) => {
    await login(page);
    await page.waitForURL('**/dashboard');

    // Count online printers from mock
    const onlinePrinters = mockPrinters.filter(p => p.isOnline && p.isActive).length;

    // Find the stat card for active printers
    const printerStat = page.locator('.bg-white.dark\\:\\bg-gray-800').filter({
      hasText: 'Active Printers'
    });

    await expect(printerStat).toContainText(onlinePrinters.toString());
  });

  test('should display recent print jobs', async ({ page }) => {
    await login(page);
    await page.waitForURL('**/dashboard');

    await expect(page.locator('text=Recent Print Jobs')).toBeVisible();

    // Check for job items
    await expect(page.locator('text=' + mockJobs[0].documentName)).toBeVisible();
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

    await expect(page.locator('text=Available Printers')).toBeVisible();

    // Check for printer items
    await expect(page.locator('text=' + mockPrinters[0].name)).toBeVisible();
  });

  test('should have link to manage printers', async ({ page }) => {
    await login(page);
    await page.waitForURL('**/dashboard');

    const manageLink = page.locator('a').filter({ hasText: 'Manage' });
    await expect(manageLink).toBeVisible();

    await manageLink.click();
    await page.waitForURL('**/printers');
  });

  test('should display environmental impact report', async ({ page }) => {
    await login(page);
    await page.waitForURL('**/dashboard');

    await expect(page.locator('text=Environmental Impact')).toBeVisible();
    await expect(page.locator('text=Pages Printed')).toBeVisible();
    await expect(page.locator('text=CO₂ Saved')).toBeVisible();
    await expect(page.locator('text=Trees Saved')).toBeVisible();
  });

  test('should show empty state for no printers', async ({ page }) => {
    // Mock empty printers list
    await page.route('**/api/v1/printers', async (route) => {
      await mockApiResponse(route, []);
    });

    await login(page);
    await page.waitForURL('**/dashboard');

    await expect(page.locator('text=No printers configured')).toBeVisible();
  });

  test('should show empty state for no jobs', async ({ page }) => {
    // Mock empty jobs list
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

    await expect(page.locator('text=No print jobs yet')).toBeVisible();
  });

  test('should display job status badges', async ({ page }) => {
    await login(page);
    await page.waitForURL('**/dashboard');

    // Check for status badges
    await expect(page.locator('text=Completed')).toBeVisible();
    await expect(page.locator('text=Processing')).toBeVisible();
  });

  test('should have navigation sidebar', async ({ page }) => {
    await login(page);
    await page.waitForURL('**/dashboard');

    // Check for sidebar elements
    await expect(page.locator('text=Dashboard')).toBeVisible();
    await expect(page.locator('text=Printers')).toBeVisible();
    await expect(page.locator('text=Print Jobs')).toBeVisible();
    await expect(page.locator('text=Settings')).toBeVisible();
  });

  test('should show user info in sidebar', async ({ page }) => {
    await login(page);
    await page.waitForURL('**/dashboard');

    // Check for user info
    await expect(page.locator('text=' + mockUsers[0].name)).toBeVisible();
    await expect(page.locator('text=' + mockUsers[0].email)).toBeVisible();

    // Check for avatar with user initial
    const avatar = page.locator('.w-10.h-10.bg-gradient-to-br');
    await expect(avatar).toContainText(mockUsers[0].name.charAt(0).toUpperCase());
  });

  test('should have logout button in sidebar', async ({ page }) => {
    await login(page);
    await page.waitForURL('**/dashboard');

    const logoutButton = page.locator('button', { hasText: 'Logout' });
    await expect(logoutButton).toBeVisible();
  });

  test('should navigate to different pages via sidebar', async ({ page }) => {
    await login(page);
    await page.waitForURL('**/dashboard');

    // Navigate to printers
    await page.click('a[href="/printers"]');
    await page.waitForURL('**/printers');

    // Navigate to jobs
    await page.click('a[href="/jobs"]');
    await page.waitForURL('**/jobs');
  });

  test('should highlight active navigation item', async ({ page }) => {
    await login(page);
    await page.waitForURL('**/dashboard');

    // Dashboard link should be active
    const dashboardLink = page.locator('a[href="/dashboard"]');
    await expect(dashboardLink).toHaveClass(/bg-blue-100/);
  });

  test('should handle API errors gracefully', async ({ page }) => {
    // Mock error responses
    await page.route('**/api/v1/printers', async (route) => {
      await route.abort('failed');
    });

    await login(page);
    await page.waitForURL('**/dashboard');

    // Page should still load, showing empty states or zero counts
    await expect(page.locator('h1')).toContainText('Welcome back');
  });

  test('should be responsive on mobile', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });
    await login(page);
    await page.waitForURL('**/dashboard');

    // Check that main content is still visible
    await expect(page.locator('h1')).toBeVisible();

    // Stats should stack vertically on mobile
    const statsContainer = page.locator('.grid.grid-cols-1.md\\:grid-cols-3');
    await expect(statsContainer).toBeVisible();
  });
});

import { test, expect } from '@playwright/test';
import { login, mockApiResponse, mockUsers, mockUsageStats, mockEnvironmentReport, mockAuditLogs, mockPrinters, mockJobs } from '../helpers';

const adminUser = mockUsers[1];

test.describe('Analytics Page (Admin)', () => {
  test.beforeEach(async ({ page }) => {
    // Setup auth mock with admin user
    await page.route('**/api/v1/auth/me', async (route) => {
      await mockApiResponse(route, adminUser);
    });

    // Setup usage stats mock
    await page.route('**/api/v1/analytics/usage*', async (route) => {
      await mockApiResponse(route, mockUsageStats);
    });

    // Setup environment report mock
    await page.route('**/api/v1/analytics/environment*', async (route) => {
      await mockApiResponse(route, mockEnvironmentReport);
    });

    // Setup audit logs mock
    await page.route('**/api/v1/analytics/audit-logs*', async (route) => {
      await mockApiResponse(route, {
        data: mockAuditLogs,
        total: mockAuditLogs.length,
        limit: 20,
        offset: 0,
      });
    });

    // Setup common API mocks needed for login flow
    await page.route('**/api/v1/printers', async (route) => {
      await mockApiResponse(route, { printers: mockPrinters });
    });

    await page.route('**/api/v1/jobs*', async (route) => {
      await mockApiResponse(route, { data: mockJobs, total: mockJobs.length, limit: 50, offset: 0 });
    });

    await login(page, {
      email: adminUser.email,
      password: 'AdminPassword123!',
      name: adminUser.name,
    });
    await page.goto('/analytics');
  });

  test('should display analytics page header', async ({ page }) => {
    await expect(page.locator('h1')).toContainText('Analytics');
    await expect(page.locator('text=Track your printing usage and environmental impact')).toBeVisible();
  });

  test('should display period selector buttons', async ({ page }) => {
    await expect(page.locator('button:has-text("7 Days")')).toBeVisible();
    await expect(page.locator('button:has-text("30 Days")')).toBeVisible();
    await expect(page.locator('button:has-text("90 Days")')).toBeVisible();
    await expect(page.locator('button:has-text("12 Months")')).toBeVisible();
  });

  test('should change period selector', async ({ page }) => {
    // Initially 30 Days should be selected
    let periodButton = page.locator('button:has-text("30 Days")');
    await expect(periodButton).toHaveClass(/bg-blue-600/);

    // Click 7 Days
    await page.click('button:has-text("7 Days")');

    // 7 Days should now be selected
    periodButton = page.locator('button:has-text("7 Days")');
    await expect(periodButton).toHaveClass(/bg-blue-600/);
  });

  test('should display metric cards', async ({ page }) => {
    await expect(page.locator('text=Total Jobs')).toBeVisible();
    await expect(page.locator('text=Pages Printed')).toBeVisible();
    await expect(page.locator('text=Success Rate')).toBeVisible();
    await expect(page.locator('text=Estimated Cost')).toBeVisible();
  });

  test('should show metric values', async ({ page }) => {
    const totalJobs = mockUsageStats.reduce((sum, s) => sum + s.jobsCount, 0);
    const totalPages = mockUsageStats.reduce((sum, s) => sum + s.pagesPrinted, 0);

    await expect(page.locator(`text=${totalJobs.toLocaleString()}`)).toBeVisible();
    await expect(page.locator(`text=${totalPages.toLocaleString()}`)).toBeVisible();
  });

  test('should show success rate percentage', async ({ page }) => {
    const totalJobs = mockUsageStats.reduce((sum, s) => sum + s.jobsCount, 0);
    const successRate =
      totalJobs > 0
        ? ((mockUsageStats.reduce((sum, s) => sum + s.jobsCompleted, 0) / totalJobs) * 100).toFixed(1)
        : '0.0';

    await expect(page.locator(`text=${successRate}%`)).toBeVisible();
  });

  test('should display environmental impact report', async ({ page }) => {
    await expect(page.locator('text=Environmental Impact')).toBeVisible();
    await expect(page.locator('text=Pages Printed')).toBeVisible();
    await expect(page.locator('text=CO₂ Saved')).toBeVisible();
    await expect(page.locator('text=Trees Saved')).toBeVisible();
  });

  test('should show environmental values', async ({ page }) => {
    await expect(page.locator(`text=${mockEnvironmentReport.pagesPrinted}`)).toBeVisible();
    await expect(page.locator(`text=${mockEnvironmentReport.co2Grams}g`)).toBeVisible();
    await expect(page.locator(`text=${mockEnvironmentReport.treesSaved}`)).toBeVisible();
  });

  test('should display print volume chart', async ({ page }) => {
    await expect(page.locator('text=Print Volume Over Time')).toBeVisible();
    // Chart would be rendered by Recharts
  });

  test('should display job status distribution chart', async ({ page }) => {
    await expect(page.locator('text=Job Status Distribution')).toBeVisible();
    // Chart would be rendered by Recharts
  });

  test('should display CO2 trend chart', async ({ page }) => {
    await expect(page.locator('text=CO₂ Emissions Trend')).toBeVisible();
    // Chart would be rendered by Recharts
  });

  test('should display audit logs section', async ({ page }) => {
    await expect(page.locator('text=Recent Activity')).toBeVisible();
  });

  test('should show audit log entries', async ({ page }) => {
    // Find audit log section
    await expect(page.locator('text=Recent Activity')).toBeVisible();

    // Check for log entries
    for (const log of mockAuditLogs.slice(0, 10)) {
      await expect(page.locator(`text=${log.action}`)).toBeVisible();
    }
  });

  test('should navigate to dashboard via sidebar', async ({ page }) => {
    await page.click('a[href="/dashboard"]');
    await page.waitForURL('**/dashboard');

    await expect(page.locator('h1')).toContainText('Welcome back');
  });

  test('should highlight Analytics in navigation', async ({ page }) => {
    const analyticsLink = page.locator('a[href="/analytics"]');
    await expect(analyticsLink).toHaveClass(/bg-blue-100/);
  });

  test('should be responsive on mobile', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });

    // Check that main content is still visible
    await expect(page.locator('h1')).toBeVisible();

    // Metric cards should stack vertically
    const metricGrid = page.locator('.grid.grid-cols-1.md\\:grid-cols-4');
    await expect(metricGrid).toBeVisible();
  });

  test('should display cost in dollars', async ({ page }) => {
    const totalCost = mockUsageStats.reduce((sum, s) => sum + s.estimatedCost, 0);
    await expect(page.locator(`text=$${totalCost.toFixed(2)}`)).toBeVisible();
  });
});

test.describe('Analytics Access Control', () => {
  test('should redirect non-admin users to dashboard', async ({ page }) => {
    // Setup auth as regular user
    await page.route('**/api/v1/auth/me', async (route) => {
      await mockApiResponse(route, mockUsers[0]);
    });

    await login(page);
    await page.goto('/analytics');

    // Should redirect to dashboard
    await page.waitForURL('**/dashboard');
    await expect(page.locator('h1')).toContainText('Welcome back');
  });

  test('should allow admin access', async ({ page }) => {
    // Setup auth as admin
    await page.route('**/api/v1/auth/me', async (route) => {
      await mockApiResponse(route, mockUsers[1]);
    });

    await page.route('**/api/v1/analytics/**', async (route) => {
      await mockApiResponse(route, mockUsageStats);
    });

    await login(page, {
      email: mockUsers[1].email,
      password: 'AdminPassword123!',
      name: mockUsers[1].name,
    });
    await page.goto('/analytics');

    await expect(page.locator('h1')).toContainText('Analytics');
  });
});

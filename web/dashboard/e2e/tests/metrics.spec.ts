import { test, expect } from '@playwright/test';
import { login, mockApiResponse, mockUsers } from '../helpers';

const adminUser = mockUsers[1];

test.describe('Metrics Dashboard (Admin)', () => {
  test.beforeEach(async ({ page }) => {
    // Setup auth mock with admin user
    await page.route('**/api/v1/auth/me', async (route) => {
      await mockApiResponse(route, adminUser);
    });

    // Setup service health mock
    await page.route('**/monitoring/health/services', async (route) => {
      await mockApiResponse(route, [
        {
          serviceName: 'auth-service',
          instance: 'localhost:8001',
          status: 'healthy',
          version: '1.0.0',
          uptime: 3600000,
          lastCheck: new Date().toISOString(),
          metrics: {
            cpuPercent: 25.5,
            memoryPercent: 45.2,
            diskPercent: 30.0,
            requestRate: 125.5,
            errorRate: 0.01,
            latency: { p50: 15, p95: 45, p99: 120 },
          },
          dependencies: [
            { name: 'postgres', type: 'database', status: 'healthy', latency: 5 },
            { name: 'redis', type: 'redis', status: 'healthy', latency: 1 },
          ],
        },
        {
          serviceName: 'job-service',
          instance: 'localhost:8003',
          status: 'healthy',
          version: '1.0.0',
          uptime: 3600000,
          lastCheck: new Date().toISOString(),
          metrics: {
            cpuPercent: 35.8,
            memoryPercent: 55.3,
            diskPercent: 40.0,
            requestRate: 85.2,
            errorRate: 0.05,
            latency: { p50: 20, p95: 65, p99: 150 },
          },
          dependencies: [],
        },
        {
          serviceName: 'registry-service',
          instance: 'localhost:8002',
          status: 'degraded',
          version: '1.0.0',
          uptime: 3600000,
          lastCheck: new Date().toISOString(),
          metrics: {
            cpuPercent: 65.2,
            memoryPercent: 78.5,
            diskPercent: 55.0,
            requestRate: 45.8,
            errorRate: 2.5,
            latency: { p50: 45, p95: 120, p99: 250 },
          },
          dependencies: [],
        },
      ]);
    });

    // Setup alert summary mock
    await page.route('**/monitoring/alerts/summary', async (route) => {
      await mockApiResponse(route, {
        total: 5,
        firing: 2,
        pending: 1,
        resolved: 2,
        bySeverity: { critical: 1, warning: 1, info: 0, none: 3 },
        byService: { 'registry-service': 2, 'job-service': 0, 'auth-service': 0 },
      });
    });

    // Setup Prometheus metrics mocks
    await page.route('**/api/v1/query*', async (route) => {
      const url = route.request().url();
      if (url.includes("rate(http_requests_total)") || url.includes("rate(http_request_duration")") || url.includes("histogram_quantile")) {
        await mockApiResponse(route, {
          status: 'success',
          data: {
            resultType: 'matrix',
            result: [
              {
                metric: { service: 'auth-service', instance: 'localhost:8001' },
                values: [
                  [1710000000, '100'],
                  [1710000015, '105'],
                  [1710000030, '98'],
                  [1710000045, '102'],
                ],
              },
              {
                metric: { service: 'job-service', instance: 'localhost:8003' },
                values: [
                  [1710000000, '80'],
                  [1710000015, '85'],
                  [1710000030, '82'],
                  [1710000045, '88'],
                ],
              },
            ],
          },
        });
      } else {
        await route.continue();
      }
    });

    await login(page, {
      email: adminUser.email,
      password: 'AdminPassword123!',
      name: adminUser.name,
    });
    await page.goto('/metrics');
  });

  test('should display metrics dashboard page header', async ({ page }) => {
    await expect(page.locator('h1')).toContainText('Metrics Dashboard');
    await expect(page.locator('text=Real-time performance metrics')).toBeVisible();
  });

  test('should display time range selector buttons', async ({ page }) => {
    await expect(page.locator('button:has-text("5 min")')).toBeVisible();
    await expect(page.locator('button:has-text("1 hour")')).toBeVisible();
    await expect(page.locator('button:has-text("6 hours")')).toBeVisible();
    await expect(page.locator('button:has-text("24 hours")')).toBeVisible();
  });

  test('should change time range selector', async ({ page }) => {
    // Initially 1 hour should be selected
    let timeButton = page.locator('button:has-text("1 hour")');
    await expect(timeButton).toHaveClass(/bg-blue-600/);

    // Click 6 hours
    await page.click('button:has-text("6 hours")');

    // 6 hours should now be selected
    timeButton = page.locator('button:has-text("6 hours")');
    await expect(timeButton).toHaveClass(/bg-blue-600/);
  });

  test('should display summary metric cards', async ({ page }) => {
    await expect(page.locator('text=Request Rate')).toBeVisible();
    await expect(page.locator('text=Error Rate')).toBeVisible();
    await expect(page.locator('text=P95 Latency')).toBeVisible();
    await expect(page.locator('text=Alerts')).toBeVisible();
  });

  test('should display service selector', async ({ page }) => {
    await expect(page.locator('text=Services:')).toBeVisible();
    await expect(page.locator('button:has-text("All Services")')).toBeVisible();
    await expect(page.locator('button:has-text("auth-service")')).toBeVisible();
    await expect(page.locator('button:has-text("job-service")')).toBeVisible();
  });

  test('should filter by service', async ({ page }) => {
    await page.click('button:has-text("auth-service")');

    const authButton = page.locator('button:has-text("auth-service")');
    await expect(authButton).toHaveClass(/bg-blue-600/);
  });

  test('should display service health cards', async ({ page }) => {
    await expect(page.locator('text=Service Health')).toBeVisible();
    await expect(page.locator('text=auth-service')).toBeVisible();
    await expect(page.locator('text=job-service')).toBeVisible();
    await expect(page.locator('text=registry-service')).toBeVisible();
  });

  test('should display service status badges', async ({ page }) => {
    const healthyBadge = page.locator('span:has-text("healthy")').first();
    await expect(healthyBadge).toBeVisible();

    const degradedBadge = page.locator('span:has-text("degraded")').first();
    await expect(degradedBadge).toBeVisible();
  });

  test('should display resource usage metrics', async ({ page }) => {
    await expect(page.locator('text=CPU:')).toBeVisible();
    await expect(page.locator('text=Memory:')).toBeVisible();
    await expect(page.locator('text=Request Rate:')).toBeVisible();
  });

  test('should toggle auto-refresh', async ({ page }) => {
    const refreshButton = page.locator('button[title*="Auto-refresh"]');
    await expect(refreshButton).toBeVisible();

    // Click to enable auto-refresh
    await refreshButton.click();
    await expect(refreshButton).toHaveClass(/bg-green-100/);
  });

  test('should navigate to monitoring via alerts card', async ({ page }) => {
    await page.locator('.bg-white.dark\\:\\bg-gray-800').filter({ hasText: 'Alerts' }).click();

    await page.waitForURL('**/monitoring');
    await expect(page.locator('h1')).toContainText('Monitoring');
  });

  test('should be responsive on mobile', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });

    await expect(page.locator('h1')).toBeVisible();

    const metricGrid = page.locator('.grid.grid-cols-1.md\\:grid-cols-4');
    await expect(metricGrid).toBeVisible();
  });
});

test.describe('Metrics Access Control', () => {
  test('should redirect non-admin users to dashboard', async ({ page }) => {
    const regularUser = mockUsers[0];

    await page.route('**/api/v1/auth/me', async (route) => {
      await mockApiResponse(route, regularUser);
    });

    await login(page);
    await page.goto('/metrics');

    await page.waitForURL('**/dashboard');
    await expect(page.locator('h1')).toContainText('Welcome back');
  });

  test('should allow admin access', async ({ page }) => {
    await page.route('**/api/v1/auth/me', async (route) => {
      await mockApiResponse(route, adminUser);
    });

    await page.route('**/monitoring/health/services', async (route) => {
      await mockApiResponse(route, []);
    });

    await page.route('**/monitoring/alerts/summary', async (route) => {
      await mockApiResponse(route, { total: 0, firing: 0, pending: 0, resolved: 0 });
    });

    await login(page, {
      email: adminUser.email,
      password: 'AdminPassword123!',
      name: adminUser.name,
    });
    await page.goto('/metrics');

    await expect(page.locator('h1')).toContainText('Metrics Dashboard');
  });
});

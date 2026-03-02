import { test, expect } from '@playwright/test';
import { login, mockApiResponse, mockUsers } from '../helpers';

const adminUser = mockUsers[1];

test.describe('Monitoring Page (Admin)', () => {
  test.beforeEach(async ({ page }) => {
    // Setup auth mock with admin user
    await page.route('**/api/v1/auth/me', async (route) => {
      await mockApiResponse(route, adminUser);
    });

    // Setup alerts mock
    await page.route('**/monitoring/alerts', async (route) => {
      await mockApiResponse(route, [
        {
          id: 'alert1',
          name: 'HighErrorRate',
          state: 'firing',
          severity: 'critical',
          service: 'registry-service',
          message: 'Error rate above 5% for 5 minutes',
          startsAt: new Date(Date.now() - 300000).toISOString(),
          endsAt: new Date(Date.now() + 300000).toISOString(),
          labels: {
            alertname: 'HighErrorRate',
            severity: 'critical',
            service: 'registry-service',
          },
          annotations: {},
          generatorURL: 'http://localhost:9090/graph',
        },
        {
          id: 'alert2',
          name: 'HighLatency',
          state: 'firing',
          severity: 'warning',
          service: 'job-service',
          message: 'P95 latency above 100ms',
          startsAt: new Date(Date.now() - 600000).toISOString(),
          labels: {
            alertname: 'HighLatency',
            severity: 'warning',
            service: 'job-service',
          },
          annotations: {},
          generatorURL: 'http://localhost:9090/graph',
        },
        {
          id: 'alert3',
          name: 'ServiceDown',
          state: 'resolved',
          severity: 'critical',
          service: 'auth-service',
          message: 'Service is back online',
          startsAt: new Date(Date.now() - 3600000).toISOString(),
          endsAt: new Date(Date.now() - 1800000).toISOString(),
          labels: {
            alertname: 'ServiceDown',
            severity: 'critical',
            service: 'auth-service',
          },
          annotations: {},
          generatorURL: 'http://localhost:9090/graph',
        },
      ]);
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
          dependencies: [],
        },
        {
          serviceName: 'registry-service',
          instance: 'localhost:8002',
          status: 'unhealthy',
          version: '1.0.0',
          uptime: 3600000,
          lastCheck: new Date().toISOString(),
          metrics: {
            cpuPercent: 85.2,
            memoryPercent: 92.5,
            diskPercent: 75.0,
            requestRate: 25.8,
            errorRate: 8.5,
            latency: { p50: 150, p95: 350, p99: 500 },
          },
          dependencies: [],
        },
      ]);
    });

    // Setup silences mock
    await page.route('**/monitoring/silences', async (route) => {
      await mockApiResponse(route, [
        {
          id: 'silence1',
          matchers: [
            { name: 'alertname', value: 'HighLatency', isRegex: false },
          ],
          startsAt: new Date(Date.now() - 3600000).toISOString(),
          endsAt: new Date(Date.now() + 3600000).toISOString(),
          createdBy: 'admin@example.com',
          comment: 'Silenced during maintenance',
        },
      ]);
    });

    await login(page, {
      email: adminUser.email,
      password: 'AdminPassword123!',
      name: adminUser.name,
    });
    await page.goto('/monitoring');
  });

  test('should display monitoring page header', async ({ page }) => {
    await expect(page.locator('h1')).toContainText('Monitoring');
    await expect(page.locator('text=Real-time alerts and service health')).toBeVisible();
  });

  test('should display summary cards', async ({ page }) => {
    await expect(page.locator('text=Total Alerts')).toBeVisible();
    await expect(page.locator('text=Firing')).toBeVisible();
    await expect(page.locator('text=Services')).toBeVisible();
    await expect(page.locator('text=Health Issues')).toBeVisible();
  });

  test('should display tabs', async ({ page }) => {
    await expect(page.locator('button:has-text("Alerts")')).toBeVisible();
    await expect(page.locator('button:has-text("Services")')).toBeVisible();
    await expect(page.locator('button:has-text("Silences")')).toBeVisible();
  });

  test('should display alerts tab by default', async ({ page }) => {
    const alertsTab = page.locator('button:has-text("Alerts")');
    await expect(alertsTab).toHaveClass(/text-blue-600/);
  });

  test('should display alert list', async ({ page }) => {
    await expect(page.locator('text=HighErrorRate')).toBeVisible();
    await expect(page.locator('text=HighLatency')).toBeVisible();
    await expect(page.locator('text=ServiceDown')).toBeVisible();
  });

  test('should display alert severity badges', async ({ page }) => {
    await expect(page.locator('span:has-text("firing")').first()).toBeVisible();
    await expect(page.locator('span:has-text("resolved")')).first()).toBeVisible();
  });

  test('should filter alerts by severity', async ({ page }) => {
    await page.click('button:has-text("Critical")');

    // Should only show critical alerts
    await expect(page.locator('text=HighErrorRate')).toBeVisible();
  });

  test('should filter alerts by state', async ({ page }) => {
    await page.click('text=State:');
    await page.click('button:has-text("Resolved")');

    // Should only show resolved alerts
    await expect(page.locator('text=ServiceDown')).toBeVisible();
  });

  test('should switch to services tab', async ({ page }) => {
    await page.click('button:has-text("Services")');

    const servicesTab = page.locator('button:has-text("Services")');
    await expect(servicesTab).toHaveClass(/text-blue-600/);

    await expect(page.locator('text=auth-service')).toBeVisible();
    await expect(page.locator('text=registry-service')).toBeVisible();
  });

  test('should display service health cards', async ({ page }) => {
    await page.click('button:has-text("Services")');

    await expect(page.locator('span:has-text("healthy")')).toBeVisible();
    await expect(page.locator('span:has-text("unhealthy")')).toBeVisible();
  });

  test('should click on service card to view details', async ({ page }) => {
    await page.click('button:has-text("Services")');

    await page.locator('.bg-white.dark\\:\\bg-gray-800').filter({ hasText: 'auth-service' }).click();

    // Should open modal
    await expect(page.locator('text=auth-service')).toBeVisible();
    await expect(page.locator('text=CPU')).toBeVisible();
    await expect(page.locator('text=Memory')).toBeVisible();
  });

  test('should switch to silences tab', async ({ page }) => {
    await page.click('button:has-text("Silences")');

    const silencesTab = page.locator('button:has-text("Silences")');
    await expect(silencesTab).toHaveClass(/text-blue-600/);

    await expect(page.locator('text=Silenced during maintenance')).toBeVisible();
  });

  test('should toggle auto-refresh', async ({ page }) => {
    const refreshButton = page.locator('button[title*="Auto-refresh"]');
    await expect(refreshButton).toBeVisible();

    await refreshButton.click();
    await expect(refreshButton).toHaveClass(/bg-green-100/);
  });

  test('should refresh all data', async ({ page }) => {
    await page.click('button:has-text("Refresh")');

    // Should reload data - verify by checking elements still exist
    await expect(page.locator('text=HighErrorRate')).toBeVisible();
  });

  test('should click on alert to view details', async ({ page }) => {
    const alertCard = page.locator('.bg-amber-50, .bg-red-50').filter({ hasText: 'HighErrorRate' }).first();
    await alertCard.click();

    // Should open modal
    await expect(page.locator('text=HighErrorRate')).toBeVisible();
    await expect(page.locator('text=Message')).toBeVisible();
    await expect(page.locator('text=Labels')).toBeVisible();
  });

  test('should close alert detail modal', async ({ page }) => {
    const alertCard = page.locator('.bg-amber-50, .bg-red-50').filter({ hasText: 'HighErrorRate' }).first();
    await alertCard.click();

    // Click close button
    await page.click('button:has-text("Close")');

    // Modal should be closed
    await expect(page.locator('text=Message')).not.toBeVisible();
  });
});

test.describe('Monitoring Access Control', () => {
  test('should redirect non-admin users to dashboard', async ({ page }) => {
    const regularUser = mockUsers[0];

    await page.route('**/api/v1/auth/me', async (route) => {
      await mockApiResponse(route, regularUser);
    });

    await login(page);
    await page.goto('/monitoring');

    await page.waitForURL('**/dashboard');
    await expect(page.locator('h1')).toContainText('Welcome back');
  });
});

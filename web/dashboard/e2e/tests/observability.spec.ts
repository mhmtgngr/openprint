import { test, expect } from '@playwright/test';
import { login, mockApiResponse, mockUsers } from '../helpers';

const adminUser = mockUsers[1];

test.describe('Observability Hub (Admin)', () => {
  test.beforeEach(async ({ page }) => {
    // Setup auth mock with admin user
    await page.route('**/api/v1/auth/me', async (route) => {
      await mockApiResponse(route, adminUser);
    });

    // Setup trace summary mock
    await page.route('**/monitoring/traces/summary*', async (route) => {
      await mockApiResponse(route, {
        totalTraces: 1250,
        errorTraces: 23,
        slowTraces: 45,
        avgDuration: 85000000,
        p95Duration: 250000000,
        p99Duration: 500000000,
        byService: {
          'auth-service': {
            count: 450,
            errorCount: 5,
            avgDuration: 45000000,
            maxDuration: 200000000,
          },
          'job-service': {
            count: 580,
            errorCount: 12,
            avgDuration: 95000000,
            maxDuration: 450000000,
          },
          'registry-service': {
            count: 220,
            errorCount: 6,
            avgDuration: 35000000,
            maxDuration: 150000000,
          },
        },
      });
    });

    // Setup trace search mock
    await page.route('**/monitoring/traces/search*', async (route) => {
      await mockApiResponse(route, [
        {
          traceID: 'trace1abc123',
          rootSpanName: 'POST /api/v1/auth/login',
          rootServiceName: 'auth-service',
          startTime: Date.now() * 1000000 - 30000000000,
          duration: 85000000,
          spanCount: 12,
        },
        {
          traceID: 'trace2def456',
          rootSpanName: 'POST /api/v1/jobs',
          rootServiceName: 'job-service',
          startTime: Date.now() * 1000000 - 60000000000,
          duration: 125000000,
          spanCount: 18,
        },
        {
          traceID: 'trace3ghi789',
          rootSpanName: 'GET /api/v1/printers',
          rootServiceName: 'registry-service',
          startTime: Date.now() * 1000000 - 90000000000,
          duration: 45000000,
          spanCount: 8,
        },
      ]);
    });

    // Setup trace detail mock
    await page.route('**/monitoring/traces/*', async (route) => {
      const url = route.request().url();
      if (url.includes('/search')) {
        await route.continue();
        return;
      }

      await mockApiResponse(route, {
        traceID: 'trace1abc123',
        rootSpanName: 'POST /api/v1/auth/login',
        rootServiceName: 'auth-service',
        duration: 85000000,
        startTime: Date.now() * 1000000 - 30000000000,
        spans: [
          {
            traceID: 'trace1abc123',
            spanID: 'span1',
            operationName: 'POST /api/v1/auth/login',
            processID: 'p1',
            startTime: Date.now() * 1000000 - 30000000000,
            duration: 85000000,
            tags: [
              { key: 'span.kind', value: 'server' },
              { key: 'http.method', value: 'POST' },
              { key: 'http.status_code', value: '200' },
            ],
            logs: [],
          },
          {
            traceID: 'trace1abc123',
            spanID: 'span2',
            operationName: 'validateCredentials',
            processID: 'p1',
            parentSpanID: 'span1',
            startTime: Date.now() * 1000000 - 29900000000,
            duration: 15000000,
            tags: [
              { key: 'span.kind', value: 'internal' },
            ],
            logs: [],
          },
          {
            traceID: 'trace1abc123',
            spanID: 'span3',
            operationName: 'db.query',
            processID: 'p1',
            parentSpanID: 'span2',
            startTime: Date.now() * 1000000 - 29800000000,
            duration: 8000000,
            tags: [
              { key: 'span.kind', value: 'client' },
              { key: 'db.type', value: 'postgres' },
            ],
            logs: [],
          },
        ],
        processes: {
          p1: {
            serviceName: 'auth-service',
            tags: [
              { key: 'hostname', value: 'auth-service-1' },
            ],
          },
        },
      });
    });

    // Setup services list mock
    await page.route('**/monitoring/traces/services', async (route) => {
      await mockApiResponse(route, ['auth-service', 'job-service', 'registry-service', 'storage-service', 'notification-service']);
    });

    await login(page, {
      email: adminUser.email,
      password: 'AdminPassword123!',
      name: adminUser.name,
    });
    await page.goto('/observability');
  });

  test('should display observability hub page header', async ({ page }) => {
    await expect(page.locator('h1')).toContainText('Observability Hub');
    await expect(page.locator('text=Distributed tracing and performance analysis')).toBeVisible();
  });

  test('should display summary cards', async ({ page }) => {
    await expect(page.locator('text=Total Traces')).toBeVisible();
    await expect(page.locator('text=Error Traces')).toBeVisible();
    await expect(page.locator('text=Avg Duration')).toBeVisible();
    await expect(page.locator('text=P99 Duration')).toBeVisible();
  });

  test('should display time range selector', async ({ page }) => {
    await expect(page.locator('text=Time Range:')).toBeVisible();
    await expect(page.locator('button:has-text("1 hour")')).toBeVisible();
    await expect(page.locator('button:has-text("6 hours")')).toBeVisible();
    await expect(page.locator('button:has-text("24 hours")')).toBeVisible();
  });

  test('should display service selector', async ({ page }) => {
    await expect(page.locator('text=Service:')).toBeVisible();
    await expect(page.locator('select')).toBeVisible();
  });

  test('should display tabs', async ({ page }) => {
    await expect(page.locator('button:has-text("Search Traces")')).toBeVisible();
    await expect(page.locator('button:has-text("View Trace")')).toBeVisible();
    await expect(page.locator('button:has-text("Summary")')).toBeVisible();
  });

  test('should display search traces tab by default', async ({ page }) => {
    const searchTab = page.locator('button:has-text("Search Traces")');
    await expect(searchTab).toHaveClass(/text-blue-600/);
  });

  test('should display trace search results', async ({ page }) => {
    await expect(page.locator('text=POST /api/v1/auth/login')).toBeVisible();
    await expect(page.locator('text=POST /api/v1/jobs')).toBeVisible();
    await expect(page.locator('text=GET /api/v1/printers')).toBeVisible();
  });

  test('should click on trace to view details', async ({ page }) => {
    await page.click('text=POST /api/v1/auth/login');

    // Should switch to View Trace tab
    const traceTab = page.locator('button:has-text("View Trace")');
    await expect(traceTab).toHaveClass(/text-blue-600/);

    // Should display trace details
    await expect(page.locator('text=Trace ID')).toBeVisible();
    await expect(page.locator('text=Total Spans')).toBeVisible();
  });

  test('should display trace viewer', async ({ page }) => {
    // First select a trace
    await page.click('text=POST /api/v1/auth/login');

    await expect(page.locator('text=Trace Spans')).toBeVisible();
    await expect(page.locator('text=Services')).toBeVisible();
  });

  test('should switch to summary tab', async ({ page }) => {
    await page.click('button:has-text("Summary")');

    const summaryTab = page.locator('button:has-text("Summary")');
    await expect(summaryTab).toHaveClass(/text-blue-600/);

    await expect(page.locator('text=Service Breakdown')).toBeVisible();
  });

  test('should display service breakdown in summary', async ({ page }) => {
    await page.click('button:has-text("Summary")');

    await expect(page.locator('text=auth-service')).toBeVisible();
    await expect(page.locator('text=job-service')).toBeVisible();
    await expect(page.locator('text=registry-service')).toBeVisible();
  });

  test('should display Grafana link', async ({ page }) => {
    await expect(page.locator('text=Grafana Dashboards')).toBeVisible();
    await expect(page.locator('button:has-text("Open Grafana")')).toBeVisible();
  });

  test('should display Open Jaeger button', async ({ page }) => {
    await expect(page.locator('button:has-text("Open Jaeger")')).toBeVisible();
  });

  test('should filter by service', async ({ page }) => {
    await page.locator('select').selectOption('auth-service');

    const select = page.locator('select');
    await expect(select).toHaveValue('auth-service');
  });

  test('should change time range', async ({ page }) => {
    await page.click('button:has-text("6 hours")');

    const timeButton = page.locator('button:has-text("6 hours")');
    await expect(timeButton).toHaveClass(/bg-blue-600/);
  });

  test('should toggle auto-refresh', async ({ page }) => {
    const refreshButton = page.locator('button[title*="Auto-refresh"]');
    await expect(refreshButton).toBeVisible();

    await refreshButton.click();
    await expect(refreshButton).toHaveClass(/bg-green-100/);
  });

  test('should view trace in Jaeger link', async ({ page }) => {
    // First select a trace
    await page.click('text=POST /api/v1/auth/login');

    await expect(page.locator('a:has-text("View in Jaeger")')).toBeVisible();
  });

  test('should display trace duration in readable format', async ({ page }) => {
    await expect(page.locator('text=ms')).toBeVisible();
  });

  test('should display error trace count', async ({ page }) => {
    await expect(page.locator('text=23')).toBeVisible();
  });
});

test.describe('Observability Access Control', () => {
  test('should redirect non-admin users to dashboard', async ({ page }) => {
    const regularUser = mockUsers[0];

    await page.route('**/api/v1/auth/me', async (route) => {
      await mockApiResponse(route, regularUser);
    });

    await login(page);
    await page.goto('/observability');

    await page.waitForURL('**/dashboard');
    await expect(page.locator('h1')).toContainText('Welcome back');
  });
});

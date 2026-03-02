import { test, expect } from '@playwright/test';
import { LoginPage, MetricsDashboardPage } from '../helpers/page-objects';
import { testUsers } from '../helpers/test-data';

test.describe('Metrics Dashboard', () => {
  let metricsPage: MetricsDashboardPage;

  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    metricsPage = new MetricsDashboardPage(page);
    await metricsPage.navigate();
  });

  test('should display metrics dashboard heading', async ({ page }) => {
    await expect(metricsPage.heading).toBeVisible();
    await expect(metricsPage.heading).toContainText(/metrics dashboard/i);
  });

  test('should display summary metric cards', async ({ page }) => {
    await expect(metricsPage.metricCards.nth(0)).toBeVisible();
    await expect(metricsPage.metricCards.nth(1)).toBeVisible();
    await expect(metricsPage.metricCards.nth(2)).toBeVisible();
    await expect(metricsPage.metricCards.nth(3)).toBeVisible();
  });

  test('should display request rate card', async ({ page }) => {
    const requestRateCard = metricsPage.metricCards.filter({ hasText: /request rate/i });
    await expect(requestRateCard).toBeVisible();
  });

  test('should display error rate card', async ({ page }) => {
    const errorRateCard = metricsPage.metricCards.filter({ hasText: /error rate/i });
    await expect(errorRateCard).toBeVisible();
  });

  test('should display p95 latency card', async ({ page }) => {
    const latencyCard = metricsPage.metricCards.filter({ hasText: /p95 latency/i });
    await expect(latencyCard).toBeVisible();
  });

  test('should display alerts summary card', async ({ page }) => {
    const alertsCard = metricsPage.metricCards.filter({ hasText: /alerts/i });
    await expect(alertsCard).toBeVisible();
  });

  test('should have time range selector buttons', async ({ page }) => {
    await expect(metricsPage.timeRangeButtons.nth(0)).toBeVisible();
    await expect(metricsPage.timeRangeButtons.nth(1)).toBeVisible();
  });

  test('should allow changing time range', async ({ page }) => {
    // Click on a different time range
    await metricsPage.selectTimeRange('15 min');

    // Verify the button is now active
    const activeButton = page.locator('button.bg-blue-600').filter({ hasText: /15 min/i });
    await expect(activeButton).toBeVisible();
  });

  test('should have service selector buttons', async ({ page }) => {
    await expect(page.getByRole('button', { name: /all services/i })).toBeVisible();
    await expect(page.getByRole('button', { name: /auth-service/i })).toBeVisible();
    await expect(page.getByRole('button', { name: /job-service/i })).toBeVisible();
  });

  test('should allow filtering by service', async ({ page }) => {
    await metricsPage.selectService('auth-service');

    // Verify the service button is now active
    const activeButton = page.locator('button.bg-blue-600').filter({ hasText: /auth-service/i });
    await expect(activeButton).toBeVisible();
  });

  test('should have auto-refresh toggle button', async ({ page }) => {
    await expect(metricsPage.autoRefreshButton.first()).toBeVisible();
  });

  test('should toggle auto-refresh on and off', async ({ page }) => {
    // Initially off (gray background)
    const autoRefreshBtn = metricsPage.autoRefreshButton.first();

    // Click to enable
    await metricsPage.toggleAutoRefresh();

    // Should now have green background (enabled)
    await expect(autoRefreshBtn).toHaveClass(/bg-green-100/);

    // Click to disable
    await metricsPage.toggleAutoRefresh();

    // Should no longer have green background
    await expect(autoRefreshBtn).not.toHaveClass(/bg-green-100/);
  });
});

test.describe('Metrics Dashboard - Charts', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);
  });

  test('should display request rate chart', async ({ page }) => {
    const metricsPage = new MetricsDashboardPage(page);
    await metricsPage.navigate();

    await expect(metricsPage.requestRateChart).toBeVisible();
  });

  test('should display error rate chart', async ({ page }) => {
    const metricsPage = new MetricsDashboardPage(page);
    await metricsPage.navigate();

    await expect(metricsPage.errorRateChart).toBeVisible();
  });

  test('should display p95 latency chart', async ({ page }) => {
    const metricsPage = new MetricsDashboardPage(page);
    await metricsPage.navigate();

    await expect(metricsPage.latencyChart).toBeVisible();
  });

  test('charts should use recharts library', async ({ page }) => {
    const metricsPage = new MetricsDashboardPage(page);
    await metricsPage.navigate();

    // Check for SVG elements rendered by recharts
    const svgs = page.locator('svg');
    await expect(svgs.first()).toBeVisible();
  });
});

test.describe('Metrics Dashboard - Service Health', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);
  });

  test('should display service health section', async ({ page }) => {
    const metricsPage = new MetricsDashboardPage(page);
    await metricsPage.navigate();

    await expect(page.getByRole('heading', { name: /service health/i })).toBeVisible();
  });

  test('should display service health cards for all services', async ({ page }) => {
    const metricsPage = new MetricsDashboardPage(page);
    await metricsPage.navigate();

    // Check for service cards
    await expect(page.locator('.bg-white').filter({ hasText: /auth-service/i })).toBeVisible();
    await expect(page.locator('.bg-white').filter({ hasText: /job-service/i })).toBeVisible();
  });

  test('should display cpu usage for services', async ({ page }) => {
    const metricsPage = new MetricsDashboardPage(page);
    await metricsPage.navigate();

    const cpuLabel = page.getByText(/cpu/i);
    await expect(cpuLabel.first()).toBeVisible();
  });

  test('should display memory usage for services', async ({ page }) => {
    const metricsPage = new MetricsDashboardPage(page);
    await metricsPage.navigate();

    const memoryLabel = page.getByText(/memory/i);
    await expect(memoryLabel.first()).toBeVisible();
  });

  test('should display request rate for services', async ({ page }) => {
    const metricsPage = new MetricsDashboardPage(page);
    await metricsPage.navigate();

    const requestRateLabel = page.getByText(/request rate/i);
    await expect(requestRateLabel.first()).toBeVisible();
  });

  test('should display p95 latency for services', async ({ page }) => {
    const metricsPage = new MetricsDashboardPage(page);
    await metricsPage.navigate();

    const latencyLabel = page.getByText(/p95 latency/i);
    await expect(latencyLabel.first()).toBeVisible();
  });
});

test.describe('Metrics Dashboard - Navigation', () => {
  test('should navigate from sidebar', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    await page.getByRole('link', { name: /metrics/i }).click();
    await expect(page).toHaveURL('/metrics');
  });

  test('should link to monitoring from alerts card', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    const metricsPage = new MetricsDashboardPage(page);
    await metricsPage.navigate();

    // Click on alerts card (should navigate to monitoring)
    const alertsCard = metricsPage.metricCards.filter({ hasText: /alerts/i });
    await alertsCard.click();

    await expect(page).toHaveURL(/\/monitoring/);
  });
});

test.describe('Metrics Dashboard - Responsive Design', () => {
  test('should be mobile responsive', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });

    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    const metricsPage = new MetricsDashboardPage(page);
    await metricsPage.navigate();

    // Metric cards should stack vertically on mobile
    await expect(metricsPage.metricCards.first()).toBeVisible();
  });

  test('should work on tablet', async ({ page }) => {
    await page.setViewportSize({ width: 768, height: 1024 });

    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    const metricsPage = new MetricsDashboardPage(page);
    await metricsPage.navigate();

    await expect(metricsPage.heading).toBeVisible();
  });
});

test.describe('Metrics Dashboard - Access Control', () => {
  test('should require admin role to access', async ({ page }) => {
    // Login as regular user
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.user.email, testUsers.user.password);

    // Try to access metrics page
    await page.goto('/metrics');

    // Should redirect to dashboard (unauthorized)
    await expect(page).toHaveURL(/\/dashboard/);
  });

  test('should be accessible to admin users', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    const metricsPage = new MetricsDashboardPage(page);
    await metricsPage.navigate();

    await expect(metricsPage.heading).toBeVisible();
  });

  test('should be accessible to owner users', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.owner.email, testUsers.owner.password);

    const metricsPage = new MetricsDashboardPage(page);
    await metricsPage.navigate();

    await expect(metricsPage.heading).toBeVisible();
  });
});

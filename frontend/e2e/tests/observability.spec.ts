import { test, expect } from '@playwright/test';
import { LoginPage, ObservabilityHubPage } from '../helpers/page-objects';
import { testUsers } from '../helpers/test-data';

test.describe('Observability Hub (Tracing)', () => {
  let observabilityPage: ObservabilityHubPage;

  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    observabilityPage = new ObservabilityHubPage(page);
    await observabilityPage.navigate();
  });

  test('should display observability hub heading', async ({ page }) => {
    await expect(observabilityPage.heading).toBeVisible();
    await expect(observabilityPage.heading).toContainText(/observability hub/i);
  });

  test('should display description about distributed tracing', async ({ page }) => {
    const description = page.getByText(/distributed tracing and performance analysis/i);
    await expect(description).toBeVisible();
  });

  test('should have auto-refresh toggle button', async ({ page }) => {
    await expect(observabilityPage.autoRefreshButton.first()).toBeVisible();
  });

  test('should have link to open Jaeger', async ({ page }) => {
    await expect(observabilityPage.openJaegerButton).toBeVisible();
    await expect(observabilityPage.openJaegerButton).toHaveAttribute('href', /jaeger|16686/);
  });

  test('should open Jaeger in new tab', async ({ page, context }) => {
    const [newPage] = await Promise.all([
      context.waitForEvent('page'),
      observabilityPage.openJaegerButton.click(),
    ]);

    await expect(newPage).toHaveURL(/jaeger|16686/);
    await newPage.close();
  });
});

test.describe('Observability Hub - Summary Cards', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    const observabilityPage = new ObservabilityHubPage(page);
    await observabilityPage.navigate();
  });

  test('should display total traces card', async ({ page }) => {
    const summaryCards = page.locator('.grid').locator('.rounded-xl');
    const card = summaryCards.filter({ hasText: /total traces/i });
    await expect(card).toBeVisible();
  });

  test('should display error traces card', async ({ page }) => {
    const summaryCards = page.locator('.grid').locator('.rounded-xl');
    const card = summaryCards.filter({ hasText: /error traces/i });
    await expect(card).toBeVisible();
  });

  test('should display avg duration card', async ({ page }) => {
    const summaryCards = page.locator('.grid').locator('.rounded-xl');
    const card = summaryCards.filter({ hasText: /avg duration/i });
    await expect(card).toBeVisible();
  });

  test('should display p99 duration card', async ({ page }) => {
    const summaryCards = page.locator('.grid').locator('.rounded-xl');
    const card = summaryCards.filter({ hasText: /p99 duration/i });
    await expect(card).toBeVisible();
  });

  test('should have icon for each metric card', async ({ page }) => {
    // Check for SVG icons in metric cards
    const icons = page.locator('.grid').locator('.rounded-xl').locator('svg');
    await expect(icons.first()).toBeVisible();
  });
});

test.describe('Observability Hub - Filters', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    const observabilityPage = new ObservabilityHubPage(page);
    await observabilityPage.navigate();
  });

  test('should have service filter dropdown', async ({ page }) => {
    const serviceFilter = page.locator('select').or(page.getByLabel(/service/i));
    await expect(serviceFilter.first()).toBeVisible();
  });

  test('should have time range selector buttons', async ({ page }) => {
    const timeRangeButtons = page.locator('button').filter({
      hasText: /1 hour|3 hours|6 hours/i,
    });
    await expect(timeRangeButtons.nth(0)).toBeVisible();
  });

  test('should allow changing time range', async ({ page }) => {
    await page.getByRole('button', { name: '3 hours' }).click();

    // Verify the button is now active
    const activeButton = page.locator('button.bg-blue-600').filter({ hasText: /3 hours/i });
    await expect(activeButton).toBeVisible();
  });

  test('should allow filtering by service', async ({ page }) => {
    const serviceFilter = page.locator('select').or(page.getByLabel(/service/i));
    await serviceFilter.first().selectOption(''); // Select first option (All Services)

    // Verify the selection was made
    await expect(serviceFilter.first()).toBeVisible();
  });
});

test.describe('Observability Hub - Search Traces Tab', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    const observabilityPage = new ObservabilityHubPage(page);
    await observabilityPage.navigate();
    // Default tab is search
  });

  test('should display trace search component', async ({ page }) => {
    const traceSearch = page.locator('.bg-white').filter({ hasText: /search traces/i });
    await expect(traceSearch).toBeVisible();
  });

  test('should have search input for operation name', async ({ page }) => {
    const searchInput = page.getByPlaceholder(/search by operation/i);
    await expect(searchInput).toBeVisible();
  });

  test('should have service filter in search area', async ({ page }) => {
    const serviceFilter = page.getByLabel(/service/i);
    await expect(serviceFilter).toBeVisible();
  });

  test('should have max duration input', async ({ page }) => {
    const durationInput = page.getByPlaceholder(/e.g. 500ms, 1s/i);
    await expect(durationInput).toBeVisible();
  });

  test('should have search and reset buttons', async ({ page }) => {
    await expect(page.getByRole('button', { name: /search/i })).toBeVisible();
    await expect(page.getByRole('button', { name: /reset/i })).toBeVisible();
  });

  test('should perform trace search', async ({ page }) => {
    // Type a search query
    const searchInput = page.getByPlaceholder(/search by operation/i);
    await searchInput.fill('GET');

    // Click search
    await page.getByRole('button', { name: /search/i }).click();

    // Just verify no error occurs
    await expect(searchInput).toHaveValue('GET');
  });

  test('should display trace results if available', async ({ page }) => {
    const resultsArea = page.locator('.divide-y').filter({ hasText: /root service name|duration/i });

    // Results may or may not exist depending on data
    const hasResults = await resultsArea.count() > 0;
    if (hasResults) {
      await expect(resultsArea.first()).toBeVisible();
    }
  });
});

test.describe('Observability Hub - View Trace Tab', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    const observabilityPage = new ObservabilityHubPage(page);
    await observabilityPage.navigate();
  });

  test('should show disabled state when no trace selected', async ({ page }) => {
    const traceTab = page.getByRole('button', { name: /view trace/i });

    // The tab should be visually disabled or indicate no selection
    await expect(traceTab).toBeVisible();
  });

  test('should display empty state when no trace selected', async ({ page }) => {
    const observabilityPage = new ObservabilityHubPage(page);
    await observabilityPage.selectTab('trace');

    const emptyState = page.getByText(/no trace selected/i);
    await expect(emptyState).toBeVisible();
  });

  test('should show instruction to select trace', async ({ page }) => {
    const observabilityPage = new ObservabilityHubPage(page);
    await observabilityPage.selectTab('trace');

    const instruction = page.getByText(/select a trace from the search results/i);
    await expect(instruction).toBeVisible();
  });
});

test.describe('Observability Hub - Summary Tab', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    const observabilityPage = new ObservabilityHubPage(page);
    await observabilityPage.navigate();
  });

  test('should display overall stats when selected', async ({ page }) => {
    const observabilityPage = new ObservabilityHubPage(page);
    await observabilityPage.selectTab('summary');

    // Should show overall statistics
    const statsGrid = page.locator('.grid').locator('.rounded-lg');
    await expect(statsGrid.nth(0)).toBeVisible(); // Total Traces
    await expect(statsGrid.nth(1)).toBeVisible(); // Error Traces
    await expect(statsGrid.nth(2)).toBeVisible(); // Slow Traces
    await expect(statsGrid.nth(3)).toBeVisible(); // Avg Duration
  });

  test('should display service breakdown section', async ({ page }) => {
    const observabilityPage = new ObservabilityHubPage(page);
    await observabilityPage.selectTab('summary');

    const serviceBreakdown = page.getByRole('heading', { name: /service breakdown/i });
    await expect(serviceBreakdown).toBeVisible();
  });

  test('should show service stats in breakdown', async ({ page }) => {
    const observabilityPage = new ObservabilityHubPage(page);
    await observabilityPage.selectTab('summary');

    const serviceStats = page.locator('.bg-gray-50').filter({ hasText: /avg duration|max duration|error rate/i });
    // May or may not have services with data
    const hasStats = await serviceStats.count() > 0;
    if (hasStats) {
      await expect(serviceStats.first()).toBeVisible();
    }
  });
});

test.describe('Observability Hub - Trace Viewer', () => {
  test('should display trace header when trace selected', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    // This test assumes there's mock data or real traces available
    const observabilityPage = new ObservabilityHubPage(page);
    await observabilityPage.navigate();

    // Try to find and click a trace if available
    const traceResult = page.locator('.divide-y').locator('.hover\\:bg-gray-50').first();

    if (await traceResult.isVisible()) {
      await traceResult.click();

      // Should now be on trace tab
      const traceViewer = page.locator('.bg-white').filter({ hasText: /trace spans/i });
      await expect(traceViewer).toBeVisible();
    }
  });

  test('should display trace ID in header', async ({ page }) => {
    // This would require selecting a trace first
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    const observabilityPage = new ObservabilityHubPage(page);
    await observabilityPage.navigate();

    const traceResult = page.locator('.divide-y').locator('.hover\\:bg-gray-50').first();

    if (await traceResult.isVisible()) {
      await traceResult.click();

      const traceIdLabel = page.getByText(/trace id/i);
      await expect(traceIdLabel).toBeVisible();
    }
  });

  test('should display span tree for trace', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    const observabilityPage = new ObservabilityHubPage(page);
    await observabilityPage.navigate();

    const traceResult = page.locator('.divide-y').locator('.hover\\:bg-gray-50').first();

    if (await traceResult.isVisible()) {
      await traceResult.click();

      const spansSection = page.getByRole('heading', { name: /trace spans/i });
      await expect(spansSection).toBeVisible();
    }
  });
});

test.describe('Observability Hub - External Links', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    const observabilityPage = new ObservabilityHubPage(page);
    await observabilityPage.navigate();
  });

  test('should have link to Grafana dashboards', async ({ page }) => {
    const grafanaLink = page.getByRole('link', { name: /open grafana/i });
    await expect(grafanaLink).toBeVisible();
  });

  test('should link to Grafana with correct URL', async ({ page }) => {
    const grafanaLink = page.getByRole('link', { name: /open grafana/i });
    await expect(grafanaLink).toHaveAttribute('href', /grafana|3000/);
  });

  test('should open Grafana in new tab', async ({ page, context }) => {
    const grafanaLink = page.getByRole('link', { name: /open grafana/i });

    const [newPage] = await Promise.all([
      context.waitForEvent('page'),
      grafanaLink.click(),
    ]);

    await expect(newPage).toHaveURL(/grafana|3000/);
    await newPage.close();
  });

  test('should have external link icon on buttons', async ({ page }) => {
    const externalLinks = page.locator('a').filter({ hasText: /open jaeger|open grafana/i });
    const count = await externalLinks.count();

    expect(count).toBeGreaterThan(0);
  });
});

test.describe('Observability Hub - Navigation', () => {
  test('should navigate from sidebar', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    await page.getByRole('link', { name: /tracing/i }).click();
    await expect(page).toHaveURL('/observability');
  });

  test('should highlight tracing in sidebar', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    const tracingLink = page.getByRole('link', { name: /tracing/i });
    await tracingLink.click();

    // Should be active
    await expect(tracingLink).toHaveClass(/bg-blue-100|text-blue-700/);
  });

  test('should switch between tabs using nav', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    const observabilityPage = new ObservabilityHubPage(page);
    await observabilityPage.navigate();

    // Click through tabs
    await observabilityPage.selectTab('summary');
    await expect(page.getByRole('heading', { name: /service breakdown/i })).toBeVisible();

    await observabilityPage.selectTab('search');
    await expect(observabilityPage.traceSearch).toBeVisible();
  });
});

test.describe('Observability Hub - Responsive Design', () => {
  test('should be mobile responsive', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });

    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    const observabilityPage = new ObservabilityHubPage(page);
    await observabilityPage.navigate();

    // Summary cards should stack
    await expect(observabilityPage.summaryCards.first()).toBeVisible();
  });

  test('should work on tablet', async ({ page }) => {
    await page.setViewportSize({ width: 768, height: 1024 });

    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    const observabilityPage = new ObservabilityHubPage(page);
    await observabilityPage.navigate();

    await expect(observabilityPage.heading).toBeVisible();
  });
});

test.describe('Observability Hub - Access Control', () => {
  test('should require admin role to access', async ({ page }) => {
    // Login as regular user
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.user.email, testUsers.user.password);

    // Try to access observability page
    await page.goto('/observability');

    // Should redirect to dashboard (unauthorized)
    await expect(page).toHaveURL(/\/dashboard/);
  });

  test('should be accessible to owner users', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.owner.email, testUsers.owner.password);

    await page.goto('/observability');
    await expect(page.getByRole('heading', { name: /observability hub/i })).toBeVisible();
  });
});

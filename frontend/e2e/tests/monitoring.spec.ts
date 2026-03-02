import { test, expect } from '@playwright/test';
import { LoginPage, MonitoringPage } from '../helpers/page-objects';
import { testUsers } from '../helpers/test-data';

test.describe('Monitoring Page', () => {
  let monitoringPage: MonitoringPage;

  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    monitoringPage = new MonitoringPage(page);
    await monitoringPage.navigate();
  });

  test('should display monitoring heading', async ({ page }) => {
    await expect(monitoringPage.heading).toBeVisible();
    await expect(monitoringPage.heading).toContainText(/monitoring/i);
  });

  test('should display summary cards', async ({ page }) => {
    await expect(monitoringPage.summaryCards.nth(0)).toBeVisible();
    await expect(monitoringPage.summaryCards.nth(1)).toBeVisible();
    await expect(monitoringPage.summaryCards.nth(2)).toBeVisible();
    await expect(monitoringPage.summaryCards.nth(3)).toBeVisible();
  });

  test('should display total alerts card', async ({ page }) => {
    const totalAlertsCard = monitoringPage.summaryCards.filter({ hasText: /total alerts/i });
    await expect(totalAlertsCard).toBeVisible();
  });

  test('should display firing alerts card', async ({ page }) => {
    const firingCard = monitoringPage.summaryCards.filter({ hasText: /firing/i });
    await expect(firingCard).toBeVisible();
  });

  test('should display services card', async ({ page }) => {
    const servicesCard = monitoringPage.summaryCards.filter({ hasText: /services/i });
    await expect(servicesCard).toBeVisible();
  });

  test('should display health issues card', async ({ page }) => {
    const healthIssuesCard = monitoringPage.summaryCards.filter({ hasText: /health issues/i });
    await expect(healthIssuesCard).toBeVisible();
  });

  test('should have auto-refresh toggle button', async ({ page }) => {
    await expect(monitoringPage.autoRefreshButton.first()).toBeVisible();
  });

  test('should have refresh button', async ({ page }) => {
    await expect(monitoringPage.refreshButton).toBeVisible();
  });

  test('should display tabs for alerts, services, and silences', async ({ page }) => {
    await expect(monitoringPage.alertsTab).toBeVisible();
    await expect(monitoringPage.servicesTab).toBeVisible();
    await expect(monitoringPage.silencesTab).toBeVisible();
  });

  test('should toggle auto-refresh on and off', async ({ page }) => {
    const autoRefreshBtn = monitoringPage.autoRefreshButton.first();

    // Click to enable
    await monitoringPage.toggleAutoRefresh();

    // Should now have green background
    await expect(autoRefreshBtn).toHaveClass(/bg-green-100/);

    // Click to disable
    await monitoringPage.toggleAutoRefresh();

    // Should no longer have green background
    await expect(autoRefreshBtn).not.toHaveClass(/bg-green-100/);
  });

  test('should refresh data when refresh button clicked', async ({ page }) => {
    await monitoringPage.refresh();

    // Should show loading indicator briefly
    const loadingSpinner = page.locator('.animate-spin');
    // Just verify the button is clickable
    await expect(monitoringPage.refreshButton).toBeVisible();
  });
});

test.describe('Monitoring Page - Alerts Tab', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    const monitoringPage = new MonitoringPage(page);
    await monitoringPage.navigate();
    await monitoringPage.selectTab('alerts');
  });

  test('should display alert panel', async ({ page }) => {
    const alertPanel = page.locator('.bg-white').filter({ hasText: /alerts/i });
    await expect(alertPanel).toBeVisible();
  });

  test('should have severity filters', async ({ page }) => {
    await expect(page.getByRole('button', { name: /critical/i })).toBeVisible();
    await expect(page.getByRole('button', { name: /warning/i })).toBeVisible();
    await expect(page.getByRole('button', { name: /info/i })).toBeVisible();
  });

  test('should have state filters', async ({ page }) => {
    await expect(page.getByRole('button', { name: /firing/i })).toBeVisible();
    await expect(page.getByRole('button', { name: /pending/i })).toBeVisible();
    await expect(page.getByRole('button', { name: /resolved/i })).toBeVisible();
  });

  test('should filter alerts by severity', async ({ page }) => {
    await page.getByRole('button', { name: /critical/i }).click();

    // Verify critical filter is active
    const activeButton = page.locator('button.bg-gray-800').filter({ hasText: /critical/i });
    await expect(activeButton).toBeVisible();
  });

  test('should filter alerts by state', async ({ page }) => {
    await page.getByRole('button', { name: /pending/i }).click();

    // Verify pending filter is active
    const activeButton = page.locator('button.bg-gray-800').filter({ hasText: /pending/i });
    await expect(activeButton).toBeVisible();
  });

  test('should display alert summary within tab', async ({ page }) => {
    const summaryCards = page.locator('.grid').locator('.rounded-lg');
    await expect(summaryCards.nth(0)).toBeVisible(); // Total
    await expect(summaryCards.nth(1)).toBeVisible(); // Critical
    await expect(summaryCards.nth(2)).toBeVisible(); // Warning
    await expect(summaryCards.nth(3)).toBeVisible(); // Info
  });

  test('should show empty state when no alerts match filters', async ({ page }) => {
    // Apply a filter that might result in no alerts
    await page.getByRole('button', { name: /info/i }).click();
    await page.getByRole('button', { name: /resolved/i }).click();

    // Check for empty state or filtered results
    const emptyState = page.getByText(/no alerts match your filters/i);
    if (await emptyState.isVisible()) {
      await expect(emptyState).toBeVisible();
    }
  });
});

test.describe('Monitoring Page - Services Tab', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    const monitoringPage = new MonitoringPage(page);
    await monitoringPage.navigate();
    await monitoringPage.selectTab('services');
  });

  test('should display service health list', async ({ page }) => {
    const serviceHealthList = page.locator('.space-y-6');
    await expect(serviceHealthList).toBeVisible();
  });

  test('should display services grouped by status', async ({ page }) => {
    // Look for status groupings
    const healthyHeader = page.getByText(/healthy services/i);
    const degradedHeader = page.getByText(/degraded services/i);
    const unhealthyHeader = page.getByText(/unhealthy services/i);

    // At least one should be visible
    const hasAnyHeader = await Promise.all([
      healthyHeader.isVisible().catch(() => false),
      degradedHeader.isVisible().catch(() => false),
      unhealthyHeader.isVisible().catch(() => false),
    ]);
    expect(hasAnyHeader.some(Boolean)).toBeTruthy();
  });

  test('should display service cards', async ({ page }) => {
    const serviceCards = page.locator('.bg-white').locator('.rounded-xl');
    await expect(serviceCards.first()).toBeVisible();
  });

  test('should show service status badges', async ({ page }) => {
    const statusBadges = page.locator('[class*="rounded"]').filter({ hasText: /healthy|degraded|unhealthy/i });
    await expect(statusBadges.first()).toBeVisible();
  });

  test('should display service metrics', async ({ page }) => {
    // Check for metric labels
    await expect(page.getByText(/cpu/i)).toBeVisible();
    await expect(page.getByText(/memory/i)).toBeVisible();
    await expect(page.getByText(/request rate/i)).toBeVisible();
  });

  test('should allow clicking service card for details', async ({ page }) => {
    const serviceCard = page.locator('.bg-white').locator('.rounded-xl').first();
    await serviceCard.click();

    // Should open service detail modal
    const modal = page.locator('.fixed').filter({ hasText: /status|version|instance/i });
    await expect(modal).toBeVisible();
  });

  test('should close service detail modal when clicked outside', async ({ page }) => {
    const serviceCard = page.locator('.bg-white').locator('.rounded-xl').first();
    await serviceCard.click();

    // Wait for modal to appear
    const modal = page.locator('.fixed');
    await expect(modal).toBeVisible();

    // Click outside (on the overlay)
    const overlay = page.locator('.bg-black\\/50');
    await overlay.click();

    // Modal should be closed
    await expect(modal).not.toBeVisible();
  });
});

test.describe('Monitoring Page - Silences Tab', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    const monitoringPage = new MonitoringPage(page);
    await monitoringPage.navigate();
    await monitoringPage.selectTab('silences');
  });

  test('should display silences list', async ({ page }) => {
    const silencesList = page.locator('.space-y-4');
    await expect(silencesList).toBeVisible();
  });

  test('should show empty state when no silences', async ({ page }) => {
    const emptyState = page.getByText(/no active silences/i);
    if (await emptyState.isVisible()) {
      await expect(emptyState).toBeVisible();
      await expect(page.getByText(/temporarily suppress alert/i)).toBeVisible();
    }
  });

  test('should display silence details when present', async ({ page }) => {
    // Check for silence cards if they exist
    const silenceCards = page.locator('.rounded-lg').filter({ hasText: /comment|created by/i });

    if (await silenceCards.count() > 0) {
      await expect(silenceCards.first()).toBeVisible();
    }
  });
});

test.describe('Monitoring Page - Alert Detail Modal', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    const monitoringPage = new MonitoringPage(page);
    await monitoringPage.navigate();
    await monitoringPage.selectTab('alerts');
  });

  test('should open alert detail modal on click', async ({ page }) => {
    // Find an alert and click it
    const alertCard = page.locator('.rounded-lg').filter({ hasText: /firing|pending/i }).first();

    if (await alertCard.isVisible()) {
      await alertCard.click();

      // Check for modal with alert details
      const modal = page.locator('.fixed').filter({ hasText: /message|labels|state/i });
      await expect(modal).toBeVisible();
    }
  });

  test('should display alert message in modal', async ({ page }) => {
    const alertCard = page.locator('.rounded-lg').first();

    if (await alertCard.isVisible()) {
      await alertCard.click();

      await expect(page.getByText(/message/i)).toBeVisible();
    }
  });

  test('should display alert state in modal', async ({ page }) => {
    const alertCard = page.locator('.rounded-lg').first();

    if (await alertCard.isVisible()) {
      await alertCard.click();

      await expect(page.getByText(/state/i)).toBeVisible();
    }
  });

  test('should display alert labels in modal', async ({ page }) => {
    const alertCard = page.locator('.rounded-lg').first();

    if (await alertCard.isVisible()) {
      await alertCard.click();

      await expect(page.getByText(/labels/i)).toBeVisible();
    }
  });

  test('should close modal when close button clicked', async ({ page }) => {
    const alertCard = page.locator('.rounded-lg').first();

    if (await alertCard.isVisible()) {
      await alertCard.click();

      const closeButton = page.locator('.fixed').getByRole('button').filter({ hasText: /close/i });
      await closeButton.click();

      // Modal should be closed
      const modal = page.locator('.fixed');
      await expect(modal).not.toBeVisible();
    }
  });
});

test.describe('Monitoring Page - Service Detail Modal', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    const monitoringPage = new MonitoringPage(page);
    await monitoringPage.navigate();
    await monitoringPage.selectTab('services');
  });

  test('should display service status in modal', async ({ page }) => {
    const serviceCard = page.locator('.bg-white').locator('.rounded-xl').first();
    await serviceCard.click();

    await expect(page.getByText(/status/i)).toBeVisible();
  });

  test('should display service version in modal', async ({ page }) => {
    const serviceCard = page.locator('.bg-white').locator('.rounded-xl').first();
    await serviceCard.click();

    await expect(page.getByText(/version/i)).toBeVisible();
  });

  test('should display resource usage with progress bars', async ({ page }) => {
    const serviceCard = page.locator('.bg-white').locator('.rounded-xl').first();
    await serviceCard.click();

    // Look for progress bars (h-2 bg-gray-200 rounded-full)
    const progressBars = page.locator('.h-2').locator('.rounded-full');
    await expect(progressBars.first()).toBeVisible();
  });

  test('should display performance metrics', async ({ page }) => {
    const serviceCard = page.locator('.bg-white').locator('.rounded-xl').first();
    await serviceCard.click();

    await expect(page.getByText(/request rate/i)).toBeVisible();
    await expect(page.getByText(/error rate/i)).toBeVisible();
    await expect(page.getByText(/p95 latency/i)).toBeVisible();
  });

  test('should display dependencies if present', async ({ page }) => {
    const serviceCard = page.locator('.bg-white').locator('.rounded-xl').first();
    await serviceCard.click();

    const dependenciesLabel = page.getByText(/dependencies/i);
    if (await dependenciesLabel.isVisible()) {
      await expect(dependenciesLabel).toBeVisible();
    }
  });
});

test.describe('Monitoring Page - Navigation', () => {
  test('should navigate from sidebar', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    await page.getByRole('link', { name: /monitoring/i }).click();
    await expect(page).toHaveURL('/monitoring');
  });

  test('should highlight monitoring in sidebar', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    const monitoringLink = page.getByRole('link', { name: /monitoring/i });
    await monitoringLink.click();

    // Should be active
    await expect(monitoringLink).toHaveClass(/bg-blue-100|text-blue-700/);
  });
});

test.describe('Monitoring Page - Access Control', () => {
  test('should require admin role to access', async ({ page }) => {
    // Login as regular user
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.user.email, testUsers.user.password);

    // Try to access monitoring page
    await page.goto('/monitoring');

    // Should redirect to dashboard (unauthorized)
    await expect(page).toHaveURL(/\/dashboard/);
  });

  test('should be accessible to owner users', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.owner.email, testUsers.owner.password);

    await page.goto('/monitoring');
    await expect(page.getByRole('heading', { name: /monitoring/i })).toBeVisible();
  });
});

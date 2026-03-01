/**
 * Visual Regression Tests
 * Tests for UI consistency and visual appearance
 */
import { test, expect } from '@playwright/test';
import { setupAuthAndNavigate, mockUsers, mockPrinters, mockJobs } from '../helpers';
import { DashboardPage } from '../pages/DashboardPage';
import { JobsPage } from '../pages/JobsPage';
import { PrintersPage } from '../pages/PrintersPage';
import { AnalyticsPage } from '../pages/AnalyticsPage';
import { SettingsPage } from '../pages/SettingsPage';

// Visual regression configuration
const SNAPSHOT_BASE_DIR = 'screenshots/baseline';
const SNAPSHOT_CURRENT_DIR = 'screenshots/current';
const SNAPSHOT_DIFF_DIR = 'screenshots/diff';

/**
 * Visual regression test helper
 * Takes a screenshot and compares it to baseline
 */
async function compareScreenshots(
  page: any,
  name: string,
  options: {
    fullPage?: boolean;
    maxDiffPixels?: number;
    maxDiffPixelRatio?: number;
    threshold?: number;
  } = {}
) {
  const {
    fullPage = true,
    maxDiffPixels = 1000,
    maxDiffPixelRatio = 0.02,
    threshold = 0.2,
  } = options;

  // Take screenshot
  await page.screenshot({
    path: `${SNAPSHOT_CURRENT_DIR}/${name}.png`,
    fullPage,
  });

  // In real implementation, this would compare with baseline using pixelmatch or similar
  // For now, we just take the screenshot
  expect(await page.screenshot({ fullPage })).toMatchSnapshot(`${name}.png`, {
    maxDiffPixels,
    maxDiffPixelRatio,
    threshold,
  });
}

test.describe('Visual Regression - Authentication', () => {
  test('should match login page screenshot', async ({ page }) => {
    await page.goto('/login');
    await page.waitForLoadState('networkidle');

    await compareScreenshots(page, 'login-page', { fullPage: false });
  });

  test('should match registration form screenshot', async ({ page }) => {
    await page.goto('/login');
    await page.waitForLoadState('networkidle');

    // Switch to registration
    await page.click('button:has-text("Sign up"), a:has-text("Register")');
    await page.waitForTimeout(500);

    await compareScreenshots(page, 'registration-page', { fullPage: false });
  });

  test('should match form validation error states', async ({ page }) => {
    await page.goto('/login');
    await page.waitForLoadState('networkidle');

    // Trigger validation errors
    await page.fill('input[type="email"]', 'invalid-email');
    await page.fill('input[type="password"]', '123');
    await page.click('button[type="submit"]');

    await page.waitForTimeout(500);
    await compareScreenshots(page, 'login-validation-errors', { fullPage: false });
  });
});

test.describe('Visual Regression - Dashboard', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuthAndNavigate(page, '/dashboard');
  });

  test('should match dashboard layout', async ({ page }) => {
    await page.waitForLoadState('networkidle');
    await compareScreenshots(page, 'dashboard-layout');
  });

  test('should match stats cards appearance', async ({ page }) => {
    await page.waitForLoadState('networkidle');
    const statsSection = page.locator('[data-testid="stat-card"], .stat-card').first();
    await statsSection.scrollIntoViewIfNeeded();
    await compareScreenshots(page, 'dashboard-stats-cards', { fullPage: false });
  });

  test('should match environmental report section', async ({ page }) => {
    await page.waitForLoadState('networkidle');
    const envSection = page.locator('[data-testid="environmental-impact"]');
    await envSection.scrollIntoViewIfNeeded();
    await compareScreenshots(page, 'dashboard-environmental-report', { fullPage: false });
  });

  test('should match mobile responsive view', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 }); // iPhone SE
    await page.reload();
    await page.waitForLoadState('networkidle');
    await compareScreenshots(page, 'dashboard-mobile');
  });

  test('should match tablet responsive view', async ({ page }) => {
    await page.setViewportSize({ width: 768, height: 1024 }); // iPad
    await page.reload();
    await page.waitForLoadState('networkidle');
    await compareScreenshots(page, 'dashboard-tablet');
  });

  test('should match empty state', async ({ page }) => {
    // Mock empty data
    await page.route('**/api/v1/jobs*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: [], total: 0, limit: 50, offset: 0 }),
      });
    });

    await page.reload();
    await page.waitForLoadState('networkidle');
    await compareScreenshots(page, 'dashboard-empty-state');
  });
});

test.describe('Visual Regression - Jobs Page', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuthAndNavigate(page, '/jobs');
  });

  test('should match jobs list layout', async ({ page }) => {
    await page.waitForLoadState('networkidle');
    await compareScreenshots(page, 'jobs-list-layout');
  });

  test('should match job status badges', async ({ page }) => {
    await page.waitForLoadState('networkidle');
    const statusBadges = page.locator('[data-testid="job-status"], .status-badge').first();
    await statusBadges.scrollIntoViewIfNeeded();
    await compareScreenshots(page, 'job-status-badges', { fullPage: false });
  });

  test('should match job creation modal', async ({ page }) => {
    await page.waitForLoadState('networkidle');
    await page.click('button:has-text("New Job"), [data-testid="create-job-button"]');
    await page.waitForTimeout(500);
    await compareScreenshots(page, 'job-creation-modal', { fullPage: false });
  });

  test('should match job detail view', async ({ page }) => {
    await page.waitForLoadState('networkidle');
    const firstJob = page.locator('[data-testid="job-item"], .job-item').first();
    await firstJob.click();
    await page.waitForTimeout(500);
    await compareScreenshots(page, 'job-detail-view');
  });
});

test.describe('Visual Regression - Printers Page', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuthAndNavigate(page, '/printers');
  });

  test('should match printers grid view', async ({ page }) => {
    await page.waitForLoadState('networkidle');
    await compareScreenshots(page, 'printers-grid-view');
  });

  test('should match printer status indicators', async ({ page }) => {
    await page.waitForLoadState('networkidle');
    const statusIndicators = page.locator('[data-testid="printer-status"], .status-badge').first();
    await statusIndicators.scrollIntoViewIfNeeded();
    await compareScreenshots(page, 'printer-status-indicators', { fullPage: false });
  });

  test('should match printer discovery modal', async ({ page }) => {
    await page.waitForLoadState('networkidle');
    await page.click('button:has-text("Discover"), [data-testid="discover-printers-button"]');
    await page.waitForTimeout(500);
    await compareScreenshots(page, 'printer-discovery-modal');
  });
});

test.describe('Visual Regression - Analytics Page', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuthAndNavigate(page, '/analytics', mockUsers[1]); // Admin user
  });

  test('should match analytics dashboard layout', async ({ page }) => {
    await page.waitForLoadState('networkidle');
    await compareScreenshots(page, 'analytics-dashboard');
  });

  test('should match metric cards design', async ({ page }) => {
    await page.waitForLoadState('networkidle');
    const metricCards = page.locator('[data-testid="metric-card"], .metric-card');
    await metricCards.first().scrollIntoViewIfNeeded();
    await compareScreenshots(page, 'analytics-metric-cards', { fullPage: false });
  });

  test('should match charts appearance', async ({ page }) => {
    await page.waitForLoadState('networkidle');
    const charts = page.locator('[data-testid="charts-container"], .charts-container');
    await charts.scrollIntoViewIfNeeded();
    await compareScreenshots(page, 'analytics-charts');
  });

  test('should match audit logs table', async ({ page }) => {
    await page.waitForLoadState('networkidle');
    const auditLogs = page.locator('[data-testid="audit-logs-table"], .audit-logs-table');
    await auditLogs.scrollIntoViewIfNeeded();
    await compareScreenshots(page, 'analytics-audit-logs');
  });
});

test.describe('Visual Regression - Settings Page', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuthAndNavigate(page, '/settings');
  });

  test('should match settings tabs layout', async ({ page }) => {
    await page.waitForLoadState('networkidle');
    await compareScreenshots(page, 'settings-tabs', { fullPage: false });
  });

  test('should match profile form', async ({ page }) => {
    await page.waitForLoadState('networkidle');
    await page.click('button:has-text("Profile"), [data-testid="tab-profile"]');
    await page.waitForTimeout(500);
    await compareScreenshots(page, 'settings-profile-form', { fullPage: false });
  });

  test('should match security section', async ({ page }) => {
    await page.waitForLoadState('networkidle');
    await page.click('button:has-text("Security"), [data-testid="tab-security"]');
    await page.waitForTimeout(500);
    await compareScreenshots(page, 'settings-security-section', { fullPage: false });
  });

  test('should match preferences form', async ({ page }) => {
    await page.waitForLoadState('networkidle');
    await page.click('button:has-text("Preferences"), [data-testid="tab-preferences"]');
    await page.waitForTimeout(500);
    await compareScreenshots(page, 'settings-preferences-form', { fullPage: false });
  });
});

test.describe('Visual Regression - Dark Mode', () => {
  test('should match dashboard in dark mode', async ({ page }) => {
    // Set dark mode
    await page.addInitScript(() => {
      localStorage.setItem('theme', 'dark');
    });

    await setupAuthAndNavigate(page, '/dashboard');
    await page.waitForLoadState('networkidle');

    await compareScreenshots(page, 'dashboard-dark-mode');
  });

  test('should match jobs page in dark mode', async ({ page }) => {
    await page.addInitScript(() => {
      localStorage.setItem('theme', 'dark');
    });

    await setupAuthAndNavigate(page, '/jobs');
    await page.waitForLoadState('networkidle');

    await compareScreenshots(page, 'jobs-dark-mode');
  });
});

test.describe('Visual Regression - Component States', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuthAndNavigate(page, '/dashboard');
  });

  test('should match button states', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const button = page.locator('button').first();
    await button.scrollIntoViewIfNeeded();

    // Normal state
    await compareScreenshots(page, 'button-normal', { fullPage: false });

    // Hover state
    await button.hover();
    await page.waitForTimeout(200);
    await compareScreenshots(page, 'button-hover', { fullPage: false });

    // Focus state
    await button.focus();
    await page.waitForTimeout(200);
    await compareScreenshots(page, 'button-focus', { fullPage: false });
  });

  test('should match form input states', async ({ page }) => {
    await page.goto('/settings');
    await page.waitForLoadState('networkidle');

    const input = page.locator('input[type="text"]').first();

    // Normal state
    await compareScreenshots(page, 'input-normal', { fullPage: false });

    // Focus state
    await input.focus();
    await page.waitForTimeout(200);
    await compareScreenshots(page, 'input-focus', { fullPage: false });

    // Filled state
    await input.fill('Test value');
    await input.blur();
    await page.waitForTimeout(200);
    await compareScreenshots(page, 'input-filled', { fullPage: false });

    // Error state
    await input.fill('');
    await input.blur();
    await page.waitForTimeout(200);
    await compareScreenshots(page, 'input-error', { fullPage: false });
  });

  test('should match loading states', async ({ page }) => {
    // Mock slow API
    await page.route('**/api/v1/jobs*', async () => {
      await new Promise((resolve) => setTimeout(resolve, 2000));
    });

    await setupAuthAndNavigate(page, '/jobs');
    await page.waitForTimeout(500);

    await compareScreenshots(page, 'loading-spinner', { fullPage: false });
  });

  test('should match empty states across pages', async ({ page }) => {
    // Mock empty responses
    await page.route('**/api/v1/**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: [], total: 0 }),
      });
    });

    await setupAuthAndNavigate(page, '/jobs');
    await compareScreenshots(page, 'empty-state-jobs');

    await page.goto('/printers');
    await page.waitForLoadState('networkidle');
    await compareScreenshots(page, 'empty-state-printers');
  });
});

test.describe('Visual Regression - Notifications and Alerts', () => {
  test('should match toast notification styles', async ({ page }) => {
    await setupAuthAndNavigate(page, '/dashboard');
    await page.waitForLoadState('networkidle');

    // Trigger different toast types via API mock
    await page.evaluate(() => {
      // Success toast
      const event = new CustomEvent('show-toast', {
        detail: { type: 'success', message: 'Operation successful' },
      });
      window.dispatchEvent(event);
    });

    await page.waitForTimeout(500);
    await compareScreenshots(page, 'toast-success', { fullPage: false });
  });

  test('should match modal appearance', async ({ page }) => {
    await setupAuthAndNavigate(page, '/jobs');
    await page.waitForLoadState('networkidle');

    await page.click('button:has-text("New Job")');
    await page.waitForTimeout(500);

    await compareScreenshots(page, 'modal-appearance', { fullPage: false });
  });

  test('should match dropdown menus', async ({ page }) => {
    await setupAuthAndNavigate(page, '/dashboard');
    await page.waitForLoadState('networkidle');

    // Click user menu
    await page.click('[data-testid="user-info"], .user-info');
    await page.waitForTimeout(500);

    await compareScreenshots(page, 'dropdown-menu', { fullPage: false });
  });
});

test.describe('Visual Regression - Accessibility', () => {
  test('should maintain focus indicators', async ({ page }) => {
    await setupAuthAndNavigate(page, '/dashboard');
    await page.waitForLoadState('networkidle');

    // Tab through elements and verify focus is visible
    await page.keyboard.press('Tab');
    await page.waitForTimeout(200);
    await compareScreenshots(page, 'focus-indicator-1', { fullPage: false });

    await page.keyboard.press('Tab');
    await page.waitForTimeout(200);
    await compareScreenshots(page, 'focus-indicator-2', { fullPage: false });
  });

  test('should have proper color contrast', async ({ page }) => {
    // This test would check WCAG contrast ratios
    // For now, we take screenshots for manual verification
    await setupAuthAndNavigate(page, '/dashboard');
    await page.waitForLoadState('networkidle');

    await compareScreenshots(page, 'dashboard-contrast-check');
  });
});

test.describe('Visual Regression - Cross-Browser', () => {
  // These tests run in all browsers configured in playwright.config.ts
  test('should have consistent layout across browsers', async ({ page }) => {
    await setupAuthAndNavigate(page, '/dashboard');
    await page.waitForLoadState('networkidle');

    await compareScreenshots(page, `dashboard-${test.info().project?.name || 'default'}`);
  });
});

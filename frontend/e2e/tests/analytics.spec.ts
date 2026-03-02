import { test, expect } from '@playwright/test';
import { LoginPage, AnalyticsPage } from '../helpers/page-objects';
import { testUsers } from '../helpers/test-data';

test.describe('Analytics Dashboard', () => {
  let loginPage: LoginPage;
  let analyticsPage: AnalyticsPage;

  test.beforeEach(async ({ page }) => {
    loginPage = new LoginPage(page);
    analyticsPage = new AnalyticsPage(page);

    // Mock analytics API
    await page.route('**/api/v1/analytics/**', (route) => {
      const url = route.request().url();
      if (url.includes('/usage')) {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([
            {
              id: 'stat-1',
              statDate: '2024-02-27',
              pagesPrinted: 45,
              colorPages: 12,
              jobsCount: 8,
              jobsCompleted: 7,
              jobsFailed: 1,
              totalBytes: 5242880,
              estimatedCost: 2.35,
              co2Grams: 8.9,
              treesSaved: 0.004,
            },
            {
              id: 'stat-2',
              statDate: '2024-02-26',
              pagesPrinted: 32,
              colorPages: 8,
              jobsCount: 5,
              jobsCompleted: 5,
              jobsFailed: 0,
              totalBytes: 3145728,
              estimatedCost: 1.67,
              co2Grams: 6.3,
              treesSaved: 0.003,
            },
            {
              id: 'stat-3',
              statDate: '2024-02-25',
              pagesPrinted: 58,
              colorPages: 20,
              jobsCount: 12,
              jobsCompleted: 11,
              jobsFailed: 1,
              totalBytes: 8388608,
              estimatedCost: 3.45,
              co2Grams: 11.5,
              treesSaved: 0.005,
            },
          ]),
        });
      } else if (url.includes('/environment')) {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            pagesPrinted: 1234,
            co2Grams: 245.6,
            treesSaved: 0.12,
            period: '30d',
          }),
        });
      } else if (url.includes('/summary')) {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            totalJobs: 125,
            completedJobs: 118,
            failedJobs: 7,
            totalPages: 1234,
            colorPages: 345,
            totalCost: 45.67,
            avgJobsPerDay: 8.5,
          }),
        });
      } else {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({}),
        });
      }
    });

    await loginPage.login(testUsers.admin.email, testUsers.admin.password);
  });

  test('should display analytics page', async ({ page }) => {
    await page.goto('/analytics');

    await expect(page.getByRole('heading', { name: /analytics/i })).toBeVisible();
  });

  test('should display period selector buttons', async ({ page }) => {
    await page.goto('/analytics');

    await expect(page.getByRole('button', { name: /7 days/i })).toBeVisible();
    await expect(page.getByRole('button', { name: /30 days/i })).toBeVisible();
    await expect(page.getByRole('button', { name: /90 days/i })).toBeVisible();
    await expect(page.getByRole('button', { name: /12 months/i })).toBeVisible();
  });

  test('should change period when clicking buttons', async ({ page }) => {
    await page.goto('/analytics');

    const thirtyDaysButton = page.getByRole('button', { name: /30 days/i });
    await thirtyDaysButton.click();

    await expect(thirtyDaysButton).toHaveClass(/bg-blue-/);
  });

  test('should display summary metric cards', async ({ page }) => {
    await page.goto('/analytics');

    await expect(page.getByText('Total Jobs')).toBeVisible();
    await expect(page.getByText('Total Pages')).toBeVisible();
    await expect(page.getByText('Color Pages')).toBeVisible();
    await expect(page.getByText('Total Cost')).toBeVisible();
  });

  test('should display metric values', async ({ page }) => {
    await page.goto('/analytics');

    await expect(page.getByText('125')).toBeVisible();
    await expect(page.getByText('1,234')).toBeVisible();
    await expect(page.getByText('345')).toBeVisible();
    await expect(page.getByText('$45.67')).toBeVisible();
  });

  test('should display usage chart', async ({ page }) => {
    await page.goto('/analytics');

    await expect(page.locator('svg').first()).toBeVisible();
  });

  test('should display environmental impact section', async ({ page }) => {
    await page.goto('/analytics');

    await expect(page.getByText(/environmental impact/i)).toBeVisible();
    await expect(page.getByText(/co2 saved/i)).toBeVisible();
    await expect(page.getByText(/trees saved/i)).toBeVisible();
  });

  test('should display environmental statistics', async ({ page }) => {
    await page.goto('/analytics');

    await expect(page.getByText('245.6')).toBeVisible();
    await expect(page.getByText('0.12')).toBeVisible();
  });

  test('should show loading state', async ({ page }) => {
    // Add delay to mock API
    await page.route('**/api/v1/analytics/**', (route) => {
      setTimeout(() => {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([]),
        });
      }, 100);
    });

    await page.goto('/analytics');

    await expect(page.locator('.animate-spin, .loading')).first().toBeVisible();
  });
});

test.describe('Analytics - Job Statistics', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);

    await page.route('**/api/v1/analytics/**', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          byStatus: [
            { status: 'completed', count: 118, percentage: 94.4 },
            { status: 'failed', count: 7, percentage: 5.6 },
          ],
          byPrinter: [
            { printerName: 'HP LaserJet Pro', count: 75, percentage: 60 },
            { printerName: 'Canon PIXMA', count: 50, percentage: 40 },
          ],
          byUser: [
            { userName: 'John Doe', count: 45, percentage: 36 },
            { userName: 'Jane Smith', count: 38, percentage: 30.4 },
            { userName: 'Bob Johnson', count: 42, percentage: 33.6 },
          ],
        }),
      });
    });

    await loginPage.login(testUsers.admin.email, testUsers.admin.password);
  });

  test('should display job status breakdown', async ({ page }) => {
    await page.goto('/analytics');

    await expect(page.getByText('Completed')).toBeVisible();
    await expect(page.getByText('Failed')).toBeVisible();
  });

  test('should display printer usage breakdown', async ({ page }) => {
    await page.goto('/analytics');

    await expect(page.getByText('HP LaserJet Pro')).toBeVisible();
    await expect(page.getByText('Canon PIXMA')).toBeVisible();
  });

  test('should display user usage breakdown', async ({ page }) => {
    await page.goto('/analytics');

    await expect(page.getByText('John Doe')).toBeVisible();
    await expect(page.getByText('Jane Smith')).toBeVisible();
    await expect(page.getByText('Bob Johnson')).toBeVisible();
  });
});

test.describe('Analytics - Trends', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);

    await page.route('**/api/v1/analytics/**', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          trends: {
            jobsChange: 12.5,
            pagesChange: -5.3,
            costChange: 8.2,
          },
        }),
      });
    });

    await loginPage.login(testUsers.admin.email, testUsers.admin.password);
  });

  test('should display trend indicators', async ({ page }) => {
    await page.goto('/analytics');

    await expect(page.getByText(/12.5%/)).toBeVisible();
  });

  test('should show positive trends in green', async ({ page }) => {
    await page.goto('/analytics');

    const positiveTrend = page.locator('.text-green-600, .text-green-500').first();
    await expect(positiveTrend).toBeVisible();
  });

  test('should show negative trends in red', async ({ page }) => {
    await page.goto('/analytics');

    const negativeTrend = page.locator('.text-red-600, .text-red-500').first();
    await expect(negativeTrend).toBeVisible();
  });
});

test.describe('Analytics - Export', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);

    await page.route('**/api/v1/analytics/**', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({}),
      });
    });

    await loginPage.login(testUsers.admin.email, testUsers.admin.password);
  });

  test('should display export button', async ({ page }) => {
    await page.goto('/analytics');

    await expect(page.getByRole('button', { name: /export/i })).toBeVisible();
  });

  test('should open export modal', async ({ page }) => {
    await page.goto('/analytics');

    await page.getByRole('button', { name: /export/i }).click();

    await expect(page.getByText(/export analytics/i)).toBeVisible();
  });

  test('should show export format options', async ({ page }) => {
    await page.goto('/analytics');

    await page.getByRole('button', { name: /export/i }).click();

    await expect(page.getByText('CSV')).toBeVisible();
    await expect(page.getByText('PDF')).toBeVisible();
    await expect(page.getByText('Excel')).toBeVisible();
  });
});

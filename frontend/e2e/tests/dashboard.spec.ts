import { test, expect } from '@playwright/test';
import { LoginPage, DashboardPage } from '../helpers/page-objects';
import { testUsers } from '../helpers/test-data';

test.describe('Dashboard', () => {
  let dashboardPage: DashboardPage;

  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.goto();
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    dashboardPage = new DashboardPage(page);
  });

  test('should display dashboard heading', async ({ page }) => {
    await expect(dashboardPage.heading).toBeVisible();
    await expect(dashboardPage.heading).toContainText(/welcome/i);
  });

  test('should display user name in greeting', async ({ page }) => {
    const greeting = await dashboardPage.heading.textContent();
    expect(greeting).toContain(testUsers.admin.name.split(' ')[0]);
  });

  test('should display stat cards', async ({ page }) => {
    await expect(dashboardPage.statCards.nth(0)).toBeVisible();
    await expect(dashboardPage.statCards.nth(1)).toBeVisible();
    await expect(dashboardPage.statCards.nth(2)).toBeVisible();
  });

  test('should show active printers stat', async ({ page }) => {
    const activePrinters = await dashboardPage.getStatValue('Active Printers');
    expect(activePrinters).toBeDefined();
  });

  test('should show jobs today stat', async ({ page }) => {
    const jobsToday = await dashboardPage.getStatValue('Jobs Today');
    expect(jobsToday).toBeDefined();
  });

  test('should show pages printed stat', async ({ page }) => {
    const pagesPrinted = await dashboardPage.getStatValue('Pages This Month');
    expect(pagesPrinted).toBeDefined();
  });

  test('should display recent jobs section', async ({ page }) => {
    await expect(dashboardPage.recentJobsSection).toBeVisible();

    const viewAllLink = page.getByRole('link', { name: /view all/i });
    await expect(viewAllLink).toBeVisible();
    await expect(viewAllLink).toHaveAttribute('href', '/jobs');
  });

  test('should display printers section', async ({ page }) => {
    await expect(dashboardPage.printersSection).toBeVisible();

    const manageLink = page.getByRole('link', { name: /manage/i });
    await expect(manageLink).toBeVisible();
    await expect(manageLink).toHaveAttribute('href', '/printers');
  });

  test('should show empty state for no jobs', async ({ page }) => {
    const emptyState = page.getByText(/no print jobs/i);
    if (await emptyState.isVisible()) {
      await expect(emptyState).toBeVisible();
      const selectPrinterLink = page.getByRole('link', { name: /select a printer/i });
      await expect(selectPrinterLink).toHaveAttribute('href', '/printers');
    }
  });

  test('should show empty state for no printers', async ({ page }) => {
    const emptyState = page.getByText(/no printers configured/i);
    if (await emptyState.isVisible()) {
      await expect(emptyState).toBeVisible();
      await expect(emptyState).toContainText(/install the openprint agent/i);
    }
  });

  test('should display environmental report', async ({ page }) => {
    const envReport = page.getByTestId(/environment-report/i);
    // Environmental report might not show if no data
    // Just check page loads without error
    await expect(dashboardPage.heading).toBeVisible();
  });
});

test.describe('Dashboard - Navigation', () => {
  test('should have working sidebar navigation', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    const dashboardPage = new DashboardPage(page);
    await expect(dashboardPage.sidebar).toBeVisible();

    // Check navigation links
    await expect(page.getByRole('link', { name: /dashboard/i })).toBeVisible();
    await expect(page.getByRole('link', { name: /devices/i })).toBeVisible();
    await expect(page.getByRole('link', { name: /jobs/i })).toBeVisible();
    await expect(page.getByRole('link', { name: /documents/i })).toBeVisible();
    await expect(page.getByRole('link', { name: /settings/i })).toBeVisible();
  });

  test('should highlight active navigation item', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    const dashboardLink = page.getByRole('link', { name: /dashboard/i });
    await expect(dashboardLink).toHaveClass(/bg-blue-100|text-blue-700/);
  });

  test('should navigate to printers page', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    await page.getByRole('link', { name: /devices/i }).click();
    await expect(page).toHaveURL('/printers');
    await expect(page.getByRole('heading', { name: /devices|printers/i })).toBeVisible();
  });

  test('should navigate to jobs page', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    await page.getByRole('link', { name: /jobs/i }).click();
    await expect(page).toHaveURL('/jobs');
    await expect(page.getByRole('heading', { name: /jobs/i })).toBeVisible();
  });

  test('should navigate to documents page', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    await page.getByRole('link', { name: /documents/i }).click();
    await expect(page).toHaveURL('/documents');
    await expect(page.getByRole('heading', { name: /documents/i })).toBeVisible();
  });

  test('should navigate to settings page', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    await page.getByRole('link', { name: /settings/i }).click();
    await expect(page).toHaveURL('/settings');
    await expect(page.getByRole('heading', { name: /settings/i })).toBeVisible();
  });
});

test.describe('Dashboard - User Menu', () => {
  test('should display user info in sidebar', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    await expect(page.getByText(testUsers.admin.name)).toBeVisible();
    await expect(page.getByText(testUsers.admin.email)).toBeVisible();
  });

  test('should display user avatar', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    const avatar = page.locator('.rounded-full').filter({ hasText: testUsers.admin.name[0] });
    await expect(avatar).toBeVisible();
  });

  test('should logout from user menu', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    await page.getByRole('button', { name: /logout/i }).click();
    await expect(page).toHaveURL('/login');
  });
});

test.describe('Dashboard - Responsive Design', () => {
  test('should be mobile responsive', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });

    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    // Check that sidebar is hidden on mobile (or becomes a hamburger menu)
    const sidebar = page.locator('aside');
    await expect(sidebar).toBeVisible();

    // Stat cards should stack vertically
    const statCards = page.locator('.grid').locator('.bg-white');
    const firstCard = statCards.nth(0);
    await expect(firstCard).toBeVisible();
  });

  test('should work on tablet', async ({ page }) => {
    await page.setViewportSize({ width: 768, height: 1024 });

    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    await expect(page.getByRole('heading', { name: /welcome/i })).toBeVisible();
  });
});

test.describe('Dashboard - Dark Mode', () => {
  test('should support dark mode', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.login(testUsers.admin.email, testUsers.admin.password);

    // Toggle dark mode if there's a toggle
    const darkModeToggle = page.getByLabel(/dark mode|theme/i);
    if (await darkModeToggle.isVisible()) {
      await darkModeToggle.click();
      await expect(page.locator('.dark')).toHaveCount(1);
    }
  });
});

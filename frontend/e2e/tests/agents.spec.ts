import { test, expect } from '@playwright/test';
import { LoginPage, DashboardPage } from '../helpers/page-objects';
import { testUsers } from '../helpers/test-data';

test.describe('Agents Management', () => {
  let loginPage: LoginPage;
  let dashboardPage: DashboardPage;

  test.beforeEach(async ({ page }) => {
    loginPage = new LoginPage(page);
    dashboardPage = new DashboardPage(page);

    // Mock agents API
    await page.route('**/api/v1/agents*', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([
          {
            id: 'agent-1',
            name: 'WORKSTATION-001',
            status: 'online',
            platform: 'windows',
            platformVersion: 'Windows 11 Pro',
            agentVersion: '1.0.0',
            ipAddress: '192.168.1.100',
            lastHeartbeat: new Date().toISOString(),
            printerCount: 2,
            jobQueueDepth: 0,
            createdAt: '2024-01-01T00:00:00Z',
          },
          {
            id: 'agent-2',
            name: 'WORKSTATION-002',
            status: 'online',
            platform: 'windows',
            platformVersion: 'Windows 10 Pro',
            agentVersion: '1.0.0',
            ipAddress: '192.168.1.101',
            lastHeartbeat: new Date(Date.now() - 120000).toISOString(),
            printerCount: 1,
            jobQueueDepth: 3,
            createdAt: '2024-01-02T00:00:00Z',
          },
          {
            id: 'agent-3',
            name: 'FINANCE-PC',
            status: 'offline',
            platform: 'windows',
            platformVersion: 'Windows 11 Enterprise',
            agentVersion: '0.9.5',
            ipAddress: '192.168.1.150',
            lastHeartbeat: new Date(Date.now() - 86400000).toISOString(),
            printerCount: 1,
            createdAt: '2024-01-03T00:00:00Z',
          },
          {
            id: 'agent-4',
            name: 'RECEPTION-DESK',
            status: 'error',
            platform: 'windows',
            platformVersion: 'Windows 10 Pro',
            agentVersion: '1.0.1',
            ipAddress: '192.168.1.200',
            lastHeartbeat: new Date(Date.now() - 300000).toISOString(),
            printerCount: 1,
            jobQueueDepth: 0,
            createdAt: '2024-01-15T00:00:00Z',
          },
        ]),
      });
    });

    await loginPage.login(testUsers.admin.email, testUsers.admin.password);
  });

  test('should display agents list page', async ({ page }) => {
    await page.goto('/agents');

    await expect(page.getByRole('heading', { name: /agents/i })).toBeVisible();
    await expect(page.getByText('Manage Windows print agents and monitor their status')).toBeVisible();
  });

  test('should display all agents', async ({ page }) => {
    await page.goto('/agents');

    await expect(page.getByText('WORKSTATION-001')).toBeVisible();
    await expect(page.getByText('WORKSTATION-002')).toBeVisible();
    await expect(page.getByText('FINANCE-PC')).toBeVisible();
    await expect(page.getByText('RECEPTION-DESK')).toBeVisible();
  });

  test('should show agent status badges', async ({ page }) => {
    await page.goto('/agents');

    await expect(page.getByText('Online')).toBeVisible();
    await expect(page.getByText('Offline')).toBeVisible();
    await expect(page.getByText('Error')).toBeVisible();
  });

  test('should filter agents by status', async ({ page }) => {
    await page.goto('/agents');

    // Click online filter
    const onlineFilter = page.getByRole('button', { name: /online/i }).first();
    await onlineFilter.click();

    // Should show filter as active
    await expect(onlineFilter).toHaveClass(/bg-green-/);
  });

  test('should navigate to agent detail page', async ({ page }) => {
    // Mock agent detail API
    await page.route('**/api/v1/agents/*/detail*', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: 'agent-1',
          name: 'WORKSTATION-001',
          status: 'online',
          platform: 'windows',
          platformVersion: 'Windows 11 Pro',
          agentVersion: '1.0.0',
          ipAddress: '192.168.1.100',
          lastHeartbeat: new Date().toISOString(),
          printerCount: 2,
          jobQueueDepth: 0,
          printers: [],
          jobHistory: [],
          healthMetrics: {
            uptime: 86400,
            totalJobsProcessed: 150,
            successfulJobs: 145,
            failedJobs: 5,
            successRate: 96.7,
          },
        }),
      });
    });

    await page.goto('/agents');

    // Click on first agent
    await page.getByText('WORKSTATION-001').click();

    // Should navigate to detail page
    await page.waitForURL('**/agents/**');
    await expect(page.getByText('Back to Agents')).toBeVisible();
  });

  test('should display agent detail tabs', async ({ page }) => {
    // Mock agent detail API
    await page.route('**/api/v1/agents/*/detail*', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: 'agent-1',
          name: 'WORKSTATION-001',
          status: 'online',
          printers: [],
          jobHistory: [],
          healthMetrics: {
            uptime: 86400,
            totalJobsProcessed: 150,
            successfulJobs: 145,
            failedJobs: 5,
            successRate: 96.7,
          },
        }),
      });
    });

    await page.goto('/agents/agent-1');

    await expect(page.getByRole('tab', { name: /overview/i })).toBeVisible();
    await expect(page.getByRole('tab', { name: /printers/i })).toBeVisible();
    await expect(page.getByRole('tab', { name: /jobs/i })).toBeVisible();
    await expect(page.getByRole('tab', { name: /health/i })).toBeVisible();
  });

  test('should switch between agent detail tabs', async ({ page }) => {
    await page.route('**/api/v1/agents/*/detail*', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: 'agent-1',
          name: 'WORKSTATION-001',
          status: 'online',
          printers: [],
          jobHistory: [],
          healthMetrics: {},
        }),
      });
    });

    await page.goto('/agents/agent-1');

    // Click Printers tab
    await page.getByRole('tab', { name: /printers/i }).click();
    await expect(page.getByRole('tab', { name: /printers/i, selected: true })).toBeVisible();

    // Click Jobs tab
    await page.getByRole('tab', { name: /jobs/i }).click();
    await expect(page.getByRole('tab', { name: /jobs/i, selected: true })).toBeVisible();

    // Click Health tab
    await page.getByRole('tab', { name: /health/i }).click();
    await expect(page.getByRole('tab', { name: /health/i, selected: true })).toBeVisible();
  });

  test('should display agent information cards on detail page', async ({ page }) => {
    await page.route('**/api/v1/agents/*/detail*', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: 'agent-1',
          name: 'WORKSTATION-001',
          status: 'online',
          platform: 'windows',
          platformVersion: 'Windows 11 Pro',
          agentVersion: '1.0.0',
          ipAddress: '192.168.1.100',
          printerCount: 2,
          jobQueueDepth: 0,
          printers: [],
          jobHistory: [],
          healthMetrics: {
            uptime: 86400,
            totalJobsProcessed: 150,
            successfulJobs: 145,
            failedJobs: 5,
            successRate: 96.7,
          },
        }),
      });
    });

    await page.goto('/agents/agent-1');

    await expect(page.getByText('Status')).toBeVisible();
    await expect(page.getByText('Printers')).toBeVisible();
    await expect(page.getByText('Jobs Processed')).toBeVisible();
    await expect(page.getByText('Success Rate')).toBeVisible();
  });

  test('should show discover printers button', async ({ page }) => {
    await page.route('**/api/v1/agents/*/detail*', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: 'agent-1',
          name: 'WORKSTATION-001',
          status: 'online',
          printers: [],
        }),
      });
    });

    await page.goto('/agents/agent-1');
    await expect(page.getByRole('button', { name: /discover printers/i })).toBeVisible();
  });

  test('should show empty state when no agents', async ({ page }) => {
    // Mock empty response
    await page.route('**/api/v1/agents*', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([]),
      });
    });

    await page.goto('/agents');

    await expect(page.getByText(/no agents found/i)).toBeVisible();
    await expect(page.getByText(/install the openprint agent/i)).toBeVisible();
  });

  test('should display agent platform information', async ({ page }) => {
    await page.goto('/agents');

    await expect(page.getByText('Windows 11 Pro')).toBeVisible();
    await expect(page.getByText('Windows 10 Pro')).toBeVisible();
    await expect(page.getByText('Windows 11 Enterprise')).toBeVisible();
  });

  test('should display agent IP addresses', async ({ page }) => {
    await page.goto('/agents');

    await expect(page.getByText('192.168.1.100')).toBeVisible();
    await expect(page.getByText('192.168.1.101')).toBeVisible();
    await expect(page.getByText('192.168.1.150')).toBeVisible();
  });

  test('should display job queue depth', async ({ page }) => {
    await page.goto('/agents');

    // Agent 2 has 3 jobs in queue
    await expect(page.getByText('3')).toBeVisible();
  });
});

test.describe('Discovered Printers', () => {
  test.beforeEach(async ({ page }) => {
    const loginPage = new LoginPage(page);

    // Mock discovered printers API
    await page.route('**/api/v1/discovered-printers*', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          printers: [
            {
              id: 'printer-discovered-1',
              agentId: 'agent-1',
              name: 'HP LaserJet Pro M404n',
              driver: 'HP Universal Printing PCL 6',
              port: '9100',
              type: 'network',
              status: 'available',
              capabilities: {
                supportsColor: false,
                supportsDuplex: true,
                supportedPaperSizes: ['A4', 'Letter'],
              },
            },
            {
              id: 'printer-discovered-2',
              agentId: 'agent-1',
              name: 'Canon PIXMA G6020',
              driver: 'Canon G6020 series',
              port: 'USB001',
              type: 'local',
              status: 'available',
              capabilities: {
                supportsColor: true,
                supportsDuplex: true,
                supportedPaperSizes: ['A4', 'Letter', 'A3'],
              },
            },
            {
              id: 'printer-discovered-3',
              agentId: 'agent-2',
              name: 'Brother HL-L5100DN',
              driver: 'Brother HL-L5100DN series',
              port: '192.168.1.50',
              type: 'network',
              status: 'offline',
              capabilities: {
                supportsColor: false,
                supportsDuplex: true,
                supportedPaperSizes: ['A4', 'Letter'],
              },
            },
          ],
          total: 3,
        }),
      });
    });

    await loginPage.login(testUsers.admin.email, testUsers.admin.password);
  });

  test('should display discovered printers page', async ({ page }) => {
    await page.goto('/discovered-printers');

    await expect(page.getByRole('heading', { name: /discovered printers/i })).toBeVisible();
  });

  test('should display printer count summary', async ({ page }) => {
    await page.goto('/discovered-printers');

    await expect(page.getByText(/total printers/i)).toBeVisible();
    await expect(page.getByText(/local/i)).toBeVisible();
    await expect(page.getByText(/network/i)).toBeVisible();
  });

  test('should display printers in table', async ({ page }) => {
    await page.goto('/discovered-printers');

    await expect(page.getByText('HP LaserJet Pro M404n')).toBeVisible();
    await expect(page.getByText('Canon PIXMA G6020')).toBeVisible();
    await expect(page.getByText('Brother HL-L5100DN')).toBeVisible();
  });

  test('should filter printers by type', async ({ page }) => {
    await page.goto('/discovered-printers');

    const networkFilter = page.getByRole('button', { name: /network/i });
    await networkFilter.click();

    // Should show filter as active
    await expect(networkFilter).toHaveClass(/active/);
  });

  test('should search printers', async ({ page }) => {
    await page.goto('/discovered-printers');

    const searchInput = page.getByPlaceholder(/search/i);
    await searchInput.fill('HP');

    await expect(page.getByText('HP LaserJet Pro M404n')).toBeVisible();
  });

  test('should show empty state when no printers', async ({ page }) => {
    await page.route('**/api/v1/discovered-printers*', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ printers: [], total: 0 }),
      });
    });

    await page.goto('/discovered-printers');

    await expect(page.getByText(/no printers found/i)).toBeVisible();
  });
});

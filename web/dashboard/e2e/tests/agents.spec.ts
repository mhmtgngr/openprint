import { test, expect } from '@playwright/test';
import { login, mockAgents, mockDiscoveredPrinters } from '../helpers';

test.describe('Agents', () => {
  test.beforeEach(async ({ page }) => {
    // Mock agents API response
    await page.route('**/api/v1/agents*', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(mockAgents),
      });
    });

    await login(page);
    await page.goto('/agents');
  });

  test('should display agents list page', async ({ page }) => {
    await expect(page.getByRole('heading', { name: 'Agents' })).toBeVisible();
    await expect(
      page.getByText('Manage Windows print agents and monitor their status')
    ).toBeVisible();
  });

  test('should display all agents', async ({ page }) => {
    // Check that agents are displayed
    await expect(page.getByText('WORKSTATION-001')).toBeVisible();
    await expect(page.getByText('WORKSTATION-002')).toBeVisible();
    await expect(page.getByText('FINANCE-PC')).toBeVisible();
  });

  test('should show agent status badges', async ({ page }) => {
    // Online status badge
    await expect(page.locator('.bg-green-100').first()).toBeVisible();
    await expect(page.getByText('Online')).toBeVisible();
  });

  test('should filter agents by status', async ({ page }) => {
    // Click online filter
    await page.click('button:has-text("Online")');

    // Should only show online agents
    const agentCount = await page.locator('ul li').count();
    expect(agentCount).toBeGreaterThan(0);
  });

  test('should navigate to agent detail page', async ({ page }) => {
    // Mock agent detail API
    await page.route('**/api/v1/agents/*/detail*', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(mockAgents[0]),
      });
    });

    // Click on first agent
    await page.click('ul li:first-child');

    // Should navigate to detail page
    await page.waitForURL('**/agents/**');
    await expect(page.getByText('Back to Agents')).toBeVisible();
  });

  test('should display agent actions', async ({ page }) => {
    // Check for delete button
    await expect(
      page.locator('button[title="Delete agent"]').first()
    ).toBeVisible();
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

    await page.reload();

    await expect(page.getByText('No agents found')).toBeVisible();
    await expect(
      page.getByText('Install the OpenPrint Agent on Windows machines')
    ).toBeVisible();
  });
});

test.describe('Agent Detail', () => {
  test.beforeEach(async ({ page }) => {
    // Mock agent API
    await page.route('**/api/v1/agents/*', (route) => {
      if (route.request().url().includes('/detail')) {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            ...mockAgents[0],
            printers: mockDiscoveredPrinters,
            jobHistory: [],
            healthMetrics: {
              uptime: 86400,
              totalJobsProcessed: 150,
              successfulJobs: 145,
              failedJobs: 5,
              averageResponseTime: 2500,
              successRate: 96.7,
              weeklyJobCounts: [],
            },
          }),
        });
      } else {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(mockAgents[0]),
        });
      }
    });

    await login(page);
    await page.goto('/agents/agent-1');
  });

  test('should display agent detail page', async ({ page }) => {
    await expect(page.getByText('Back to Agents')).toBeVisible();
    await expect(page.getByText('WORKSTATION-001')).toBeVisible();
  });

  test('should display agent tabs', async ({ page }) => {
    await expect(page.getByRole('button', { name: /Overview/i })).toBeVisible();
    await expect(page.getByRole('button', { name: /Printers/i })).toBeVisible();
    await expect(page.getByRole('button', { name: /Jobs/i })).toBeVisible();
    await expect(page.getByRole('button', { name: /Health/i })).toBeVisible();
  });

  test('should switch between tabs', async ({ page }) => {
    // Click Printers tab
    await page.click('button:has-text("Printers")');
    await expect(page.getByText('Printers')).toBeVisible();

    // Click Jobs tab
    await page.click('button:has-text("Jobs")');
    await expect(page.getByText('Jobs')).toBeVisible();

    // Click Health tab
    await page.click('button:has-text("Health")');
    await expect(page.getByText('Weekly Job Activity')).toBeVisible();
  });

  test('should show discover printers button', async ({ page }) => {
    await expect(page.getByText('Discover Printers')).toBeVisible();
  });

  test('should display agent information cards', async ({ page }) => {
    await expect(page.getByText('Status')).toBeVisible();
    await expect(page.getByText('Printers')).toBeVisible();
    await expect(page.getByText('Jobs Processed')).toBeVisible();
    await expect(page.getByText('Success Rate')).toBeVisible();
  });
});

test.describe('Discovered Printers', () => {
  test.beforeEach(async ({ page }) => {
    // Mock discovered printers API
    await page.route('**/api/v1/discovered-printers*', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          printers: mockDiscoveredPrinters,
          total: mockDiscoveredPrinters.length,
        }),
      });
    });

    await login(page);
    await page.goto('/discovered-printers');
  });

  test('should display discovered printers page', async ({ page }) => {
    await expect(
      page.getByRole('heading', { name: 'Discovered Printers' })
    ).toBeVisible();
  });

  test('should display printer count summary', async ({ page }) => {
    await expect(page.getByText(/Total Printers/)).toBeVisible();
    await expect(page.getByText(/Local/)).toBeVisible();
    await expect(page.getByText(/Network/)).toBeVisible();
  });

  test('should display printers table', async ({ page }) => {
    await expect(page.getByText('HP LaserJet Pro')).toBeVisible();
    await expect(page.getByText('Canon PIXMA')).toBeVisible();
  });

  test('should filter printers by type', async ({ page }) => {
    await page.click('button:has-text("Local")');

    // Should filter the list
    const localButton = page.locator('button:has-text("Local")');
    await expect(localButton).toHaveClass(/bg-blue-100/);
  });

  test('should search printers', async ({ page }) => {
    const searchInput = page.getByPlaceholder('Search printers by name or driver');
    await searchInput.fill('HP');

    // Should filter results
    await expect(page.getByText('HP LaserJet Pro')).toBeVisible();
  });

  test('should show empty state when no printers', async ({ page }) => {
    await page.route('**/api/v1/discovered-printers*', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ printers: [], total: 0 }),
      });
    });

    await page.reload();

    await expect(page.getByText('No printers found')).toBeVisible();
  });
});

test.describe('Job Assignments', () => {
  test.beforeEach(async ({ page }) => {
    // Mock job assignments API
    await page.route('**/api/v1/job-assignments*', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              id: 'assignment-1',
              jobId: 'job-1',
              agentId: 'agent-1',
              status: 'assigned',
              priority: 1,
              assignedAt: '2024-02-27T10:00:00Z',
              job: {
                id: 'job-1',
                documentName: 'Test Document.pdf',
                pageCount: 5,
              },
              agent: {
                id: 'agent-1',
                name: 'WORKSTATION-001',
                status: 'online',
              },
            },
          ],
          total: 1,
          limit: 50,
          offset: 0,
        }),
      });
    });

    // Mock jobs API
    await page.route('**/api/v1/jobs*', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [],
          total: 0,
          limit: 50,
          offset: 0,
        }),
      });
    });

    await login(page);
    await page.goto('/job-assignments');
  });

  test('should display job assignments page', async ({ page }) => {
    await expect(
      page.getByRole('heading', { name: 'Job Assignments' })
    ).toBeVisible();
    await expect(
      page.getByText('Assign print jobs to specific agents or users')
    ).toBeVisible();
  });

  test('should display new assignment button', async ({ page }) => {
    await expect(page.getByRole('button', { name: 'New Assignment' })).toBeVisible();
  });

  test('should open assignment modal', async ({ page }) => {
    await page.click('button:has-text("New Assignment")');
    await expect(page.getByText('Create New Assignment')).toBeVisible();
    await expect(page.getByText('Select Job')).toBeVisible();
  });

  test('should display assignments list', async ({ page }) => {
    await expect(page.getByText('Test Document.pdf')).toBeVisible();
    await expect(page.getByText('WORKSTATION-001')).toBeVisible();
  });
});

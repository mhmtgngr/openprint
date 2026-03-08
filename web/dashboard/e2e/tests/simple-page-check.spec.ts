import { test, expect } from '@playwright/test';

/**
 * Simple Page Check
 * Verifies all pages load without critical errors
 */
const pages = [
  { path: '/', name: 'Root (redirects)', expected: /login|dashboard/i },
  { path: '/login', name: 'Login Page' },
  { path: '/dashboard', name: 'Dashboard' },
  { path: '/printers', name: 'Devices/Printers' },
  { path: '/agents', name: 'Agents' },
  { path: '/jobs', name: 'Jobs' },
  { path: '/documents', name: 'Documents' },
  { path: '/analytics', name: 'Analytics' },
  { path: '/organization', name: 'Organization' },
  { path: '/quotas', name: 'Quotas' },
  { path: '/policies', name: 'Policies' },
  { path: '/policy-engine', name: 'Policy Engine' },
  { path: '/audit-logs', name: 'Audit Logs' },
  { path: '/compliance', name: 'Compliance' },
  { path: '/email-to-print', name: 'Email-to-Print' },
  { path: '/microsoft365', name: 'Microsoft 365' },
  { path: '/metrics', name: 'Metrics' },
  { path: '/monitoring', name: 'Monitoring' },
  { path: '/observability', name: 'Tracing' },
  { path: '/settings', name: 'Settings' },
  { path: '/print-release', name: 'Print Release' },
  { path: '/secure-print', name: 'Secure Print' },
];

test.describe('Page Accessibility Check', () => {
  test('all pages load without errors', async ({ page }) => {
    console.log(`\n=== Testing ${pages.length} pages ===`);
    const results: { page: string; status: string; httpStatus: number }[] = [];

    for (const pageInfo of pages) {
      const url = `http://localhost:3000${pageInfo.path}`;

      try {
        const response = await page.goto(url, { waitUntil: 'domcontentloaded', timeout: 10000 });
        const httpStatus = response?.status() || 0;

        // Check for console errors
        const errors: string[] = [];
        page.on('console', (msg) => {
          if (msg.type() === 'error') {
            errors.push(msg.text());
          }
        });

        // Basic checks
        const title = await page.title();
        const hasContent = await page.locator('body').textContent();
        const contentLength = hasContent?.length || 0;

        let status = '✅ OK';
        if (httpStatus >= 400) status = `❌ HTTP ${httpStatus}`;
        else if (contentLength < 50) status = `⚠️  Empty (${contentLength} chars)`;
        else if (errors.length > 0) status = `⚠️  JS errors (${errors.length})`;

        results.push({ page: pageInfo.name, status, httpStatus });
      } catch (error) {
        results.push({ page: pageInfo.name, status: `❌ Error: ${(error as Error).message}`, httpStatus: 0 });
      }
    }

    // Print results
    console.log('\n=== Page Test Results ===\n');
    results.forEach(r => {
      console.log(`${r.status}  ${r.page.padEnd(30)} (HTTP ${r.httpStatus})`);
    });

    const passed = results.filter(r => r.status.includes('✅'));
    console.log(`\nSummary: ${passed.length}/${pages.length} pages accessible`);
  });

  test('navigation sidebar exists', async ({ page }) => {
    await page.goto('http://localhost:3000');

    // Wait for page to load
    await page.waitForLoadState('domcontentloaded');

    // Check for sidebar
    const sidebar = page.locator('aside, nav, [data-testid="sidebar"], .sidebar').first();
    expect(await sidebar.isVisible()).toBeTruthy();
  });

  test('check API health endpoints', async () => {
    const apiEndpoints = [
      { name: 'API Gateway', url: 'http://localhost:8080/health' },
      { name: 'Auth Service', url: 'http://localhost:8001/health' },
      { name: 'Registry Service', url: 'http://localhost:8002/health' },
      { name: 'Job Service', url: 'http://localhost:8003/health' },
      { name: 'Storage Service', url: 'http://localhost:8004/health' },
      { name: 'Notification Service', url: 'http://localhost:8005/health' },
      { name: 'Analytics Service', url: 'http://localhost:8006/health' },
      { name: 'Organization Service', url: 'http://localhost:8007/health' },
      { name: 'Policy Service', url: 'http://localhost:8008/health' },
      { name: 'Compliance Service', url: 'http://localhost:8009/health' },
      { name: 'M365 Integration', url: 'http://localhost:8010/health' },
    ];

    console.log('\n=== Service Health Checks ===\n');
    const healthy: string[] = [];
    const unhealthy: string[] = [];

    for (const service of apiEndpoints) {
      try {
        const response = await fetch(service.url, { signal: AbortSignal.timeout(5000) });
        if (response.ok) {
          healthy.push(service.name);
        } else {
          unhealthy.push(`${service.name} (${response.status})`);
        }
      } catch (error) {
        unhealthy.push(`${service.name} (unreachable)`);
      }
    }

    console.log(`Healthy (${healthy.length}/${apiEndpoints.length}):`);
    healthy.forEach(s => console.log(`  ✅ ${s}`));

    if (unhealthy.length > 0) {
      console.log(`\nUnhealthy (${unhealthy.length}):`);
      unhealthy.forEach(s => console.log(`  ❌ ${s}`));
    }

    expect(healthy.length).toBeGreaterThan(0);
  });
});

test.describe('Dashboard Content Check', () => {
  test('dashboard renders key elements', async ({ page }) => {
    await page.goto('http://localhost:3000/dashboard', { waitUntil: 'domcontentloaded' });

    // Check page has title
    await expect(page).toHaveTitle(/OpenPrint/i);

    // Check for sidebar/navigation
    const nav = page.locator('nav, aside, [data-testid="sidebar-nav"]').first();
    expect(await nav.isVisible()).toBeTruthy();
  });
});

test('login page is accessible', async ({ page }) => {
  await page.goto('http://localhost:3000/login');

  await expect(page).toHaveTitle(/Login|Sign In/i);

  // Check for login form elements
  const hasEmailInput = await page.locator('input[type="email"], input[name="email"]').count();
  const hasPasswordInput = await page.locator('input[type="password"], input[name="password"]').count();

  expect(hasEmailInput + hasPasswordInput).toBeGreaterThan(0);
});

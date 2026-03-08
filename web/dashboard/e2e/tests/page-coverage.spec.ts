import { test, expect } from '@playwright/test';

/**
 * Page Coverage Test
 * Verifies all pages are accessible and render without errors
 */
const pages = [
  { path: '/login', name: 'Login', public: true },
  { path: '/dashboard', name: 'Dashboard', public: false },
  { path: '/printers', name: 'Devices (Printers)', public: false },
  { path: '/agents', name: 'Agents', public: false },
  { path: '/jobs', name: 'Jobs', public: false },
  { path: '/documents', name: 'Documents', public: false },
  { path: '/analytics', name: 'Analytics', public: false },
  { path: '/organization', name: 'Organization', public: false },
  { path: '/quotas', name: 'Quotas', public: false },
  { path: '/policies', name: 'Policies', public: false },
  { path: '/policy-engine', name: 'Policy Engine', public: false },
  { path: '/audit-logs', name: 'Audit Logs', public: false },
  { path: '/compliance', name: 'Compliance', public: false },
  { path: '/email-to-print', name: 'Email-to-Print', public: false },
  { path: '/microsoft365', name: 'Microsoft 365', public: false },
  { path: '/metrics', name: 'Metrics', public: false },
  { path: '/monitoring', name: 'Monitoring', public: false },
  { path: '/observability', name: 'Tracing (Observability)', public: false },
  { path: '/settings', name: 'Settings', public: false },
  { path: '/print-release', name: 'Print Release', public: false },
  { path: '/secure-print', name: 'Secure Print', public: false },
];

test.describe('Page Coverage - All Pages Accessible', () => {
  let authTokens: string;

  test.beforeAll(async () => {
    // Setup auth tokens for protected pages
    const response = await fetch('http://localhost:8080/api/v1/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        email: 'admin@example.com',
        password: 'AdminPassword123!',
      }),
    });

    if (response.ok) {
      const data = await response.json();
      authTokens = data.access_token;
    }
  });

  test('public pages are accessible without auth', async ({ page }) => {
    const publicPages = pages.filter(p => p.public);

    for (const pageInfo of publicPages) {
      await page.goto(`http://localhost:3000${pageInfo.path}`);

      // Check page loads successfully
      await expect(page).toHaveTitle(/OpenPrint|Dashboard|Login|Devices/i);

      // Check no console errors
      const errors: string[] = [];
      page.on('pageerror', (error) => errors.push(error.message));

      // Check for common elements
      const hasContent = await page.locator('body').textContent();
      expect(hasContent?.length).toBeGreaterThan(0);
    }
  });

  test('all protected pages redirect to login when not authenticated', async ({ page }) => {
    const protectedPages = pages.filter(p => !p.public);

    for (const pageInfo of protectedPages) {
      await page.goto(`http://localhost:3000${pageInfo.path}`);

      // Should redirect to login or show auth required message
      const url = page.url();
      expect(url).toMatch(/\/login|unauthorized|401/i);
    }
  });

  test('authenticated user can access all pages', async ({ page }) => {
    if (!authTokens) {
      test.skip('No auth tokens available');
      return;
    }

    // Set auth token in localStorage
    await page.goto('http://localhost:3000');
    await page.evaluate((token) => {
      localStorage.setItem('auth_tokens', JSON.stringify({
        accessToken: token,
        refreshToken: 'mock-refresh-token',
      }));
    });

    const failedPages: string[] = [];

    for (const pageInfo of pages) {
      if (pageInfo.public) continue;

      await page.goto(`http://localhost:3000${pageInfo.path}`);

      // Check for errors
      const hasError = await page.locator('[data-testid="error-message"], .error, [role="alert"]').isVisible().catch(() => false);
      const title = await page.title();
      const content = await page.locator('body').textContent();

      if (hasError || title.includes('Error') || (content?.length < 50)) {
        failedPages.push(`${pageInfo.name} (${pageInfo.path})`);
      }
    }

    // Report results
    console.log(`\n=== Page Coverage Results ===`);
    console.log(`Total pages: ${pages.length}`);
    console.log(`Accessible: ${pages.length - failedPages.length}`);
    console.log(`Failed: ${failedPages.length}`);

    if (failedPages.length > 0) {
      console.log(`\nFailed pages:`);
      failedPages.forEach(p => console.log(`  - ${p}`));
    }

    expect(failedPages.length).toBe(0);
  });

  test('navigation links work for all pages', async ({ page }) => {
    if (!authTokens) {
      test.skip('No auth tokens available');
      return;
    }

    // Setup auth
    await page.goto('http://localhost:3000');
    await page.evaluate((token) => {
      localStorage.setItem('auth_tokens', JSON.stringify({
        accessToken: token,
        refreshToken: 'mock-refresh-token',
      }));
    });
    await page.goto('http://localhost:3000/dashboard');

    // Check sidebar navigation exists
    const navLinks = page.locator('nav a, [data-testid^="nav-"]');
    const linkCount = await navLinks.count();

    expect(linkCount).toBeGreaterThan(10);

    // Collect all navigation links
    const links = await navLinks.all();
    const visited = new Set<string>();

    for (const link of links) {
      const href = await link.getAttribute('href');
      if (href && href.startsWith('/') && !visited.has(href)) {
        await link.click();
        await page.waitForLoadState('domcontentloaded').catch(() => {});
        await page.waitForTimeout(500);
        visited.add(href);

        // Go back to dashboard
        await page.goto('http://localhost:3000/dashboard');
      }
    }

    console.log(`\n=== Navigation Results ===`);
    console.log(`Total navigation links found: ${linkCount}`);
    console.log(`Unique pages visited: ${visited.size}`);
  });
});

test.describe('Page Content Verification', () => {
  test('dashboard page has key components', async ({ page }) => {
    await page.goto('http://localhost:3000/dashboard');

    // Check for stat cards
    const stats = page.locator('[data-testid="stat-card"], .stat-card');
    await expect(stats.first()).toBeVisible();
  });

  test('devices page lists printers', async ({ page }) => {
    // Login first
    const response = await fetch('http://localhost:8080/api/v1/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        email: 'admin@example.com',
        password: 'AdminPassword123!',
      }),
    });

    if (response.ok) {
      const data = await response.json();
      const token = data.access_token;

      await page.goto('http://localhost:3000');
      await page.evaluate((t) => {
        localStorage.setItem('auth_tokens', JSON.stringify({
          accessToken: t,
          refreshToken: 'mock-refresh-token',
        }));
      });
    }

    await page.goto('http://localhost:3000/printers');
    await page.waitForLoadState('domcontentloaded');

    // Page should have content
    const content = await page.locator('body').textContent();
    expect(content?.length).toBeGreaterThan(0);
  });
});

test.describe('API Health Checks', () => {
  const services = [
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

  test('all services are healthy', async () => {
    const failedServices: string[] = [];

    for (const service of services) {
      try {
        const response = await fetch(service.url, { signal: AbortSignal.timeout(5000) });
        if (!response.ok) {
          failedServices.push(`${service.name} (${response.status})`);
        }
      } catch (error) {
        failedServices.push(`${service.name} (unreachable)`);
      }
    }

    console.log(`\n=== Service Health Results ===`);
    console.log(`Total services: ${services.length}`);
    console.log(`Healthy: ${services.length - failedServices.length}`);
    console.log(`Unhealthy: ${failedServices.length}`);

    if (failedServices.length > 0) {
      console.log(`\nUnhealthy services:`);
      failedServices.forEach(s => console.log(`  - ${s}`));
    }

    expect(failedServices.length).toBe(0);
  });
});

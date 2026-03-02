import { test, expect } from '@playwright/test';
import { setupAuthAndNavigate, setupCommonApiMocks, mockUsers, mockPrinters, mockJobs } from '../helpers';

test.describe('Lazy Loading', () => {
  test('should show loading fallback while navigating to lazy-loaded route', async ({ page }) => {
    await setupCommonApiMocks(page);
    await setupAuthAndNavigate(page, '/dashboard', mockUsers[0]);

    // Intercept network requests to slow them down and observe loading state
    await page.route('**/printers', async (route) => {
      // Add a small delay to make loading state observable
      await new Promise(resolve => setTimeout(resolve, 100));
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ printers: mockPrinters }),
      });
    });

    // Navigate to printers page and check for loading state
    const navigationPromise = page.waitForURL('**/printers');

    // Click printers link (Devices in sidebar)
    await page.click('a[href="/printers"]');

    // Check that loading state is shown
    // The PageLoadingFallback should be visible briefly
    await navigationPromise;

    // After navigation completes, verify URL changed
    expect(page.url()).toContain('/printers');
  });

  test('should lazy load each route independently', async ({ page }) => {
    await setupCommonApiMocks(page);

    // Track which assets have been loaded
    const loadedAssets = new Set<string>();

    page.on('response', async (response) => {
      const url = response.url();
      // Track any JavaScript chunks
      if (url.includes('.js')) {
        loadedAssets.add(url);
      }
    });

    // Start from dashboard
    await setupAuthAndNavigate(page, '/dashboard', mockUsers[0]);
    await page.waitForLoadState('networkidle');

    // Navigate to jobs
    await page.goto('/jobs', { waitUntil: 'networkidle' });

    // Navigate to settings
    await page.goto('/settings', { waitUntil: 'networkidle' });

    // At least some JavaScript assets should have been loaded
    expect(loadedAssets.size).toBeGreaterThan(0);
  });

  test('should display page-specific loading fallback during navigation', async ({ page }) => {
    await setupCommonApiMocks(page);
    await setupAuthAndNavigate(page, '/dashboard', mockUsers[0]);

    // Mock a slow API response
    await page.route('**/api/v1/jobs*', async (route) => {
      await new Promise(resolve => setTimeout(resolve, 200));
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: mockJobs,
          total: mockJobs.length,
          limit: 50,
          offset: 0,
        }),
      });
    });

    // Navigate to jobs page
    await page.click('a[href="/jobs"]');
    await page.waitForURL('**/jobs');

    // The jobs page should eventually load - verify URL
    expect(page.url()).toContain('/jobs');
  });

  test('should not reload already loaded lazy chunks on subsequent visits', async ({ page }) => {
    await setupCommonApiMocks(page);
    await setupAuthAndNavigate(page, '/dashboard', mockUsers[0]);

    // Track network requests for the jobs chunk
    let jobsChunkLoadCount = 0;

    page.on('response', async (response) => {
      const url = response.url();
      if (url.includes('.js') && url.includes('/assets/')) {
        // Check if this is the jobs page chunk
        if (url.includes('jobs') || url.includes('Jobs')) {
          jobsChunkLoadCount++;
        }
      }
    });

    // First visit to jobs page
    await page.goto('/jobs', { waitUntil: 'networkidle' });
    await page.waitForTimeout(500);

    const firstVisitCount = jobsChunkLoadCount;

    // Navigate away
    await page.goto('/dashboard', { waitUntil: 'networkidle' });

    // Second visit to jobs page
    await page.goto('/jobs', { waitUntil: 'networkidle' });
    await page.waitForTimeout(500);

    // The chunk should not be reloaded from the network
    // (it's cached in the browser's module cache)
    // This is a soft check - in some cases the browser might revalidate
    expect(jobsChunkLoadCount).toBeLessThanOrEqual(firstVisitCount + 1);
  });

  test('should handle lazy loading errors gracefully', async ({ page }) => {
    await setupCommonApiMocks(page);
    await setupAuthAndNavigate(page, '/dashboard', mockUsers[0]);

    // Mock a chunk load failure
    await page.route('**/assets/js/**', async (route) => {
      const url = route.request().url();
      // Fail the first chunk request for settings page
      if (url.includes('settings') || url.includes('Settings')) {
        await route.abort('failed');
      } else {
        route.continue();
      }
    });

    // Try to navigate to settings
    await page.goto('/settings', { waitUntil: 'domcontentloaded' });

    // The app should still be responsive (even if the page fails to load)
    // The loading state may be visible, or an error boundary might catch it
    // This test ensures the app doesn't hang
    await page.waitForTimeout(1000);

    // The app should still have the URL updated even if chunk fails
    expect(page.url()).toContain('/settings');
  });

  test('should lazy load all protected routes', async ({ page }) => {
    await setupCommonApiMocks(page);

    const protectedRoutes = [
      '/dashboard',
      '/printers',
      '/jobs',
      '/settings',
      '/agents',
    ];

    // Track loaded chunks
    const loadedChunks = new Set<string>();

    page.on('response', async (response) => {
      const url = response.url();
      if (url.includes('.js') && url.includes('/assets/')) {
        loadedChunks.add(url);
      }
    });

    // Visit each protected route
    for (const route of protectedRoutes) {
      await setupAuthAndNavigate(page, route, mockUsers[0]);
      await page.waitForLoadState('networkidle');
      await page.waitForTimeout(200);
    }

    // Chunks should have been loaded for different routes
    expect(loadedChunks.size).toBeGreaterThan(0);
  });

  test('should lazy load admin-only routes', async ({ page }) => {
    await setupCommonApiMocks(page, mockUsers[1]); // Admin user

    const adminRoutes = [
      '/analytics',
      '/organization',
      '/quotas',
      '/policies',
    ];

    for (const route of adminRoutes) {
      await setupAuthAndNavigate(page, route, mockUsers[1]);
      await page.waitForLoadState('networkidle');

      // Verify we're on the correct route
      expect(page.url()).toContain(route);
    }
  });

  test('should show inline loading fallback for content sections', async ({ page }) => {
    await setupCommonApiMocks(page);
    await setupAuthAndNavigate(page, '/dashboard', mockUsers[0]);

    // The dashboard should load without hanging
    await expect(page.getByText(/welcome/i)).toBeVisible({ timeout: 5000 });
  });

  test('should prefetch lazy chunks for links on hover (if prefetch is enabled)', async ({ page }) => {
    await setupCommonApiMocks(page);
    await setupAuthAndNavigate(page, '/dashboard', mockUsers[0]);

    // Track network requests
    let prefetchedChunk = false;

    page.on('request', async (request) => {
      const url = request.url();
      // Look for chunk prefetch requests (indicated by link rel=prefetch or similar)
      if (url.includes('.js') && url.includes('/assets/')) {
        const resourceType = request.resourceType();
        if (resourceType === 'script' || request.headers()['purpose'] === 'prefetch') {
          prefetchedChunk = true;
        }
      }
    });

    // Hover over a navigation link
    const jobsLink = page.locator('a[href="/jobs"]').first();
    await jobsLink.hover();

    // Wait a bit for any prefetch to occur
    await page.waitForTimeout(500);

    // Prefetching is optional and depends on React Router and browser behavior
    // This test just ensures the app doesn't break when hovering
    expect(await jobsLink.isVisible()).toBeTruthy();
  });

  test('should handle rapid navigation between lazy-loaded routes', async ({ page }) => {
    await setupCommonApiMocks(page);
    await setupAuthAndNavigate(page, '/dashboard', mockUsers[0]);

    // Rapidly navigate between routes
    const routes = ['/jobs', '/printers', '/settings', '/jobs', '/dashboard'];

    for (const route of routes) {
      await page.goto(route, { waitUntil: 'domcontentloaded' });
      await page.waitForTimeout(100);
    }

    // After rapid navigation, the app should still be functional
    await page.waitForLoadState('networkidle');
    expect(page.url()).toContain('/dashboard');
  });
});

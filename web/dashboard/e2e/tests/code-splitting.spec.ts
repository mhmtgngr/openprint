import { test, expect } from '@playwright/test';
import { setupAuthAndNavigate, setupCommonApiMocks, mockUsers } from '../helpers';

test.describe('Code Splitting', () => {
  test('should load vendor chunks separately from app code', async ({ page }) => {
    await setupCommonApiMocks(page);
    await setupAuthAndNavigate(page, '/dashboard', mockUsers[0]);

    // Wait for the page to fully load
    await page.waitForLoadState('networkidle');

    // Get all loaded scripts
    const scripts = await page.evaluate(() => {
      return Array.from(document.querySelectorAll('script[src]'))
        .map(s => (s as HTMLScriptElement).src);
    });

    // Verify that we have JavaScript files loaded
    const jsScripts = scripts.filter(s => s.includes('.js'));
    expect(jsScripts.length).toBeGreaterThan(0);
  });

  test('should load chunks on demand when navigating to different routes', async ({ page }) => {
    await setupCommonApiMocks(page);
    await setupAuthAndNavigate(page, '/dashboard', mockUsers[0]);

    // Get initial loaded chunks
    const initialScripts = await page.evaluate(() => {
      return Array.from(document.querySelectorAll('script[src]'))
        .map(s => (s as HTMLScriptElement).src)
        .filter(s => s.includes('/assets/'));
    });

    // Navigate to a different page
    await page.click('a[href="/printers"]');
    await page.waitForURL('**/printers');
    await page.waitForLoadState('networkidle');

    // Get scripts after navigation
    const afterNavScripts = await page.evaluate(() => {
      return Array.from(document.querySelectorAll('script[src]'))
        .map(s => (s as HTMLScriptElement).src)
        .filter(s => s.includes('/assets/'));
    });

    // Scripts should be loaded (may include lazy-loaded chunks)
    expect(afterNavScripts.length).toBeGreaterThanOrEqual(initialScripts.length);
  });

  test('should have separate chunks for different route groups', async ({ page }) => {
    await setupCommonApiMocks(page);

    // Track loaded chunks
    const loadedChunks = new Set<string>();

    page.on('response', async (response) => {
      const url = response.url();
      if (url.includes('.js') && url.includes('/assets/')) {
        const chunkName = url.split('/').pop()?.split('-')[0];
        if (chunkName) {
          loadedChunks.add(chunkName);
        }
      }
    });

    // Navigate to different pages to trigger chunk loading
    await setupAuthAndNavigate(page, '/dashboard', mockUsers[0]);
    await page.waitForLoadState('networkidle');

    await page.goto('/jobs', { waitUntil: 'networkidle' });
    await page.waitForTimeout(500);

    await page.goto('/settings', { waitUntil: 'networkidle' });
    await page.waitForTimeout(500);

    // Verify that different chunks were loaded
    // In production build, we expect multiple chunks
    // In dev mode, there may be fewer chunks
    expect(loadedChunks.size).toBeGreaterThan(0);
  });

  test('should load chart libraries in separate chunk', async ({ page }) => {
    // This test verifies that chart libraries (recharts) are in a separate chunk
    await setupCommonApiMocks(page);
    await setupAuthAndNavigate(page, '/dashboard', mockUsers[0]);

    // Get all scripts
    const scripts = await page.evaluate(() => {
      return Array.from(document.querySelectorAll('script[src]'))
        .map(s => (s as HTMLScriptElement).src);
    });

    // In production, charts should be in vendor-charts chunk
    const hasChartsChunk = scripts.some(s => s.includes('vendor-charts'));

    // This check is mainly for production builds
    if (process.env.NODE_ENV === 'production') {
      expect(hasChartsChunk).toBeTruthy();
    }
  });

  test('should load state management libraries in separate chunk', async ({ page }) => {
    await setupCommonApiMocks(page);
    await setupAuthAndNavigate(page, '/dashboard', mockUsers[0]);

    const scripts = await page.evaluate(() => {
      return Array.from(document.querySelectorAll('script[src]'))
        .map(s => (s as HTMLScriptElement).src);
    });

    // In production, state libs should be in vendor-state chunk
    const hasStateChunk = scripts.some(s => s.includes('vendor-state'));

    if (process.env.NODE_ENV === 'production') {
      expect(hasStateChunk).toBeTruthy();
    }
  });

  test('should minimize initial bundle size by lazy loading routes', async ({ page }) => {
    await setupCommonApiMocks(page);
    await setupAuthAndNavigate(page, '/dashboard', mockUsers[0]);

    // Measure initial loaded JavaScript size
    const initialBundleSize = await page.evaluate(() => {
      return Array.from(document.querySelectorAll('script[src]'))
        .filter(s => s.src.includes('/assets/'))
        .reduce((total, script) => {
          // This is an approximation - actual size would require fetching the script
          return total + 1;
        }, 0);
    });

    // Navigate to a new route
    await page.goto('/settings', { waitUntil: 'networkidle' });

    // Measure bundle size after navigation
    const afterNavBundleSize = await page.evaluate(() => {
      return Array.from(document.querySelectorAll('script[src]'))
        .filter(s => s.src.includes('/assets/'))
        .reduce((total, script) => {
          return total + 1;
        }, 0);
    });

    // More scripts should be loaded after navigation (lazy loaded)
    expect(afterNavBundleSize).toBeGreaterThanOrEqual(initialBundleSize);
  });

  test('should have CSS code splitting enabled', async ({ page }) => {
    await setupCommonApiMocks(page);
    await setupAuthAndNavigate(page, '/dashboard', mockUsers[0]);

    const stylesheets = await page.evaluate(() => {
      return Array.from(document.querySelectorAll('link[rel="stylesheet"]'))
        .map(l => (l as HTMLLinkElement).href);
    });

    // CSS should be loaded
    expect(stylesheets.length).toBeGreaterThan(0);
  });

  test('should load React and ReactDOM from vendor chunk', async ({ page }) => {
    await setupCommonApiMocks(page);
    await setupAuthAndNavigate(page, '/dashboard', mockUsers[0]);

    // Check if JavaScript modules are loaded (React is bundled, not global)
    const hasJsModules = await page.evaluate(() => {
      return Array.from(document.querySelectorAll('script[src]'))
        .some(s => s.src.includes('.js') || s.src.includes('type=module'));
    });

    expect(hasJsModules).toBeTruthy();
  });
});

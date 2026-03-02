/**
 * Production Environment Configuration for E2E Tests (Smoke Tests Only)
 */
import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: 0,
  workers: 1, // Run serially to avoid rate limiting
  reporter: [
    ['html', { outputFolder: 'playwright-report' }],
    ['list'],
    ['github'],
  ],
  use: {
    baseURL: process.env.PRODUCTION_URL || 'https://openprint.cloud',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
  },
  projects: [
    {
      name: 'chromium-smoke',
      use: { ...devices['Desktop Chrome'] },
      testMatch: /.*\.spec\.ts/,
      grep: /@smoke|@critical/,
    },
  ],
});

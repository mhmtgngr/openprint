/**
 * Global setup for Playwright E2E tests
 * This runs once before all tests
 */
import { FullConfig } from '@playwright/test';

async function globalSetup(config: FullConfig) {
  // Any global setup can go here
  // For example, starting a test database, generating test data, etc.

  console.log('🚀 Playwright E2E Test Suite Starting...');
  console.log(`📁 Test directory: ${config.projects?.[0]?.testDir || 'e2e'}`);
  console.log(`🌐 Base URL: ${config.projects?.[0]?.use?.baseURL || 'http://localhost:3000'}`);

  // You can also set environment variables here
  // process.env.TEST_MODE = 'e2e';
}

export default globalSetup;

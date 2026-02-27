import { FullConfig } from '@playwright/test';

async function globalSetup(config: FullConfig) {
  // Global setup before all tests run
  // This could include:
  // - Starting a mock server
  // - Setting up test database
  // - Generating test tokens
  console.log('Starting E2E test run...');
}

export default globalSetup;

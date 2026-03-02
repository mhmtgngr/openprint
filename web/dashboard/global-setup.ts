import { FullConfig } from '@playwright/test';

async function globalSetup(config: FullConfig) {
  // Note: baseURL cannot be modified in globalSetup
  // Use environment variable BASE_URL in playwright.config.ts instead
  if (process.env.BASE_URL) {
    console.log(`BASE_URL environment variable is set: ${process.env.BASE_URL}`);
    console.log('Note: Update playwright.config.ts to use this URL');
  }
  console.log('Starting E2E test run...');
}

export default globalSetup;

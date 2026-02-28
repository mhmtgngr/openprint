import { FullConfig } from '@playwright/test';

async function globalSetup(config: FullConfig) {
  // Override baseURL from environment variable (for testing against deployed instances)
  if (process.env.BASE_URL) {
    config.use!.baseURL = process.env.BASE_URL;
    console.log(`Using baseURL from environment: ${process.env.BASE_URL}`);
  }
  console.log('Starting E2E test run...');
}

export default globalSetup;

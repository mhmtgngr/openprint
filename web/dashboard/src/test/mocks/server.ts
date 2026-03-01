/**
 * MSW Server Setup
 * Configures the Mock Service Worker for request interception in tests
 */

import { setupServer } from 'msw/node';
import { handlers } from './handlers';

// Create MSW server with all handlers
export const server = setupServer(...handlers);

// Server lifecycle hooks for Vitest
export const setupMSW = () => {
  // Listen to requests before all tests
  server.listen({ onUnhandledRequest: 'warn' });

  // Reset handlers after each test
  afterEach(() => {
    server.resetHandlers();
  });
};

// Close server after all tests
export const teardownMSW = () => {
  server.close();
};

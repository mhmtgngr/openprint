/**
 * Custom Test Utilities
 * Provides render functions with providers, custom matchers, and test data factories
 */

import React, { ReactElement } from 'react';
import { render, RenderOptions } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter, MemoryRouter, Routes, Route } from 'react-router-dom';
import userEvent from '@testing-library/user-event';

// ============================================================================
// Test Data Factories
// ============================================================================

export const createMockJob = (overrides = {}) => ({
  id: `job-${Math.random().toString(36).substr(2, 9)}`,
  documentName: 'Test Document.pdf',
  status: 'queued',
  pageCount: 5,
  colorPages: 0,
  fileSize: 1024000,
  createdAt: new Date().toISOString(),
  printer: null,
  errorMessage: null,
  ...overrides,
});

export const createMockJobs = (count: number, status?: string) =>
  Array.from({ length: count }, (_, i) =>
    createMockJob({
      id: `job-${i}`,
      documentName: `Document ${i + 1}.pdf`,
      status: status || (['queued', 'processing', 'completed', 'failed', 'cancelled'] as const)[
        i % 5
      ],
    })
  );

export const createMockAgent = (overrides = {}) => ({
  id: `agent-${Math.random().toString(36).substr(2, 9)}`,
  name: 'Test Agent',
  platform: 'linux',
  agentVersion: '1.0.0',
  status: 'online',
  createdAt: new Date().toISOString(),
  lastSeen: new Date().toISOString(),
  printerCount: 2,
  ...overrides,
});

export const createMockPrinter = (overrides = {}) => ({
  id: `printer-${Math.random().toString(36).substr(2, 9)}`,
  name: 'Test Printer',
  agentName: 'Test Agent',
  agentId: 'agent-1',
  isOnline: true,
  isActive: true,
  capabilities: {
    color: true,
    duplex: true,
    paperSizes: ['A4', 'Letter'],
  },
  createdAt: new Date().toISOString(),
  lastSeen: new Date().toISOString(),
  ...overrides,
});

export const createMockDocument = (overrides = {}) => ({
  id: `doc-${Math.random().toString(36).substr(2, 9)}`,
  name: 'Test Document.pdf',
  size: 1024000,
  mimeType: 'application/pdf',
  isEncrypted: false,
  createdAt: new Date().toISOString(),
  updatedAt: new Date().toISOString(),
  ownerEmail: 'test@example.com',
  ...overrides,
});

export const createMockActivity = (overrides = {}) => ({
  id: `activity-${Math.random().toString(36).substr(2, 9)}`,
  type: 'job_completed',
  timestamp: new Date().toISOString(),
  jobName: 'Test Job',
  printerName: 'Test Printer',
  userName: 'Test User',
  details: 'Job completed successfully',
  ...overrides,
});

export const createMockAnalytics = (days = 7) =>
  Array.from({ length: days }, (_, i) => {
    const date = new Date();
    date.setDate(date.getDate() - i);
    return {
      statDate: date.toISOString().split('T')[0],
      jobsCount: Math.floor(Math.random() * 50) + 20,
      jobsCompleted: Math.floor(Math.random() * 45) + 18,
      jobsFailed: Math.floor(Math.random() * 5),
      pagesPrinted: Math.floor(Math.random() * 500) + 200,
    };
  });

// ============================================================================
// Custom Render Functions
// ============================================================================

interface AllTheProvidersProps {
  children: React.ReactNode;
  queryClient?: QueryClient;
  router?: typeof BrowserRouter | typeof MemoryRouter;
  routerProps?: {
    initialEntries?: string[];
    initialIndex?: number;
  };
}

const createTestQueryClient = () =>
  new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        gcTime: 0,
      },
      mutations: {
        retry: false,
      },
    },
  });

const AllTheProviders = ({ children, queryClient, router: Router = BrowserRouter, routerProps }: AllTheProvidersProps) => {
  const testQueryClient = queryClient || createTestQueryClient();

  if (Router === MemoryRouter) {
    return (
      <QueryClientProvider client={testQueryClient}>
        <MemoryRouter {...routerProps}>
          {children}
        </MemoryRouter>
      </QueryClientProvider>
    );
  }

  return (
    <QueryClientProvider client={testQueryClient}>
      <Router>
        {children}
      </Router>
    </QueryClientProvider>
  );
};

interface CustomRenderOptions extends Omit<RenderOptions, 'wrapper'> {
  queryClient?: QueryClient;
  router?: typeof BrowserRouter | typeof MemoryRouter;
  routerProps?: {
    initialEntries?: string[];
    initialIndex?: number;
  };
}

const customRender = (ui: ReactElement, options: CustomRenderOptions = {}) => {
  const { queryClient, router, routerProps, ...renderOptions } = options;

  return {
    user: userEvent.setup(),
    ...render(ui, {
      wrapper: ({ children }) => (
        <AllTheProviders
          queryClient={queryClient}
          router={router}
          routerProps={routerProps}
        >
          {children}
        </AllTheProviders>
      ),
      ...renderOptions,
    }),
  };
};

// Render with just QueryClient provider
const renderWithQueryClient = (ui: ReactElement, options: CustomRenderOptions = {}) => {
  const { queryClient, ...renderOptions } = options;
  const testQueryClient = queryClient || createTestQueryClient();

  return {
    user: userEvent.setup(),
    ...render(ui, {
      wrapper: ({ children }) => (
        <QueryClientProvider client={testQueryClient}>
          {children}
        </QueryClientProvider>
      ),
      ...renderOptions,
    }),
  };
};

// Render with router for testing navigation
const renderWithRouter = (
  ui: ReactElement,
  routes: Record<string, ReactElement> = {},
  initialEntries: string[] = ['/']
) => {
  const testQueryClient = createTestQueryClient();

  return {
    user: userEvent.setup(),
    ...render(
      <QueryClientProvider client={testQueryClient}>
        <MemoryRouter initialEntries={initialEntries}>
          <Routes>
            {Object.entries(routes).map(([path, element]) => (
              <Route key={path} path={path} element={element} />
            ))}
            <Route path="*" element={ui} />
          </Routes>
        </MemoryRouter>
      </QueryClientProvider>
    ),
  };
};

// ============================================================================
// Re-exports
// ============================================================================

export * from '@testing-library/react';
export { customRender as render, renderWithQueryClient, renderWithRouter };
export { default as userEvent } from '@testing-library/user-event';
export { createTestQueryClient };

// ============================================================================
// Helper Functions
// ============================================================================

/**
 * Wait for all pending React Query promises to resolve
 */
export const waitForQueryUpdates = async () => {
  return new Promise<void>((resolve) => {
    setTimeout(() => resolve(), 0);
  });
};

/**
 * Mock window.location
 */
export const mockLocation = (href: string) => {
  delete (window as any).location;
  (window as any).location = new URL(href);
};

/**
 * Create a mock function that returns a value after a delay
 */
export const createAsyncMock = <T extends (...args: any[]) => any>(
  fn: T,
  delay = 100
): T => {
  return ((...args: Parameters<T>) =>
    new Promise<Awaited<ReturnType<T>>>((resolve) => {
      setTimeout(() => {
        resolve(fn(...args) as Awaited<ReturnType<T>>);
      }, delay);
    })) as T;
};

/**
 * Suppress console errors in a test block
 */
export const suppressConsoleErrors = () => {
  const originalError = console.error;
  beforeEach(() => {
    console.error = vi.fn();
  });
  afterEach(() => {
    console.error = originalError;
  });
};

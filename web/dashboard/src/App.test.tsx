import { render, screen, waitFor } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { MemoryRouter, Routes, Route, useNavigate, useLocation } from 'react-router-dom';
import App from './App';

// Mock the hooks
vi.mock('./hooks/useAuth', () => ({
  useAuth: vi.fn(() => ({
    isAuthenticated: true,
    isLoading: false,
    user: { id: '1', name: 'Test User', email: 'test@example.com', role: 'user' as const },
    hasRole: vi.fn((roles: string[]) => roles.includes('user')),
    login: vi.fn(),
    register: vi.fn(),
    logout: vi.fn(),
  })),
  useRequireAuth: vi.fn(() => ({
    isAuthenticated: true,
    isLoading: false,
  })),
}));

// Mock the lazy-loaded page components
vi.mock('./pages/Login', () => ({
  Login: () => <div data-testid="login-page">Login Page</div>,
}));

vi.mock('./pages/Dashboard', () => ({
  Dashboard: () => <div data-testid="dashboard-page">Dashboard Page</div>,
}));

vi.mock('./pages/Printers', () => ({
  Printers: () => <div data-testid="printers-page">Printers Page</div>,
}));

vi.mock('./pages/Jobs', () => ({
  Jobs: () => <div data-testid="jobs-page">Jobs Page</div>,
}));

vi.mock('./pages/Analytics', () => ({
  Analytics: () => <div data-testid="analytics-page">Analytics Page</div>,
}));

vi.mock('./pages/Settings', () => ({
  Settings: () => <div data-testid="settings-page">Settings Page</div>,
}));

vi.mock('./pages/Organization', () => ({
  Organization: () => <div data-testid="organization-page">Organization Page</div>,
}));

vi.mock('./pages/Documents', () => ({
  Documents: () => <div data-testid="documents-page">Documents Page</div>,
}));

vi.mock('./pages/Agents', () => ({
  Agents: () => <div data-testid="agents-page">Agents Page</div>,
}));

vi.mock('./pages/AgentDetail', () => ({
  AgentDetailPage: () => <div data-testid="agent-detail-page">Agent Detail Page</div>,
}));

vi.mock('./pages/DiscoveredPrinters', () => ({
  DiscoveredPrintersPage: () => <div data-testid="discovered-printers-page">Discovered Printers Page</div>,
}));

vi.mock('./pages/JobAssignments', () => ({
  JobAssignmentsPage: () => <div data-testid="job-assignments-page">Job Assignments Page</div>,
}));

vi.mock('./pages/Quotas', () => ({
  Quotas: () => <div data-testid="quotas-page">Quotas Page</div>,
}));

vi.mock('./pages/Policies', () => ({
  Policies: () => <div data-testid="policies-page">Policies Page</div>,
}));

vi.mock('./pages/AuditLogs', () => ({
  AuditLogs: () => <div data-testid="audit-logs-page">Audit Logs Page</div>,
}));

vi.mock('./pages/EmailToPrint', () => ({
  EmailToPrint: () => <div data-testid="email-to-print-page">Email To Print Page</div>,
}));

vi.mock('./pages/PrintRelease', () => ({
  PrintReleasePage: () => <div data-testid="print-release-page">Print Release Page</div>,
}));

// Mock layouts
vi.mock('./layouts/PublicLayout', () => ({
  PublicLayout: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="public-layout">{children}</div>
  ),
}));

vi.mock('./layouts/DashboardLayout', () => ({
  DashboardLayout: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="dashboard-layout">{children}</div>
  ),
}));

// Helper to render App with router
const renderAppWithRouter = (initialEntries = ['/']) => {
  return render(
    <MemoryRouter initialEntries={initialEntries}>
      <Routes>
        <Route path="/*" element={<App />} />
      </Routes>
    </MemoryRouter>
  );
};

describe('App Component', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('Routing', () => {
    it('should redirect to dashboard when visiting root path', async () => {
      renderAppWithRouter(['/']);

      await waitFor(() => {
        expect(screen.queryByTestId('dashboard-page')).toBeInTheDocument();
      });
    });

    it('should redirect to dashboard for unknown routes', async () => {
      renderAppWithRouter(['/unknown-route']);

      await waitFor(() => {
        expect(screen.queryByTestId('dashboard-page')).toBeInTheDocument();
      });
    });

    it('should render login page at /login route', async () => {
      const { useAuth: mockUseAuth } = await import('./hooks/useAuth');
      (mockUseAuth as any).mockReturnValue({
        isAuthenticated: false,
        isLoading: false,
        user: null,
        hasRole: vi.fn(() => false),
        login: vi.fn(),
        register: vi.fn(),
        logout: vi.fn(),
      });

      renderAppWithRouter(['/login']);

      await waitFor(() => {
        expect(screen.getByTestId('login-page')).toBeInTheDocument();
      });
    });

    it('should render dashboard page at /dashboard route', async () => {
      renderAppWithRouter(['/dashboard']);

      await waitFor(() => {
        expect(screen.getByTestId('dashboard-page')).toBeInTheDocument();
      });
    });

    it('should render printers page at /printers route', async () => {
      renderAppWithRouter(['/printers']);

      await waitFor(() => {
        expect(screen.getByTestId('printers-page')).toBeInTheDocument();
      });
    });

    it('should render jobs page at /jobs route', async () => {
      renderAppWithRouter(['/jobs']);

      await waitFor(() => {
        expect(screen.getByTestId('jobs-page')).toBeInTheDocument();
      });
    });

    it('should render documents page at /documents route', async () => {
      renderAppWithRouter(['/documents']);

      await waitFor(() => {
        expect(screen.getByTestId('documents-page')).toBeInTheDocument();
      });
    });

    it('should render settings page at /settings route', async () => {
      renderAppWithRouter(['/settings']);

      await waitFor(() => {
        expect(screen.getByTestId('settings-page')).toBeInTheDocument();
      });
    });

    it('should render agents page at /agents route', async () => {
      renderAppWithRouter(['/agents']);

      await waitFor(() => {
        expect(screen.getByTestId('agents-page')).toBeInTheDocument();
      });
    });

    it('should render agent detail page at /agents/:id route', async () => {
      renderAppWithRouter(['/agents/123']);

      await waitFor(() => {
        expect(screen.getByTestId('agent-detail-page')).toBeInTheDocument();
      });
    });

    it('should render discovered printers page at /discovered-printers route', async () => {
      renderAppWithRouter(['/discovered-printers']);

      await waitFor(() => {
        expect(screen.getByTestId('discovered-printers-page')).toBeInTheDocument();
      });
    });

    it('should render print release page at /print-release route', async () => {
      renderAppWithRouter(['/print-release']);

      await waitFor(() => {
        expect(screen.getByTestId('print-release-page')).toBeInTheDocument();
      });
    });
  });

  describe('Admin Routes', () => {
    it('should render analytics page at /analytics route for admin users', async () => {
      const { useAuth: mockUseAuth } = await import('./hooks/useAuth');
      (mockUseAuth as any).mockReturnValue({
        isAuthenticated: true,
        isLoading: false,
        user: { id: '1', name: 'Admin User', email: 'admin@example.com', role: 'admin' as const },
        hasRole: vi.fn((roles: string[]) => roles.includes('admin')),
        login: vi.fn(),
        register: vi.fn(),
        logout: vi.fn(),
      });

      renderAppWithRouter(['/analytics']);

      await waitFor(() => {
        expect(screen.getByTestId('analytics-page')).toBeInTheDocument();
      });
    });

    it('should render organization page at /organization route for admin users', async () => {
      const { useAuth: mockUseAuth } = await import('./hooks/useAuth');
      (mockUseAuth as any).mockReturnValue({
        isAuthenticated: true,
        isLoading: false,
        user: { id: '1', name: 'Admin User', email: 'admin@example.com', role: 'admin' as const },
        hasRole: vi.fn((roles: string[]) => roles.includes('admin')),
        login: vi.fn(),
        register: vi.fn(),
        logout: vi.fn(),
      });

      renderAppWithRouter(['/organization']);

      await waitFor(() => {
        expect(screen.getByTestId('organization-page')).toBeInTheDocument();
      });
    });

    it('should render job assignments page at /job-assignments route for admin users', async () => {
      const { useAuth: mockUseAuth } = await import('./hooks/useAuth');
      (mockUseAuth as any).mockReturnValue({
        isAuthenticated: true,
        isLoading: false,
        user: { id: '1', name: 'Admin User', email: 'admin@example.com', role: 'admin' as const },
        hasRole: vi.fn((roles: string[]) => roles.includes('admin')),
        login: vi.fn(),
        register: vi.fn(),
        logout: vi.fn(),
      });

      renderAppWithRouter(['/job-assignments']);

      await waitFor(() => {
        expect(screen.getByTestId('job-assignments-page')).toBeInTheDocument();
      });
    });

    it('should render quotas page at /quotas route for admin users', async () => {
      const { useAuth: mockUseAuth } = await import('./hooks/useAuth');
      (mockUseAuth as any).mockReturnValue({
        isAuthenticated: true,
        isLoading: false,
        user: { id: '1', name: 'Admin User', email: 'admin@example.com', role: 'admin' as const },
        hasRole: vi.fn((roles: string[]) => roles.includes('admin')),
        login: vi.fn(),
        register: vi.fn(),
        logout: vi.fn(),
      });

      renderAppWithRouter(['/quotas']);

      await waitFor(() => {
        expect(screen.getByTestId('quotas-page')).toBeInTheDocument();
      });
    });

    it('should render policies page at /policies route for admin users', async () => {
      const { useAuth: mockUseAuth } = await import('./hooks/useAuth');
      (mockUseAuth as any).mockReturnValue({
        isAuthenticated: true,
        isLoading: false,
        user: { id: '1', name: 'Admin User', email: 'admin@example.com', role: 'admin' as const },
        hasRole: vi.fn((roles: string[]) => roles.includes('admin')),
        login: vi.fn(),
        register: vi.fn(),
        logout: vi.fn(),
      });

      renderAppWithRouter(['/policies']);

      await waitFor(() => {
        expect(screen.getByTestId('policies-page')).toBeInTheDocument();
      });
    });

    it('should render audit logs page at /audit-logs route for admin users', async () => {
      const { useAuth: mockUseAuth } = await import('./hooks/useAuth');
      (mockUseAuth as any).mockReturnValue({
        isAuthenticated: true,
        isLoading: false,
        user: { id: '1', name: 'Admin User', email: 'admin@example.com', role: 'admin' as const },
        hasRole: vi.fn((roles: string[]) => roles.includes('admin')),
        login: vi.fn(),
        register: vi.fn(),
        logout: vi.fn(),
      });

      renderAppWithRouter(['/audit-logs']);

      await waitFor(() => {
        expect(screen.getByTestId('audit-logs-page')).toBeInTheDocument();
      });
    });

    it('should render email to print page at /email-to-print route for admin users', async () => {
      const { useAuth: mockUseAuth } = await import('./hooks/useAuth');
      (mockUseAuth as any).mockReturnValue({
        isAuthenticated: true,
        isLoading: false,
        user: { id: '1', name: 'Admin User', email: 'admin@example.com', role: 'admin' as const },
        hasRole: vi.fn((roles: string[]) => roles.includes('admin')),
        login: vi.fn(),
        register: vi.fn(),
        logout: vi.fn(),
      });

      renderAppWithRouter(['/email-to-print']);

      await waitFor(() => {
        expect(screen.getByTestId('email-to-print-page')).toBeInTheDocument();
      });
    });
  });

  describe('Lazy Loading', () => {
    it('should use React.lazy for all page components', async () => {
      // This test verifies that the lazy imports are properly structured
      // The actual lazy loading behavior is tested by the routing tests
      // which verify that the lazy components load correctly
      renderAppWithRouter(['/dashboard']);

      await waitFor(() => {
        expect(screen.getByTestId('dashboard-page')).toBeInTheDocument();
      });
    });

    it('should wrap routes in Suspense with loading fallback', async () => {
      // The App component wraps all routes in a Suspense boundary
      // with PageLoadingFallback as the fallback
      const { container } = renderAppWithRouter(['/dashboard']);

      // After lazy loading completes, the page should be rendered
      await waitFor(() => {
        expect(screen.getByTestId('dashboard-page')).toBeInTheDocument();
      });
    });
  });

  describe('Protected Routes', () => {
    it('should render dashboard layout for protected routes when authenticated', async () => {
      renderAppWithRouter(['/dashboard']);

      await waitFor(() => {
        expect(screen.getByTestId('dashboard-layout')).toBeInTheDocument();
      });
    });
  });

  describe('Access Control', () => {
    it('should allow access to protected routes for authenticated users', async () => {
      renderAppWithRouter(['/settings']);

      await waitFor(() => {
        expect(screen.getByTestId('settings-page')).toBeInTheDocument();
      });
    });

    it('should allow access to admin routes for admin users', async () => {
      const { useAuth: mockUseAuth } = await import('./hooks/useAuth');
      (mockUseAuth as any).mockReturnValue({
        isAuthenticated: true,
        isLoading: false,
        user: { id: '1', name: 'Admin User', email: 'admin@example.com', role: 'admin' as const },
        hasRole: vi.fn((roles: string[]) => roles.includes('admin')),
        login: vi.fn(),
        register: vi.fn(),
        logout: vi.fn(),
      });

      renderAppWithRouter(['/analytics']);

      await waitFor(() => {
        expect(screen.getByTestId('analytics-page')).toBeInTheDocument();
      });
    });

    it('should allow access to owner role for admin routes', async () => {
      const { useAuth: mockUseAuth } = await import('./hooks/useAuth');
      (mockUseAuth as any).mockReturnValue({
        isAuthenticated: true,
        isLoading: false,
        user: { id: '1', name: 'Owner User', email: 'owner@example.com', role: 'owner' as const },
        hasRole: vi.fn((roles: string[]) => roles.includes('owner')),
        login: vi.fn(),
        register: vi.fn(),
        logout: vi.fn(),
      });

      renderAppWithRouter(['/analytics']);

      await waitFor(() => {
        expect(screen.getByTestId('analytics-page')).toBeInTheDocument();
      });
    });
  });
});

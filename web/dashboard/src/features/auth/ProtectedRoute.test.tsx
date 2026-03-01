/**
 * ProtectedRoute Component Tests
 * Tests for route protection and authentication checks
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@/test/utils/test-utils';
import { ProtectedRoute, AdminRoute } from './ProtectedRoute';
import * as useAuthHook from '@/hooks/useAuth';

// Mock the useAuth hook
vi.mock('@/hooks/useAuth');

describe('ProtectedRoute', () => {
  const mockUseAuth = vi.spyOn(useAuthHook, 'useAuth');

  beforeEach(() => {
    mockUseAuth.mockClear();
  });

  describe('Loading State', () => {
    it('should show loading spinner when isLoading is true', () => {
      mockUseAuth.mockReturnValue({
        isAuthenticated: false,
        isLoading: true,
        user: null,
        error: null,
        login: vi.fn(),
        register: vi.fn(),
        logout: vi.fn(),
        hasRole: vi.fn(() => false),
      });

      render(
        <ProtectedRoute>
          <div>Protected Content</div>
        </ProtectedRoute>
      );

      expect(screen.getByText(/loading/i)).toBeInTheDocument();
    });

    it('should not render children while loading', () => {
      mockUseAuth.mockReturnValue({
        isAuthenticated: false,
        isLoading: true,
        user: null,
        error: null,
        login: vi.fn(),
        register: vi.fn(),
        logout: vi.fn(),
        hasRole: vi.fn(() => false),
      });

      render(
        <ProtectedRoute>
          <div>Protected Content</div>
        </ProtectedRoute>
      );

      expect(screen.queryByText('Protected Content')).not.toBeInTheDocument();
    });

    it('should render loading with proper styling', () => {
      mockUseAuth.mockReturnValue({
        isAuthenticated: false,
        isLoading: true,
        user: null,
        error: null,
        login: vi.fn(),
        register: vi.fn(),
        logout: vi.fn(),
        hasRole: vi.fn(() => false),
      });

      render(
        <ProtectedRoute>
          <div>Protected Content</div>
        </ProtectedRoute>
      );

      const container = screen.getByText(/loading/i).closest('div');
      expect(container).toHaveClass(/flex items-center justify-center/);
    });
  });

  describe('Authentication Check', () => {
    it('should redirect to login when not authenticated', () => {
      mockUseAuth.mockReturnValue({
        isAuthenticated: false,
        isLoading: false,
        user: null,
        error: null,
        login: vi.fn(),
        register: vi.fn(),
        logout: vi.fn(),
        hasRole: vi.fn(() => false),
      });

      render(
        <ProtectedRoute>
          <div>Protected Content</div>
        </ProtectedRoute>
      );

      // Navigate component should be rendered (redirecting)
      expect(screen.queryByText('Protected Content')).not.toBeInTheDocument();
    });

    it('should render children when authenticated', () => {
      mockUseAuth.mockReturnValue({
        isAuthenticated: true,
        isLoading: false,
        user: {
          id: '1',
          email: 'test@example.com',
          name: 'Test User',
          role: 'user',
          orgId: 'org-1',
          isActive: true,
          emailVerified: true,
          createdAt: '2025-01-01T00:00:00Z',
        },
        error: null,
        login: vi.fn(),
        register: vi.fn(),
        logout: vi.fn(),
        hasRole: vi.fn(() => true),
      });

      render(
        <ProtectedRoute>
          <div>Protected Content</div>
        </ProtectedRoute>
      );

      expect(screen.getByText('Protected Content')).toBeInTheDocument();
    });

    it('should use custom redirectTo when specified', () => {
      mockUseAuth.mockReturnValue({
        isAuthenticated: false,
        isLoading: false,
        user: null,
        error: null,
        login: vi.fn(),
        register: vi.fn(),
        logout: vi.fn(),
        hasRole: vi.fn(() => false),
      });

      render(
        <ProtectedRoute redirectTo="/custom-login">
          <div>Protected Content</div>
        </ProtectedRoute>
      );

      expect(screen.queryByText('Protected Content')).not.toBeInTheDocument();
    });
  });

  describe('Role-Based Access', () => {
    it('should allow access when user has required role', () => {
      mockUseAuth.mockReturnValue({
        isAuthenticated: true,
        isLoading: false,
        user: {
          id: '1',
          email: 'admin@example.com',
          name: 'Admin',
          role: 'admin',
          orgId: 'org-1',
          isActive: true,
          emailVerified: true,
          createdAt: '2025-01-01T00:00:00Z',
        },
        error: null,
        login: vi.fn(),
        register: vi.fn(),
        logout: vi.fn(),
        hasRole: vi.fn((roles) => roles.includes('admin')),
      });

      render(
        <ProtectedRoute requiredRoles={['admin']}>
          <div>Admin Content</div>
        </ProtectedRoute>
      );

      expect(screen.getByText('Admin Content')).toBeInTheDocument();
    });

    it('should deny access when user lacks required role', () => {
      mockUseAuth.mockReturnValue({
        isAuthenticated: true,
        isLoading: false,
        user: {
          id: '1',
          email: 'user@example.com',
          name: 'User',
          role: 'user',
          orgId: 'org-1',
          isActive: true,
          emailVerified: true,
          createdAt: '2025-01-01T00:00:00Z',
        },
        error: null,
        login: vi.fn(),
        register: vi.fn(),
        logout: vi.fn(),
        hasRole: vi.fn((roles) => roles.includes('admin')),
      });

      render(
        <ProtectedRoute requiredRoles={['admin']}>
          <div>Admin Content</div>
        </ProtectedRoute>
      );

      expect(screen.queryByText('Admin Content')).not.toBeInTheDocument();
    });

    it('should check hasRole with correct role array', () => {
      const hasRoleMock = vi.fn();
      mockUseAuth.mockReturnValue({
        isAuthenticated: true,
        isLoading: false,
        user: {
          id: '1',
          email: 'user@example.com',
          name: 'User',
          role: 'user',
          orgId: 'org-1',
          isActive: true,
          emailVerified: true,
          createdAt: '2025-01-01T00:00:00Z',
        },
        error: null,
        login: vi.fn(),
        register: vi.fn(),
        logout: vi.fn(),
        hasRole: hasRoleMock,
      });

      render(
        <ProtectedRoute requiredRoles={['admin', 'owner']}>
          <div>Protected Content</div>
        </ProtectedRoute>
      );

      expect(hasRoleMock).toHaveBeenCalledWith(['admin', 'owner']);
    });

    it('should allow access when user has one of multiple required roles', () => {
      mockUseAuth.mockReturnValue({
        isAuthenticated: true,
        isLoading: false,
        user: {
          id: '1',
          email: 'owner@example.com',
          name: 'Owner',
          role: 'owner',
          orgId: 'org-1',
          isActive: true,
          emailVerified: true,
          createdAt: '2025-01-01T00:00:00Z',
        },
        error: null,
        login: vi.fn(),
        register: vi.fn(),
        logout: vi.fn(),
        hasRole: vi.fn((roles) => roles.some((r: string) => ['admin', 'owner'].includes(r))),
      });

      render(
        <ProtectedRoute requiredRoles={['admin', 'owner']}>
          <div>Owner Content</div>
        </ProtectedRoute>
      );

      expect(screen.getByText('Owner Content')).toBeInTheDocument();
    });

    it('should not check roles when requiredRoles is not provided', () => {
      const hasRoleMock = vi.fn();
      mockUseAuth.mockReturnValue({
        isAuthenticated: true,
        isLoading: false,
        user: {
          id: '1',
          email: 'user@example.com',
          name: 'User',
          role: 'user',
          orgId: 'org-1',
          isActive: true,
          emailVerified: true,
          createdAt: '2025-01-01T00:00:00Z',
        },
        error: null,
        login: vi.fn(),
        register: vi.fn(),
        logout: vi.fn(),
        hasRole: hasRoleMock,
      });

      render(
        <ProtectedRoute>
          <div>Content</div>
        </ProtectedRoute>
      );

      expect(hasRoleMock).not.toHaveBeenCalled();
      expect(screen.getByText('Content')).toBeInTheDocument();
    });

    it('should not check roles when requiredRoles is empty array', () => {
      const hasRoleMock = vi.fn();
      mockUseAuth.mockReturnValue({
        isAuthenticated: true,
        isLoading: false,
        user: {
          id: '1',
          email: 'user@example.com',
          name: 'User',
          role: 'user',
          orgId: 'org-1',
          isActive: true,
          emailVerified: true,
          createdAt: '2025-01-01T00:00:00Z',
        },
        error: null,
        login: vi.fn(),
        register: vi.fn(),
        logout: vi.fn(),
        hasRole: hasRoleMock,
      });

      render(
        <ProtectedRoute requiredRoles={[]}>
          <div>Content</div>
        </ProtectedRoute>
      );

      expect(hasRoleMock).not.toHaveBeenCalled();
      expect(screen.getByText('Content')).toBeInTheDocument();
    });
  });

  describe('AdminRoute Convenience Wrapper', () => {
    it('should render children for admin users', () => {
      mockUseAuth.mockReturnValue({
        isAuthenticated: true,
        isLoading: false,
        user: {
          id: '1',
          email: 'admin@example.com',
          name: 'Admin',
          role: 'admin',
          orgId: 'org-1',
          isActive: true,
          emailVerified: true,
          createdAt: '2025-01-01T00:00:00Z',
        },
        error: null,
        login: vi.fn(),
        register: vi.fn(),
        logout: vi.fn(),
        hasRole: vi.fn((roles) => roles.includes('admin')),
      });

      render(
        <AdminRoute>
          <div>Admin Only Content</div>
        </AdminRoute>
      );

      expect(screen.getByText('Admin Only Content')).toBeInTheDocument();
    });

    it('should deny access for non-admin users', () => {
      mockUseAuth.mockReturnValue({
        isAuthenticated: true,
        isLoading: false,
        user: {
          id: '1',
          email: 'user@example.com',
          name: 'User',
          role: 'user',
          orgId: 'org-1',
          isActive: true,
          emailVerified: true,
          createdAt: '2025-01-01T00:00:00Z',
        },
        error: null,
        login: vi.fn(),
        register: vi.fn(),
        logout: vi.fn(),
        hasRole: vi.fn((roles) => roles.includes('admin') || roles.includes('owner')),
      });

      render(
        <AdminRoute>
          <div>Admin Only Content</div>
        </AdminRoute>
      );

      expect(screen.queryByText('Admin Only Content')).not.toBeInTheDocument();
    });

    it('should allow access for owner users', () => {
      mockUseAuth.mockReturnValue({
        isAuthenticated: true,
        isLoading: false,
        user: {
          id: '1',
          email: 'owner@example.com',
          name: 'Owner',
          role: 'owner',
          orgId: 'org-1',
          isActive: true,
          emailVerified: true,
          createdAt: '2025-01-01T00:00:00Z',
        },
        error: null,
        login: vi.fn(),
        register: vi.fn(),
        logout: vi.fn(),
        hasRole: vi.fn((roles) => roles.includes('owner')),
      });

      render(
        <AdminRoute>
          <div>Owner Content</div>
        </AdminRoute>
      );

      expect(screen.getByText('Owner Content')).toBeInTheDocument();
    });
  });
});

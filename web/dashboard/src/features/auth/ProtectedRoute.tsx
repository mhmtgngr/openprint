import { useEffect } from 'react';
import { Navigate } from 'react-router-dom';
import { useAuth } from '@/hooks/useAuth';
import type { ProtectedRouteProps } from '@/types/auth';

export const ProtectedRoute = ({
  children,
  redirectTo = '/login',
  requiredRoles,
}: ProtectedRouteProps) => {
  const { isAuthenticated, isLoading, hasRole } = useAuth();

  useEffect(() => {
    // Redirect if not authenticated after loading completes
    if (!isLoading && !isAuthenticated) {
      // Navigate will be handled by the Navigate component below
    }
  }, [isAuthenticated, isLoading]);

  // Show loading spinner while checking auth
  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
        <div className="flex flex-col items-center gap-4">
          <div className="w-12 h-12 border-4 border-blue-600 border-t-transparent rounded-full animate-spin" />
          <p className="text-gray-600 dark:text-gray-400">Loading...</p>
        </div>
      </div>
    );
  }

  // Redirect to login if not authenticated
  if (!isAuthenticated) {
    return <Navigate to={redirectTo} replace />;
  }

  // Check role requirements if specified
  if (requiredRoles && requiredRoles.length > 0) {
    if (!hasRole(requiredRoles)) {
      return <Navigate to="/dashboard" replace />;
    }
  }

  return <>{children}</>;
};

// AdminRoute is a convenience wrapper for admin/owner only routes
export const AdminRoute = ({ children }: { children: React.ReactNode }) => {
  return (
    <ProtectedRoute requiredRoles={['admin', 'owner']}>
      {children}
    </ProtectedRoute>
  );
};

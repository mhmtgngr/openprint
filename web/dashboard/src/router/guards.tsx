import { Navigate } from 'react-router-dom';
import { useRequireAuth, useAuth } from '@/hooks/useAuth';
import { DashboardLayout } from '@/layouts/DashboardLayout';
import { PublicLayout } from '@/layouts/PublicLayout';
import type { AuthLevel } from './routes';

const LoadingSpinner = () => (
  <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
    <div className="flex flex-col items-center gap-4">
      <div className="w-12 h-12 border-4 border-blue-600 border-t-transparent rounded-full animate-spin" />
      <p className="text-gray-600 dark:text-gray-400">Loading...</p>
    </div>
  </div>
);

const ProtectedGuard = ({ children }: { children: React.ReactNode }) => {
  const { isAuthenticated, isLoading } = useRequireAuth();
  if (isLoading) return <LoadingSpinner />;
  if (!isAuthenticated) return <Navigate to="/login" replace />;
  return <DashboardLayout>{children}</DashboardLayout>;
};

const AdminGuard = ({ children }: { children: React.ReactNode }) => {
  const { isAuthenticated, isLoading, hasRole } = useAuth();
  if (isLoading) return <LoadingSpinner />;
  if (!isAuthenticated || !hasRole(['admin', 'owner', 'platform_admin'])) {
    return <Navigate to="/dashboard" replace />;
  }
  return <DashboardLayout>{children}</DashboardLayout>;
};

const PlatformAdminGuard = ({ children }: { children: React.ReactNode }) => {
  const { isAuthenticated, isLoading, user } = useAuth();
  if (isLoading) return <LoadingSpinner />;
  const isPlatformAdmin = user?.role === 'platform_admin' || user?.isPlatformAdmin;
  if (!isAuthenticated || !isPlatformAdmin) {
    return <Navigate to="/dashboard" replace />;
  }
  return <DashboardLayout>{children}</DashboardLayout>;
};

const PublicGuard = ({ children }: { children: React.ReactNode }) => (
  <PublicLayout>{children}</PublicLayout>
);

/** Map auth level to the appropriate route guard wrapper. */
const guardMap: Record<AuthLevel, React.FC<{ children: React.ReactNode }>> = {
  public: PublicGuard,
  authenticated: ProtectedGuard,
  admin: AdminGuard,
  platform_admin: PlatformAdminGuard,
};

export const getGuard = (level: AuthLevel): React.FC<{ children: React.ReactNode }> =>
  guardMap[level];

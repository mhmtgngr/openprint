import { lazy, Suspense } from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import { useRequireAuth, useAuth } from './hooks/useAuth';
import { PublicLayout } from './layouts/PublicLayout';
import { DashboardLayout } from './layouts/DashboardLayout';
import { PageLoadingFallback } from './components/LoadingFallback';
import { ErrorBoundary } from './components/ErrorBoundary';

// Lazy-loaded page components for code-splitting
const Login = lazy(() => import('./pages/Login').then(m => ({ default: m.Login })));
const Dashboard = lazy(() => import('./pages/Dashboard').then(m => ({ default: m.Dashboard })));
const Printers = lazy(() => import('./pages/Printers').then(m => ({ default: m.Printers })));
const Jobs = lazy(() => import('./pages/Jobs').then(m => ({ default: m.Jobs })));
const Analytics = lazy(() => import('./pages/Analytics').then(m => ({ default: m.Analytics })));
const Settings = lazy(() => import('./pages/Settings').then(m => ({ default: m.Settings })));
const Organization = lazy(() => import('./pages/Organization').then(m => ({ default: m.Organization })));
const Documents = lazy(() => import('./pages/Documents').then(m => ({ default: m.Documents })));
const Agents = lazy(() => import('./pages/Agents').then(m => ({ default: m.Agents })));
const AgentDetailPage = lazy(() => import('./pages/AgentDetail').then(m => ({ default: m.AgentDetailPage })));
const DiscoveredPrintersPage = lazy(() => import('./pages/DiscoveredPrinters').then(m => ({ default: m.DiscoveredPrintersPage })));
const JobAssignmentsPage = lazy(() => import('./pages/JobAssignments').then(m => ({ default: m.JobAssignmentsPage })));
const Quotas = lazy(() => import('./pages/Quotas').then(m => ({ default: m.Quotas })));
const Policies = lazy(() => import('./pages/Policies').then(m => ({ default: m.Policies })));
const AuditLogs = lazy(() => import('./pages/AuditLogs').then(m => ({ default: m.AuditLogs })));
const EmailToPrint = lazy(() => import('./pages/EmailToPrint').then(m => ({ default: m.EmailToPrint })));
const PrintReleasePage = lazy(() => import('./pages/PrintRelease').then(m => ({ default: m.PrintReleasePage })));
const Compliance = lazy(() => import('./pages/Compliance').then(m => ({ default: m.Compliance })));
const Microsoft365 = lazy(() => import('./pages/Microsoft365').then(m => ({ default: m.Microsoft365 })));
const SecurePrint = lazy(() => import('./pages/SecurePrint').then(m => ({ default: m.SecurePrint })));
const PoliciesEngine = lazy(() => import('./pages/PoliciesEngine').then(m => ({ default: m.PoliciesEngine })));
const MetricsDashboard = lazy(() => import('./pages/MetricsDashboard').then(m => ({ default: m.MetricsDashboard })));
const Monitoring = lazy(() => import('./pages/Monitoring').then(m => ({ default: m.Monitoring })));
const ObservabilityHub = lazy(() => import('./pages/ObservabilityHub').then(m => ({ default: m.ObservabilityHub })));

// New feature pages
const GuestPrinting = lazy(() => import('./pages/GuestPrinting').then(m => ({ default: m.GuestPrinting })));
const FollowMe = lazy(() => import('./pages/FollowMe').then(m => ({ default: m.FollowMe })));
const SupplyManagement = lazy(() => import('./pages/SupplyManagement').then(m => ({ default: m.SupplyManagement })));
const DriverManagement = lazy(() => import('./pages/DriverManagement').then(m => ({ default: m.DriverManagement })));
const UserGroups = lazy(() => import('./pages/UserGroups').then(m => ({ default: m.UserGroups })));
const NotFound = lazy(() => import('./pages/NotFound').then(m => ({ default: m.NotFound })));

// Platform Admin routes (multi-tenancy)
const OrganizationsList = lazy(() => import('./pages/admin/OrganizationsList').then(m => ({ default: m.OrganizationsList })));
const OrganizationDetail = lazy(() => import('./pages/admin/OrganizationDetail').then(m => ({ default: m.OrganizationDetail })));

const SuspenseWrapper = ({ children }: { children: React.ReactNode }) => (
  <Suspense fallback={<PageLoadingFallback />}>
    <ErrorBoundary>
      {children}
    </ErrorBoundary>
  </Suspense>
);

const ProtectedRoute = ({ children }: { children: React.ReactNode }) => {
  const { isAuthenticated, isLoading } = useRequireAuth();

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

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  return <DashboardLayout>{children}</DashboardLayout>;
};

const AdminRoute = ({ children }: { children: React.ReactNode }) => {
  const { isAuthenticated, isLoading, hasRole } = useAuth();

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
        <div className="w-12 h-12 border-4 border-blue-600 border-t-transparent rounded-full animate-spin" />
      </div>
    );
  }

  if (!isAuthenticated || !hasRole(['admin', 'owner', 'platform_admin'])) {
    return <Navigate to="/dashboard" replace />;
  }

  return <DashboardLayout>{children}</DashboardLayout>;
};

// Platform Admin Route - only for platform administrators
const PlatformAdminRoute = ({ children }: { children: React.ReactNode }) => {
  const { isAuthenticated, isLoading, user } = useAuth();

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
        <div className="w-12 h-12 border-4 border-blue-600 border-t-transparent rounded-full animate-spin" />
      </div>
    );
  }

  const isPlatformAdmin = user?.role === 'platform_admin' || user?.isPlatformAdmin;

  if (!isAuthenticated || !isPlatformAdmin) {
    return <Navigate to="/dashboard" replace />;
  }

  return <DashboardLayout>{children}</DashboardLayout>;
};

function App() {
  return (
    <ErrorBoundary>
      <Suspense fallback={<PageLoadingFallback />}>
        <Routes>
          {/* Public routes */}
          <Route
            path="/login"
            element={
              <PublicLayout>
                <SuspenseWrapper>
                  <Login />
                </SuspenseWrapper>
              </PublicLayout>
            }
          />

          {/* Protected routes */}
          <Route
            path="/dashboard"
            element={
              <ProtectedRoute>
                <SuspenseWrapper>
                  <Dashboard />
                </SuspenseWrapper>
              </ProtectedRoute>
            }
          />
          <Route
            path="/printers"
            element={
              <ProtectedRoute>
                <SuspenseWrapper>
                  <Printers />
                </SuspenseWrapper>
              </ProtectedRoute>
            }
          />
          <Route
            path="/jobs"
            element={
              <ProtectedRoute>
                <SuspenseWrapper>
                  <Jobs />
                </SuspenseWrapper>
              </ProtectedRoute>
            }
          />
          <Route
            path="/documents"
            element={
              <ProtectedRoute>
                <SuspenseWrapper>
                  <Documents />
                </SuspenseWrapper>
              </ProtectedRoute>
            }
          />
          <Route
            path="/settings"
            element={
              <ProtectedRoute>
                <SuspenseWrapper>
                  <Settings />
                </SuspenseWrapper>
              </ProtectedRoute>
            }
          />
          <Route
            path="/analytics"
            element={
              <AdminRoute>
                <SuspenseWrapper>
                  <Analytics />
                </SuspenseWrapper>
              </AdminRoute>
            }
          />
          <Route
            path="/organization"
            element={
              <AdminRoute>
                <SuspenseWrapper>
                  <Organization />
                </SuspenseWrapper>
              </AdminRoute>
            }
          />
          <Route
            path="/agents"
            element={
              <ProtectedRoute>
                <SuspenseWrapper>
                  <Agents />
                </SuspenseWrapper>
              </ProtectedRoute>
            }
          />
          <Route
            path="/agents/:id"
            element={
              <ProtectedRoute>
                <SuspenseWrapper>
                  <AgentDetailPage />
                </SuspenseWrapper>
              </ProtectedRoute>
            }
          />
          <Route
            path="/discovered-printers"
            element={
              <ProtectedRoute>
                <SuspenseWrapper>
                  <DiscoveredPrintersPage />
                </SuspenseWrapper>
              </ProtectedRoute>
            }
          />
          <Route
            path="/job-assignments"
            element={
              <AdminRoute>
                <SuspenseWrapper>
                  <JobAssignmentsPage />
                </SuspenseWrapper>
              </AdminRoute>
            }
          />
          <Route
            path="/quotas"
            element={
              <AdminRoute>
                <SuspenseWrapper>
                  <Quotas />
                </SuspenseWrapper>
              </AdminRoute>
            }
          />
          <Route
            path="/policies"
            element={
              <AdminRoute>
                <SuspenseWrapper>
                  <Policies />
                </SuspenseWrapper>
              </AdminRoute>
            }
          />
          <Route
            path="/audit-logs"
            element={
              <AdminRoute>
                <SuspenseWrapper>
                  <AuditLogs />
                </SuspenseWrapper>
              </AdminRoute>
            }
          />
          <Route
            path="/email-to-print"
            element={
              <AdminRoute>
                <SuspenseWrapper>
                  <EmailToPrint />
                </SuspenseWrapper>
              </AdminRoute>
            }
          />
          <Route
            path="/print-release"
            element={
              <ProtectedRoute>
                <SuspenseWrapper>
                  <PrintReleasePage />
                </SuspenseWrapper>
              </ProtectedRoute>
            }
          />
          <Route
            path="/compliance"
            element={
              <AdminRoute>
                <SuspenseWrapper>
                  <Compliance />
                </SuspenseWrapper>
              </AdminRoute>
            }
          />
          <Route
            path="/microsoft365"
            element={
              <AdminRoute>
                <SuspenseWrapper>
                  <Microsoft365 />
                </SuspenseWrapper>
              </AdminRoute>
            }
          />
          <Route
            path="/secure-print"
            element={
              <ProtectedRoute>
                <SuspenseWrapper>
                  <SecurePrint />
                </SuspenseWrapper>
              </ProtectedRoute>
            }
          />
          <Route
            path="/policy-engine"
            element={
              <AdminRoute>
                <SuspenseWrapper>
                  <PoliciesEngine />
                </SuspenseWrapper>
              </AdminRoute>
            }
          />
          <Route
            path="/metrics"
            element={
              <AdminRoute>
                <SuspenseWrapper>
                  <MetricsDashboard />
                </SuspenseWrapper>
              </AdminRoute>
            }
          />
          <Route
            path="/monitoring"
            element={
              <AdminRoute>
                <SuspenseWrapper>
                  <Monitoring />
                </SuspenseWrapper>
              </AdminRoute>
            }
          />
          <Route
            path="/observability"
            element={
              <AdminRoute>
                <SuspenseWrapper>
                  <ObservabilityHub />
                </SuspenseWrapper>
              </AdminRoute>
            }
          />

          {/* New feature routes */}
          <Route
            path="/guest-printing"
            element={
              <AdminRoute>
                <SuspenseWrapper>
                  <GuestPrinting />
                </SuspenseWrapper>
              </AdminRoute>
            }
          />
          <Route
            path="/follow-me"
            element={
              <ProtectedRoute>
                <SuspenseWrapper>
                  <FollowMe />
                </SuspenseWrapper>
              </ProtectedRoute>
            }
          />
          <Route
            path="/supplies"
            element={
              <AdminRoute>
                <SuspenseWrapper>
                  <SupplyManagement />
                </SuspenseWrapper>
              </AdminRoute>
            }
          />
          <Route
            path="/drivers"
            element={
              <AdminRoute>
                <SuspenseWrapper>
                  <DriverManagement />
                </SuspenseWrapper>
              </AdminRoute>
            }
          />
          <Route
            path="/groups"
            element={
              <AdminRoute>
                <SuspenseWrapper>
                  <UserGroups />
                </SuspenseWrapper>
              </AdminRoute>
            }
          />

          {/* Platform Admin routes (multi-tenancy) */}
          <Route
            path="/admin/organizations"
            element={
              <PlatformAdminRoute>
                <SuspenseWrapper>
                  <OrganizationsList />
                </SuspenseWrapper>
              </PlatformAdminRoute>
            }
          />
          <Route
            path="/admin/organizations/:orgId"
            element={
              <PlatformAdminRoute>
                <SuspenseWrapper>
                  <OrganizationDetail />
                </SuspenseWrapper>
              </PlatformAdminRoute>
            }
          />

          {/* Default redirect */}
          <Route path="/" element={<Navigate to="/dashboard" replace />} />

          {/* 404 - Not Found */}
          <Route
            path="*"
            element={
              <SuspenseWrapper>
                <NotFound />
              </SuspenseWrapper>
            }
          />
        </Routes>
      </Suspense>
    </ErrorBoundary>
  );
}

export default App;

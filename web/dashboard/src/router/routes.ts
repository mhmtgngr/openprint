import { lazy } from 'react';

// Auth levels matching the route guards
export type AuthLevel = 'public' | 'authenticated' | 'admin' | 'platform_admin';

export interface RouteConfig {
  path: string;
  auth: AuthLevel;
  component: ReturnType<typeof lazy>;
}

// Helper to lazy-load with named export
const lazyPage = (importFn: () => Promise<Record<string, unknown>>, name: string) =>
  lazy(() => importFn().then(m => ({ default: m[name] as React.ComponentType })));

// --- Route definitions ---
// Adding a new route is a single line here.

export const routes: RouteConfig[] = [
  // Public
  { path: '/login', auth: 'public', component: lazyPage(() => import('@/pages/Login'), 'Login') },

  // Authenticated user routes
  { path: '/dashboard', auth: 'authenticated', component: lazyPage(() => import('@/pages/Dashboard'), 'Dashboard') },
  { path: '/permissions', auth: 'authenticated', component: lazyPage(() => import('@/pages/Permissions'), 'PermissionsPage') },
  { path: '/printers', auth: 'authenticated', component: lazyPage(() => import('@/pages/Printers'), 'Printers') },
  { path: '/jobs', auth: 'authenticated', component: lazyPage(() => import('@/pages/Jobs'), 'Jobs') },
  { path: '/documents', auth: 'authenticated', component: lazyPage(() => import('@/pages/Documents'), 'Documents') },
  { path: '/settings', auth: 'authenticated', component: lazyPage(() => import('@/pages/Settings'), 'Settings') },
  { path: '/agents', auth: 'authenticated', component: lazyPage(() => import('@/pages/Agents'), 'Agents') },
  { path: '/agents/:id', auth: 'authenticated', component: lazyPage(() => import('@/pages/AgentDetail'), 'AgentDetailPage') },
  { path: '/discovered-printers', auth: 'authenticated', component: lazyPage(() => import('@/pages/DiscoveredPrinters'), 'DiscoveredPrintersPage') },
  { path: '/print-release', auth: 'authenticated', component: lazyPage(() => import('@/pages/PrintRelease'), 'PrintReleasePage') },
  { path: '/secure-print', auth: 'authenticated', component: lazyPage(() => import('@/pages/SecurePrint'), 'SecurePrint') },
  { path: '/follow-me', auth: 'authenticated', component: lazyPage(() => import('@/pages/FollowMe'), 'FollowMe') },

  // Admin routes
  { path: '/analytics', auth: 'admin', component: lazyPage(() => import('@/pages/Analytics'), 'Analytics') },
  { path: '/organization', auth: 'admin', component: lazyPage(() => import('@/pages/Organization'), 'Organization') },
  { path: '/job-assignments', auth: 'admin', component: lazyPage(() => import('@/pages/JobAssignments'), 'JobAssignmentsPage') },
  { path: '/quotas', auth: 'admin', component: lazyPage(() => import('@/pages/Quotas'), 'Quotas') },
  { path: '/policies', auth: 'admin', component: lazyPage(() => import('@/pages/Policies'), 'Policies') },
  { path: '/audit-logs', auth: 'admin', component: lazyPage(() => import('@/pages/AuditLogs'), 'AuditLogs') },
  { path: '/email-to-print', auth: 'admin', component: lazyPage(() => import('@/pages/EmailToPrint'), 'EmailToPrint') },
  { path: '/compliance', auth: 'admin', component: lazyPage(() => import('@/pages/Compliance'), 'Compliance') },
  { path: '/microsoft365', auth: 'admin', component: lazyPage(() => import('@/pages/Microsoft365'), 'Microsoft365') },
  { path: '/policy-engine', auth: 'admin', component: lazyPage(() => import('@/pages/PoliciesEngine'), 'PoliciesEngine') },
  { path: '/metrics', auth: 'admin', component: lazyPage(() => import('@/pages/MetricsDashboard'), 'MetricsDashboard') },
  { path: '/monitoring', auth: 'admin', component: lazyPage(() => import('@/pages/Monitoring'), 'Monitoring') },
  { path: '/observability', auth: 'admin', component: lazyPage(() => import('@/pages/ObservabilityHub'), 'ObservabilityHub') },
  { path: '/guest-printing', auth: 'admin', component: lazyPage(() => import('@/pages/GuestPrinting'), 'GuestPrinting') },
  { path: '/supplies', auth: 'admin', component: lazyPage(() => import('@/pages/SupplyManagement'), 'SupplyManagement') },
  { path: '/drivers', auth: 'admin', component: lazyPage(() => import('@/pages/DriverManagement'), 'DriverManagement') },
  { path: '/groups', auth: 'admin', component: lazyPage(() => import('@/pages/UserGroups'), 'UserGroups') },

  // Platform admin routes
  { path: '/admin/organizations', auth: 'platform_admin', component: lazyPage(() => import('@/pages/admin/OrganizationsList'), 'OrganizationsList') },
  { path: '/admin/organizations/:orgId', auth: 'platform_admin', component: lazyPage(() => import('@/pages/admin/OrganizationDetail'), 'OrganizationDetail') },
];

// 404 route (loaded separately since it has no auth guard)
export const notFoundComponent = lazyPage(() => import('@/pages/NotFound'), 'NotFound');

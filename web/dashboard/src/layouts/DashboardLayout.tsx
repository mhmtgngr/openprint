import { Link, useLocation } from 'react-router-dom';
import { useState } from 'react';
import { useAuth } from '@/hooks/useAuth';
import { NotificationCenter } from '@/components/NotificationCenter';
import { QuickPrintDialog } from '@/components/QuickPrintDialog';
import {
  HomeIcon,
  PrinterIcon,
  DocumentIcon,
  FolderIcon,
  CogIcon,
  LogoutIcon,
  MetricsIcon,
  BellIcon,
  PulseIcon,
  ChartIcon,
  UsersIcon,
} from '@/components/icons';

const allRoles = ['user', 'admin', 'owner', 'platform_admin'];
const adminRoles = ['admin', 'owner', 'platform_admin'];
const platformAdminRoles = ['platform_admin'];

interface NavItem {
  path: string;
  icon: React.FC<{ className?: string }>;
  label: string;
  roles: string[];
  section?: string;
}

const navItems: NavItem[] = [
  // --- User section ---
  { path: '/dashboard', icon: HomeIcon, label: 'Dashboard', roles: allRoles },
  { path: '/printers', icon: PrinterIcon, label: 'Devices', roles: allRoles },
  { path: '/jobs', icon: DocumentIcon, label: 'Jobs', roles: allRoles },
  { path: '/follow-me', icon: PrinterIcon, label: 'Follow-Me', roles: allRoles },
  { path: '/documents', icon: FolderIcon, label: 'Documents', roles: allRoles },
  { path: '/secure-print', icon: DocumentIcon, label: 'Secure Print', roles: allRoles },

  // --- Admin section ---
  { path: '/analytics', icon: ChartIcon, label: 'Analytics', roles: adminRoles, section: 'Admin' },
  { path: '/organization', icon: UsersIcon, label: 'Organization', roles: adminRoles },
  { path: '/policies', icon: DocumentIcon, label: 'Policies', roles: adminRoles },
  { path: '/quotas', icon: MetricsIcon, label: 'Quotas', roles: adminRoles },
  { path: '/supplies', icon: PulseIcon, label: 'Supplies', roles: adminRoles },
  { path: '/drivers', icon: FolderIcon, label: 'Drivers', roles: adminRoles },
  { path: '/groups', icon: UsersIcon, label: 'Groups', roles: adminRoles },
  { path: '/guest-printing', icon: DocumentIcon, label: 'Guest Print', roles: adminRoles },
  { path: '/email-to-print', icon: DocumentIcon, label: 'Email-to-Print', roles: adminRoles },
  { path: '/audit-logs', icon: FolderIcon, label: 'Audit Logs', roles: adminRoles },

  // --- System section ---
  { path: '/metrics', icon: MetricsIcon, label: 'Metrics', roles: adminRoles, section: 'System' },
  { path: '/monitoring', icon: BellIcon, label: 'Monitoring', roles: adminRoles },
  { path: '/observability', icon: PulseIcon, label: 'Tracing', roles: adminRoles },

  // --- Platform Admin ---
  { path: '/admin/organizations', icon: UsersIcon, label: 'Organizations', roles: platformAdminRoles, section: 'Platform' },

  // --- Always last ---
  { path: '/settings', icon: CogIcon, label: 'Settings', roles: allRoles },
];

interface DashboardLayoutProps {
  children: React.ReactNode;
}

export const DashboardLayout = ({ children }: DashboardLayoutProps) => {
  const location = useLocation();
  const { user, logout } = useAuth();
  const [showQuickPrint, setShowQuickPrint] = useState(false);

  const filteredNavItems = navItems.filter(
    (item) => user && item.roles.includes(user.role)
  );

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
      {/* Sidebar */}
      <aside className="fixed inset-y-0 left-0 w-64 bg-white dark:bg-gray-800 border-r border-gray-200 dark:border-gray-700 z-10">
        <div className="flex flex-col h-full">
          {/* Logo */}
          <Link to="/dashboard" className="flex items-center gap-2 p-6 border-b border-gray-200 dark:border-gray-700">
            <div className="w-8 h-8 bg-gradient-to-br from-blue-500 to-cyan-500 rounded-lg flex items-center justify-center">
              <svg className="w-5 h-5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2h2m2 4h6a2 2 0 002-2v-4a2 2 0 00-2-2H9a2 2 0 00-2 2v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z"
                />
              </svg>
            </div>
            <span className="text-xl font-bold text-gray-900 dark:text-gray-100">
              OpenPrint
            </span>
          </Link>

          {/* Navigation */}
          <nav className="flex-1 p-4 space-y-1 overflow-y-auto">
            {filteredNavItems.map((item, index) => {
              const Icon = item.icon;
              const isActive = location.pathname === item.path ||
                location.pathname.startsWith(`${item.path}/`);

              // Show section header if this item starts a new section
              const showSection = item.section && (
                index === 0 || filteredNavItems[index - 1]?.section !== item.section
              );

              return (
                <div key={item.path}>
                  {showSection && (
                    <div className="pt-4 pb-1 px-4">
                      <p className="text-xs font-semibold uppercase tracking-wider text-gray-400 dark:text-gray-500">
                        {item.section}
                      </p>
                    </div>
                  )}
                  <Link
                    to={item.path}
                    className={`
                      flex items-center gap-3 px-4 py-2.5 rounded-lg transition-colors
                      ${isActive
                        ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300'
                        : 'text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700'
                      }
                    `}
                  >
                    <Icon className="w-5 h-5" />
                    <span className="font-medium text-sm">{item.label}</span>
                  </Link>
                </div>
              );
            })}
          </nav>

          {/* User menu */}
          <div className="p-4 border-t border-gray-200 dark:border-gray-700">
            <div className="flex items-center gap-3 mb-3 px-4">
              <div className="w-10 h-10 bg-gradient-to-br from-blue-500 to-cyan-500 rounded-full flex items-center justify-center text-white font-semibold">
                {user?.name?.charAt(0).toUpperCase() || user?.email?.charAt(0).toUpperCase()}
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium text-gray-900 dark:text-gray-100 truncate">
                  {user?.name}
                </p>
                <p className="text-xs text-gray-500 dark:text-gray-400 truncate">{user?.email}</p>
              </div>
            </div>
            <button
              onClick={logout}
              className="flex items-center gap-3 px-4 py-2 w-full text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
            >
              <LogoutIcon className="w-5 h-5" />
              <span className="font-medium">Logout</span>
            </button>
          </div>
        </div>
      </aside>

      {/* Main content */}
      <div className="ml-64">
        {/* Header */}
        <header className="bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 px-8 py-4">
          <div className="flex items-center justify-between">
            <h1 className="text-2xl font-semibold text-gray-900 dark:text-gray-100">
              {navItems.find(item => location.pathname.startsWith(item.path))?.label || 'Dashboard'}
            </h1>
            <div className="flex items-center gap-2">
              {/* Quick Print button */}
              <button
                onClick={() => setShowQuickPrint(true)}
                className="flex items-center gap-2 px-3 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-lg transition-colors"
              >
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" d="M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2h2m2 4h6a2 2 0 002-2v-4a2 2 0 00-2-2H9a2 2 0 00-2 2v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z" />
                </svg>
                Quick Print
              </button>
              {/* Notification Center */}
              <NotificationCenter />
            </div>
          </div>
        </header>

        {/* Page content */}
        <main className="p-8">
          {children}
        </main>
      </div>

      {/* Quick Print Dialog */}
      {showQuickPrint && (
        <QuickPrintDialog onClose={() => setShowQuickPrint(false)} />
      )}
    </div>
  );
};

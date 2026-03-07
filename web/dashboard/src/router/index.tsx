// Navigation configuration for use in sidebar
export interface NavItem {
  path: string;
  label: string;
  icon: string;
  roles: string[];
  category?: string;
}

export const navConfig: NavItem[] = [
  // Main navigation (all users)
  { path: '/dashboard', label: 'Dashboard', icon: 'HomeIcon', roles: ['user', 'admin', 'owner'], category: 'main' },
  { path: '/printers', label: 'Devices', icon: 'PrinterIcon', roles: ['user', 'admin', 'owner'], category: 'main' },
  { path: '/jobs', label: 'Jobs', icon: 'DocumentIcon', roles: ['user', 'admin', 'owner'], category: 'main' },
  { path: '/documents', label: 'Documents', icon: 'FolderIcon', roles: ['user', 'admin', 'owner'], category: 'main' },

  // Print management
  { path: '/print-release', label: 'Print Release', icon: 'LockClosedIcon', roles: ['user', 'admin', 'owner'], category: 'print' },
  { path: '/secure-print', label: 'Secure Print', icon: 'ShieldIcon', roles: ['user', 'admin', 'owner'], category: 'print' },

  // Admin only
  { path: '/agents', label: 'Agents', icon: 'UsersIcon', roles: ['admin', 'owner'], category: 'admin' },
  { path: '/analytics', label: 'Analytics', icon: 'ChartIcon', roles: ['admin', 'owner'], category: 'admin' },
  { path: '/organization', label: 'Organization', icon: 'UsersIcon', roles: ['admin', 'owner'], category: 'admin' },
  { path: '/quotas', label: 'Quotas', icon: 'MetricsIcon', roles: ['admin', 'owner'], category: 'admin' },
  { path: '/policies', label: 'Policies', icon: 'ShieldIcon', roles: ['admin', 'owner'], category: 'admin' },
  { path: '/policy-engine', label: 'Policy Engine', icon: 'ShieldIcon', roles: ['admin', 'owner'], category: 'admin' },
  { path: '/audit-logs', label: 'Audit Logs', icon: 'ClipboardDocumentListIcon', roles: ['admin', 'owner'], category: 'admin' },
  { path: '/compliance', label: 'Compliance', icon: 'CheckCircleIcon', roles: ['admin', 'owner'], category: 'admin' },

  // Integrations
  { path: '/email-to-print', label: 'Email-to-Print', icon: 'EnvelopeIcon', roles: ['admin', 'owner'], category: 'integrations' },
  { path: '/microsoft365', label: 'Microsoft 365', icon: 'CloudIcon', roles: ['admin', 'owner'], category: 'integrations' },

  // Observability
  { path: '/metrics', label: 'Metrics', icon: 'MetricsIcon', roles: ['admin', 'owner'], category: 'observability' },
  { path: '/monitoring', label: 'Monitoring', icon: 'BellIcon', roles: ['admin', 'owner'], category: 'observability' },
  { path: '/observability', label: 'Tracing', icon: 'PulseIcon', roles: ['admin', 'owner'], category: 'observability' },

  // Settings
  { path: '/settings', label: 'Settings', icon: 'CogIcon', roles: ['user', 'admin', 'owner'], category: 'settings' },
];

// Helper to get nav items by user role
export const getNavItemsForRole = (role: string): NavItem[] => {
  return navConfig.filter(item => item.roles.includes(role));
};

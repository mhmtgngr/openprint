// Navigation configuration for use in sidebar
export interface NavItem {
  path: string;
  label: string;
  icon: string;
  roles: string[];
}

export const navConfig: NavItem[] = [
  { path: '/dashboard', label: 'Dashboard', icon: 'HomeIcon', roles: ['user', 'admin', 'owner'] },
  { path: '/jobs', label: 'Jobs', icon: 'DocumentIcon', roles: ['user', 'admin', 'owner'] },
  { path: '/printers', label: 'Devices', icon: 'PrinterIcon', roles: ['user', 'admin', 'owner'] },
  { path: '/documents', label: 'Documents', icon: 'FolderIcon', roles: ['user', 'admin', 'owner'] },
  { path: '/print-release', label: 'Print Release', icon: 'LockClosedIcon', roles: ['user', 'admin', 'owner'] },
  { path: '/secure-print', label: 'Secure Print', icon: 'ShieldCheckIcon', roles: ['user', 'admin', 'owner'] },
  { path: '/settings', label: 'Settings', icon: 'CogIcon', roles: ['user', 'admin', 'owner'] },
  { path: '/quotas', label: 'Quotas', icon: 'ChartBarIcon', roles: ['admin', 'owner'] },
  { path: '/policies', label: 'Policies', icon: 'ShieldCheckIcon', roles: ['admin', 'owner'] },
  { path: '/policy-engine', label: 'Policy Engine', icon: 'BeakerIcon', roles: ['admin', 'owner'] },
  { path: '/audit-logs', label: 'Audit Logs', icon: 'ClipboardDocumentListIcon', roles: ['admin', 'owner'] },
  { path: '/compliance', label: 'Compliance', icon: 'CheckCircleIcon', roles: ['admin', 'owner'] },
  { path: '/email-to-print', label: 'Email-to-Print', icon: 'EnvelopeIcon', roles: ['admin', 'owner'] },
  { path: '/microsoft365', label: 'Microsoft 365', icon: 'CloudIcon', roles: ['admin', 'owner'] },
];

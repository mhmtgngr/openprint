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
  { path: '/settings', label: 'Settings', icon: 'CogIcon', roles: ['user', 'admin', 'owner'] },
];

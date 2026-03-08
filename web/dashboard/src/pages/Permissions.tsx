import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { RolePermissions, RoleComparison, PermissionBadge } from '@/components/RolePermissions';
import { ShieldIcon, UsersIcon, LockClosedIcon } from '@/components/icons';

// Mock permission groups - in production, fetch from API
const mockPermissionGroups = [
  {
    name: 'Printer Management',
    description: 'Control access to printer resources',
    permissions: [
      { name: 'Create Printers', resource: 'printers', action: 'create', description: 'Add new printers to the system', granted: false },
      { name: 'View Printers', resource: 'printers', action: 'read', description: 'View printer details and configuration', granted: true },
      { name: 'Update Printers', resource: 'printers', action: 'update', description: 'Modify printer settings and configuration', granted: false },
      { name: 'Delete Printers', resource: 'printers', action: 'delete', description: 'Remove printers from the system', granted: false },
      { name: 'List Printers', resource: 'printers', action: 'list', description: 'View list of all printers', granted: true },
    ],
  },
  {
    name: 'Agent Management',
    description: 'Control access to print agents',
    permissions: [
      { name: 'Create Agents', resource: 'agents', action: 'create', description: 'Register new print agents', granted: false },
      { name: 'View Agents', resource: 'agents', action: 'read', description: 'View agent details and status', granted: true },
      { name: 'Update Agents', resource: 'agents', action: 'update', description: 'Modify agent configuration', granted: false },
      { name: 'Delete Agents', resource: 'agents', action: 'delete', description: 'Remove agents from the system', granted: false },
      { name: 'List Agents', resource: 'agents', action: 'list', description: 'View list of all agents', granted: true },
      { name: 'View Metrics', resource: 'agents', action: 'view_metrics', description: 'View agent performance metrics', granted: true },
    ],
  },
  {
    name: 'Job Management',
    description: 'Control access to print jobs',
    permissions: [
      { name: 'Create Jobs', resource: 'jobs', action: 'create', description: 'Submit new print jobs', granted: true },
      { name: 'View Any Job', resource: 'jobs', action: 'read', description: 'View any print job details', granted: false },
      { name: 'View Own Jobs', resource: 'jobs', action: 'read_own', description: 'View only own print job details', granted: true },
      { name: 'Update Jobs', resource: 'jobs', action: 'update', description: 'Modify print job settings', granted: false },
      { name: 'Cancel Jobs', resource: 'jobs', action: 'cancel', description: 'Cancel queued or active print jobs', granted: true },
      { name: 'List Jobs', resource: 'jobs', action: 'list', description: 'View list of print jobs', granted: true },
    ],
  },
  {
    name: 'User Management',
    description: 'Control access to user accounts',
    permissions: [
      { name: 'Create Users', resource: 'users', action: 'create', description: 'Create new user accounts', granted: false },
      { name: 'View Users', resource: 'users', action: 'read', description: 'View user account details', granted: false },
      { name: 'Update Users', resource: 'users', action: 'update', description: 'Modify user account settings', granted: false },
      { name: 'Delete Users', resource: 'users', action: 'delete', description: 'Remove user accounts', granted: false },
      { name: 'Manage Roles', resource: 'users', action: 'manage_roles', description: 'Assign and modify user roles', granted: false },
    ],
  },
  {
    name: 'Reporting & Analytics',
    description: 'Access to reports and usage data',
    permissions: [
      { name: 'View Reports', resource: 'reports', action: 'read', description: 'View usage and performance reports', granted: true },
      { name: 'Export Reports', resource: 'reports', action: 'export', description: 'Export reports in various formats', granted: false },
      { name: 'View Costs', resource: 'reports', action: 'view_costs', description: 'View cost and billing information', granted: false },
      { name: 'View Usage', resource: 'reports', action: 'view_usage', description: 'View usage statistics and analytics', granted: true },
    ],
  },
];

const allPermissions = mockPermissionGroups.flatMap(g =>
  g.permissions.map(p => `${p.resource}:${p.action}`)
);

export const PermissionsPage = () => {
  const [viewMode, setViewMode] = useState<'current' | 'compare'>('current');

  // Fetch user's current role
  const { data: user } = useQuery({
    queryKey: ['currentUser'],
    queryFn: async () => {
      const res = await fetch('/api/v1/users/me');
      if (!res.ok) throw new Error('Failed to fetch user');
      return res.json();
    },
  });

  const currentRole = user?.role || 'org_user';

  // In production, fetch permissions for the current role from API
  const rolePermissions = mockPermissionGroups.map(group => ({
    ...group,
    permissions: group.permissions.map(perm => {
      // Mock permission check based on role
      let granted = false;
      switch (currentRole) {
        case 'platform_admin':
          granted = true;
          break;
        case 'org_admin':
          granted = perm.action !== 'create' || perm.resource === 'agents' || perm.resource === 'printers';
          if (perm.resource === 'users' || perm.resource === 'reports') granted = perm.action === 'read' || perm.action === 'list';
          break;
        case 'org_user':
          granted = ['read', 'read_own', 'list', 'create', 'cancel'].includes(perm.action) &&
                    !['users', 'organizations', 'settings'].includes(perm.resource);
          if (perm.resource === 'jobs' && perm.action === 'create') granted = true;
          break;
        case 'org_viewer':
          granted = ['read', 'read_own', 'list'].includes(perm.action);
          break;
      }
      return { ...perm, granted };
    }),
  }));

  return (
    <div className="max-w-6xl mx-auto space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">Permissions & Roles</h1>
          <p className="text-gray-600 dark:text-gray-400 mt-1">
            Understand what you can do based on your role and permissions
          </p>
        </div>
        <div className="flex items-center gap-2 p-2 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
          <ShieldIcon className="w-5 h-5 text-blue-600 dark:text-blue-400" />
          <span className="text-sm font-medium text-blue-900 dark:text-blue-100">
            {currentRole.replace(/_/g, ' ').replace(/\b\w/g, (l: string) => l.toUpperCase())}
          </span>
        </div>
      </div>

      {/* View Mode Tabs */}
      <div className="border-b border-gray-200 dark:border-gray-700">
        <nav className="flex gap-6">
          <button
            onClick={() => setViewMode('current')}
            className={`pb-3 px-1 border-b-2 font-medium text-sm transition-colors ${
              viewMode === 'current'
                ? 'border-blue-600 text-blue-600 dark:text-blue-400'
                : 'border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300'
            }`}
          >
            My Permissions
          </button>
          <button
            onClick={() => setViewMode('compare')}
            className={`pb-3 px-1 border-b-2 font-medium text-sm transition-colors ${
              viewMode === 'compare'
                ? 'border-blue-600 text-blue-600 dark:text-blue-400'
                : 'border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300'
            }`}
          >
            Compare Roles
          </button>
        </nav>
      </div>

      {/* Current Role View */}
      {viewMode === 'current' && (
        <div className="space-y-6">
          {/* Quick Summary */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="bg-white dark:bg-gray-800 rounded-lg p-4 border border-gray-200 dark:border-gray-700">
              <div className="flex items-center gap-3">
                <div className="p-2 bg-green-100 dark:bg-green-900/30 rounded-lg text-green-600 dark:text-green-400">
                  <ShieldIcon className="w-5 h-5" />
                </div>
                <div>
                  <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">
                    {rolePermissions.reduce((sum, g) => sum + g.permissions.filter(p => p.granted).length, 0)}
                  </p>
                  <p className="text-sm text-gray-500 dark:text-gray-400">Permissions Granted</p>
                </div>
              </div>
            </div>

            <div className="bg-white dark:bg-gray-800 rounded-lg p-4 border border-gray-200 dark:border-gray-700">
              <div className="flex items-center gap-3">
                <div className="p-2 bg-blue-100 dark:bg-blue-900/30 rounded-lg text-blue-600 dark:text-blue-400">
                  <LockClosedIcon className="w-5 h-5" />
                </div>
                <div>
                  <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">
                    {rolePermissions.reduce((sum, g) => sum + g.permissions.filter(p => !p.granted).length, 0)}
                  </p>
                  <p className="text-sm text-gray-500 dark:text-gray-400">Restricted</p>
                </div>
              </div>
            </div>

            <div className="bg-white dark:bg-gray-800 rounded-lg p-4 border border-gray-200 dark:border-gray-700">
              <div className="flex items-center gap-3">
                <div className="p-2 bg-purple-100 dark:bg-purple-900/30 rounded-lg text-purple-600 dark:text-purple-400">
                  <UsersIcon className="w-5 h-5" />
                </div>
                <div>
                  <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">4</p>
                  <p className="text-sm text-gray-500 dark:text-gray-400">Role Types</p>
                </div>
              </div>
            </div>
          </div>

          {/* Permission Breakdown */}
          <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
            <div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
              <h2 className="font-semibold text-gray-900 dark:text-gray-100">Your Permissions</h2>
              <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
                Detailed breakdown of permissions for your current role
              </p>
            </div>
            <div className="p-6">
              <RolePermissions role={currentRole} groups={rolePermissions} />
            </div>
          </div>
        </div>
      )}

      {/* Role Comparison View */}
      {viewMode === 'compare' && (
        <div className="space-y-6">
          <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
            <div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
              <h2 className="font-semibold text-gray-900 dark:text-gray-100">Role Comparison</h2>
              <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
                Compare permissions across different roles
              </p>
            </div>
            <div className="p-6">
              <RoleComparison
                roles={['platform_admin', 'org_admin', 'org_user', 'org_viewer']}
                permissions={allPermissions.slice(0, 15)}
              />
              <p className="text-xs text-gray-500 dark:text-gray-400 mt-4">
                Showing first 15 permissions. Contact administrator for full comparison.
              </p>
            </div>
          </div>

          {/* Role Descriptions */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {[
              {
                role: 'platform_admin',
                name: 'Platform Admin',
                description: 'Full system access including all organizations, billing, and certificates.',
                color: 'red',
              },
              {
                role: 'org_admin',
                name: 'Organization Admin',
                description: 'Organization administrator with full access to manage organization resources.',
                color: 'blue',
              },
              {
                role: 'org_user',
                name: 'Organization User',
                description: 'Standard user who can submit print jobs and view their own resources.',
                color: 'green',
              },
              {
                role: 'org_viewer',
                name: 'Organization Viewer',
                description: 'Read-only access to view organization resources without making changes.',
                color: 'gray',
              },
            ].map((roleInfo) => (
              <div
                key={roleInfo.role}
                className={`bg-white dark:bg-gray-800 rounded-lg p-4 border-l-4 ${
                  roleInfo.color === 'red' ? 'border-red-500' :
                  roleInfo.color === 'blue' ? 'border-blue-500' :
                  roleInfo.color === 'green' ? 'border-green-500' :
                  'border-gray-500'
                } border border-gray-200 dark:border-gray-700`}
              >
                <h3 className="font-semibold text-gray-900 dark:text-gray-100">{roleInfo.name}</h3>
                <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">{roleInfo.description}</p>
                <div className="mt-3 flex flex-wrap gap-2">
                  {roleInfo.role === 'platform_admin' && (
                    <>
                      <PermissionBadge permission="printers:manage" />
                      <PermissionBadge permission="users:create" />
                      <PermissionBadge permission="orgs:manage_billing" />
                    </>
                  )}
                  {roleInfo.role === 'org_admin' && (
                    <>
                      <PermissionBadge permission="printers:create" />
                      <PermissionBadge permission="agents:manage" />
                      <PermissionBadge permission="users:create" />
                    </>
                  )}
                  {roleInfo.role === 'org_user' && (
                    <>
                      <PermissionBadge permission="jobs:create" />
                      <PermissionBadge permission="printers:read" />
                      <PermissionBadge permission="reports:read" />
                    </>
                  )}
                  {roleInfo.role === 'org_viewer' && (
                    <>
                      <PermissionBadge permission="printers:read" />
                      <PermissionBadge permission="jobs:read" />
                      <PermissionBadge permission="reports:read" />
                    </>
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};

export default PermissionsPage;

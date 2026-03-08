import { useMemo } from 'react';
import { ShieldIcon, CheckIcon, XCircleIcon } from './icons';

export interface Permission {
  name: string;
  resource: string;
  action: string;
  description: string;
  granted: boolean;
}

export interface PermissionGroup {
  name: string;
  description: string;
  permissions: Permission[];
}

interface RolePermissionsProps {
  role?: string;
  groups?: PermissionGroup[];
  compact?: boolean;
  showOnlyGranted?: boolean;
}

const roleDescriptions: Record<string, string> = {
  platform_admin: 'Full system access including all organizations, billing, and certificates.',
  org_admin: 'Organization administrator with full access to manage organization resources.',
  org_user: 'Standard organization user who can submit print jobs and view their own resources.',
  org_viewer: 'Read-only access to view organization resources without making changes.',
  admin: 'Legacy platform admin role (use platform_admin).',
  user: 'Legacy user role (use org_user).',
  viewer: 'Legacy viewer role (use org_viewer).',
};

export const RolePermissions = ({ role = 'org_user', groups, compact = false, showOnlyGranted = false }: RolePermissionsProps) => {
  const roleInfo = useMemo(() => ({
    name: role.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase()),
    description: roleDescriptions[role] || 'Custom role with specific permissions.',
  }), [role]);

  const filteredGroups = useMemo(() => {
    if (!groups) return [];
    if (!showOnlyGranted) return groups;
    return groups
      .map(g => ({
        ...g,
        permissions: g.permissions.filter(p => p.granted),
      }))
      .filter(g => g.permissions.length > 0);
  }, [groups, showOnlyGranted]);

  const totalPermissions = useMemo(() => {
    return groups?.reduce((sum, g) => sum + g.permissions.length, 0) ?? 0;
  }, [groups]);

  const grantedPermissions = useMemo(() => {
    return groups?.reduce((sum, g) => sum + g.permissions.filter(p => p.granted).length, 0) ?? 0;
  }, [groups]);

  if (compact) {
    return (
      <div className="text-sm text-gray-600 dark:text-gray-400">
        <span className="font-medium text-gray-900 dark:text-gray-100">{roleInfo.name}</span>
        <span className="mx-1">·</span>
        {grantedPermissions} of {totalPermissions} permissions
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Role Header */}
      <div className="flex items-start gap-4 p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg border border-blue-200 dark:border-blue-800">
        <div className="p-2 bg-blue-100 dark:bg-blue-900/40 rounded-lg text-blue-600 dark:text-blue-400">
          <ShieldIcon className="w-6 h-6" />
        </div>
        <div className="flex-1">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">{roleInfo.name}</h3>
          <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">{roleInfo.description}</p>
          <div className="mt-2 flex items-center gap-4 text-sm">
            <span className="text-gray-600 dark:text-gray-400">
              <span className="font-semibold text-gray-900 dark:text-gray-100">{grantedPermissions}</span> of {totalPermissions} permissions granted
            </span>
            <div className="flex-1 max-w-xs bg-gray-200 dark:bg-gray-700 rounded-full h-2">
              <div
                className="bg-blue-600 h-2 rounded-full transition-all"
                style={{ width: `${totalPermissions > 0 ? (grantedPermissions / totalPermissions) * 100 : 0}%` }}
              />
            </div>
          </div>
        </div>
      </div>

      {/* Permission Groups */}
      <div className="space-y-4">
        {filteredGroups.map((group) => (
          <div key={group.name} className="border border-gray-200 dark:border-gray-700 rounded-lg overflow-hidden">
            <div className="px-4 py-3 bg-gray-50 dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700">
              <h4 className="font-medium text-gray-900 dark:text-gray-100">{group.name}</h4>
              <p className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">{group.description}</p>
            </div>
            <div className="divide-y divide-gray-200 dark:divide-gray-700">
              {group.permissions.map((perm) => (
                <div
                  key={perm.name}
                  className="px-4 py-3 flex items-start gap-3 hover:bg-gray-50 dark:hover:bg-gray-800/50"
                >
                  <div className={`p-1 rounded ${perm.granted ? 'bg-green-100 dark:bg-green-900/30 text-green-600 dark:text-green-400' : 'bg-gray-100 dark:bg-gray-800 text-gray-400 dark:text-gray-600'}`}>
                    {perm.granted ? <CheckIcon className="w-4 h-4" /> : <XCircleIcon className="w-4 h-4" />}
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <span className={`text-sm font-medium ${perm.granted ? 'text-gray-900 dark:text-gray-100' : 'text-gray-500 dark:text-gray-500'}`}>
                        {perm.name}
                      </span>
                      <span className="text-xs px-2 py-0.5 rounded bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400">
                        {perm.resource}:{perm.action}
                      </span>
                    </div>
                    <p className={`text-xs mt-1 ${perm.granted ? 'text-gray-600 dark:text-gray-400' : 'text-gray-400 dark:text-gray-500'}`}>
                      {perm.description}
                    </p>
                  </div>
                </div>
              ))}
            </div>
          </div>
        ))}
      </div>

      {/* Legend */}
      <div className="flex items-center gap-6 text-sm text-gray-600 dark:text-gray-400">
        <div className="flex items-center gap-2">
          <div className="p-1 rounded bg-green-100 dark:bg-green-900/30 text-green-600 dark:text-green-400">
            <CheckIcon className="w-4 h-4" />
          </div>
          <span>Granted</span>
        </div>
        <div className="flex items-center gap-2">
          <div className="p-1 rounded bg-gray-100 dark:bg-gray-800 text-gray-400 dark:text-gray-600">
            <XCircleIcon className="w-4 h-4" />
          </div>
          <span>Not Granted</span>
        </div>
      </div>
    </div>
  );
};

interface PermissionBadgeProps {
  permission: string;
  granted?: boolean;
  size?: 'sm' | 'md';
}

export const PermissionBadge = ({ permission, granted = true, size = 'sm' }: PermissionBadgeProps) => {
  const [resource, action] = permission.split(':');

  const sizeClasses = size === 'sm' ? 'text-xs px-2 py-0.5' : 'text-sm px-3 py-1';

  return (
    <span
      className={`inline-flex items-center gap-1.5 rounded-full ${sizeClasses} ${
        granted
          ? 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400 border border-green-200 dark:border-green-800'
          : 'bg-gray-100 dark:bg-gray-800 text-gray-500 dark:text-gray-500 border border-gray-200 dark:border-gray-700'
      }`}
    >
      {granted && <CheckIcon className="w-3 h-3" />}
      <span className="font-medium">{resource}</span>
      <span className="text-gray-400">:</span>
      <span>{action}</span>
    </span>
  );
};

interface RoleComparisonProps {
  roles: string[];
  permissions: string[];
}

export const RoleComparison = ({ roles, permissions }: RoleComparisonProps) => {
  // This would typically fetch from API, using mock data for now
  const rolePerms: Record<string, Set<string>> = {
    platform_admin: new Set(permissions), // All permissions
    org_admin: new Set(permissions.filter(p => !p.includes('organizations:'))),
    org_user: new Set(permissions.filter(p => p.includes('read') || p.includes('create') || p.includes('list'))),
    org_viewer: new Set(permissions.filter(p => p.includes('read') || p.includes('list'))),
  };

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-gray-200 dark:border-gray-700">
            <th className="px-4 py-3 text-left font-medium text-gray-900 dark:text-gray-100">Permission</th>
            {roles.map(role => (
              <th key={role} className="px-4 py-3 text-center font-medium text-gray-900 dark:text-gray-100">
                {role.replace(/_/g, ' ')}
              </th>
            ))}
          </tr>
        </thead>
        <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
          {permissions.map(perm => (
            <tr key={perm} className="hover:bg-gray-50 dark:hover:bg-gray-800/50">
              <td className="px-4 py-3 text-gray-700 dark:text-gray-300 font-mono text-xs">{perm}</td>
              {roles.map(role => {
                const hasPermission = rolePerms[role]?.has(perm);
                return (
                  <td key={role} className="px-4 py-3 text-center">
                    {hasPermission ? (
                      <CheckIcon className="w-5 h-5 text-green-600 dark:text-green-400 mx-auto" />
                    ) : (
                      <span className="text-gray-300 dark:text-gray-700">—</span>
                    )}
                  </td>
                );
              })}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};

/**
 * OrgUserCard - Component displaying and managing organization users
 *
 * Features:
 * - List of organization members with roles
 * - Role management
 * - User removal
 * - Invite new members
 */

import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import type { OrganizationUser, OrgRole } from '@/types';
import { orgUsersApi } from '@/services/platformAdminApi';

interface OrgUserCardProps {
  users: OrganizationUser[];
  organizationId: string;
  isLoading?: boolean;
  currentUserId?: string;
  canManage?: boolean;
  onInvite?: () => void;
  className?: string;
}

const RoleBadge = ({ role }: { role: OrgRole }) => {
  const styles: Record<OrgRole, string> = {
    owner: 'bg-purple-100 dark:bg-purple-900/30 text-purple-700 dark:text-purple-300',
    admin: 'bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300',
    member: 'bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300',
    viewer: 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300',
  };

  const labels: Record<OrgRole, string> = {
    owner: 'Owner',
    admin: 'Admin',
    member: 'Member',
    viewer: 'Viewer',
  };

  return (
    <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${styles[role]}`}>
      {labels[role]}
    </span>
  );
};

const StatusBadge = ({ status }: { status: string }) => {
  const styles: Record<string, string> = {
    active: 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300',
    pending: 'bg-yellow-100 dark:bg-yellow-900/30 text-yellow-700 dark:text-yellow-300',
    invited: 'bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300',
    deactivated: 'bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-300',
  };

  const label = status.charAt(0).toUpperCase() + status.slice(1);

  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${styles[status] || styles.pending}`}>
      {label}
    </span>
  );
};

const UserAvatar = ({ name, email }: { name: string; email: string }) => {
  const initials = name
    .split(' ')
    .map(n => n.charAt(0).toUpperCase())
    .join('')
    .slice(0, 2);

  const colors = [
    'from-blue-500 to-cyan-500',
    'from-purple-500 to-pink-500',
    'from-green-500 to-emerald-500',
    'from-orange-500 to-red-500',
    'from-indigo-500 to-purple-500',
  ];

  const colorIndex = (name.charCodeAt(0) + email.charCodeAt(0)) % colors.length;

  return (
    <div className={`w-10 h-10 bg-gradient-to-br ${colors[colorIndex]} rounded-full flex items-center justify-center text-white font-semibold text-sm`}>
      {initials}
    </div>
  );
};

// Icons
const MoreIcon = () => (
  <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 5v.01M12 12v.01M12 19v.01M12 6a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2z" />
  </svg>
);

const TrashIcon = () => (
  <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
  </svg>
);

export const OrgUserCard = ({
  users,
  organizationId,
  isLoading = false,
  currentUserId,
  canManage = true,
  onInvite,
  className = '',
}: OrgUserCardProps) => {
  const queryClient = useQueryClient();
  const [actionMenuOpen, setActionMenuOpen] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState('');

  const updateRoleMutation = useMutation({
    mutationFn: ({ userId, role }: { userId: string; role: OrgRole }) =>
      orgUsersApi.updateUserRole(organizationId, userId, role),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['organizationUsers', organizationId] });
      setActionMenuOpen(null);
    },
  });

  const removeUserMutation = useMutation({
    mutationFn: (userId: string) =>
      orgUsersApi.removeOrganizationUser(organizationId, userId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['organizationUsers', organizationId] });
      setActionMenuOpen(null);
    },
  });

  const handleRoleChange = (userId: string, newRole: OrgRole) => {
    updateRoleMutation.mutate({ userId, role: newRole });
  };

  const handleRemoveUser = (userId: string, userName: string) => {
    if (confirm(`Are you sure you want to remove ${userName} from this organization?`)) {
      removeUserMutation.mutate(userId);
    }
  };

  const filteredUsers = users.filter(user => {
    if (!searchQuery) return true;
    const query = searchQuery.toLowerCase();
    return (
      user.user?.name?.toLowerCase().includes(query) ||
      user.user?.email?.toLowerCase().includes(query)
    );
  });

  const roleCounts = users.reduce((acc, user) => {
    acc[user.role] = (acc[user.role] || 0) + 1;
    return acc;
  }, {} as Record<OrgRole, number>);

  if (isLoading) {
    return (
      <div className={`bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 ${className}`}>
        <div className="p-6">
          <div className="animate-pulse space-y-4">
            <div className="h-6 bg-gray-200 dark:bg-gray-700 rounded w-1/4" />
            <div className="space-y-3">
              {[1, 2, 3].map(i => (
                <div key={i} className="h-16 bg-gray-200 dark:bg-gray-700 rounded" />
              ))}
            </div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className={`bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 ${className}`}>
      {/* Header */}
      <div className="p-6 border-b border-gray-200 dark:border-gray-700">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
              Organization Members
            </h2>
            <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
              {users.length} {users.length === 1 ? 'member' : 'members'}
            </p>
          </div>
          {canManage && onInvite && (
            <button
              onClick={onInvite}
              className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors font-medium text-sm"
            >
              Invite Member
            </button>
          )}
        </div>

        {/* Role Counts */}
        <div className="flex gap-4 mt-4">
          {(['owner', 'admin', 'member', 'viewer'] as OrgRole[]).map(role => (
            roleCounts[role] > 0 ? (
              <div key={role} className="flex items-center gap-2">
                <RoleBadge role={role} />
                <span className="text-sm text-gray-600 dark:text-gray-400">{roleCounts[role]}</span>
              </div>
            ) : null
          ))}
        </div>
      </div>

      {/* Search */}
      <div className="px-6 pt-4">
        <div className="relative">
          <svg
            className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
          <input
            type="text"
            placeholder="Search members..."
            value={searchQuery}
            onChange={e => setSearchQuery(e.target.value)}
            className="w-full pl-10 pr-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-gray-100 text-sm"
          />
        </div>
      </div>

      {/* Users List */}
      <div className="p-6 pt-4">
        {filteredUsers.length === 0 ? (
          <div className="text-center py-8">
            <p className="text-gray-500 dark:text-gray-400">
              {searchQuery ? 'No members found matching your search' : 'No members in this organization'}
            </p>
          </div>
        ) : (
          <div className="space-y-2">
            {filteredUsers.map(member => (
              <div
                key={member.id}
                className="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-700/50 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
              >
                <div className="flex items-center gap-3">
                  {member.user ? (
                    <>
                      <UserAvatar name={member.user.name} email={member.user.email} />
                      <div>
                        <p className="text-sm font-medium text-gray-900 dark:text-gray-100 flex items-center gap-2">
                          {member.user.name}
                          {member.userId === currentUserId && (
                            <span className="text-xs text-gray-500 dark:text-gray-400">(You)</span>
                          )}
                        </p>
                        <p className="text-xs text-gray-500 dark:text-gray-400">{member.user.email}</p>
                      </div>
                    </>
                  ) : (
                    <div>
                      <p className="text-sm font-medium text-gray-900 dark:text-gray-100">
                        {member.userId}
                      </p>
                    </div>
                  )}
                </div>

                <div className="flex items-center gap-3">
                  <StatusBadge status={member.status} />
                  <RoleBadge role={member.role} />

                  {canManage && member.userId !== currentUserId && (
                    <div className="relative">
                      <button
                        onClick={() => setActionMenuOpen(actionMenuOpen === member.id ? null : member.id)}
                        className="p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 rounded"
                      >
                        <MoreIcon />
                      </button>

                      {actionMenuOpen === member.id && (
                        <>
                          <div
                            className="fixed inset-0 z-10"
                            onClick={() => setActionMenuOpen(null)}
                          />
                          <div className="absolute right-0 top-full mt-1 w-48 bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 z-20 py-1">
                            {/* Role Change */}
                            <div className="px-3 py-2 text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">
                              Change Role
                            </div>
                            {(['owner', 'admin', 'member', 'viewer'] as OrgRole[]).map(role => (
                              <button
                                key={role}
                                onClick={() => handleRoleChange(member.userId, role)}
                                disabled={updateRoleMutation.isPending}
                                className={`w-full text-left px-3 py-2 text-sm hover:bg-gray-100 dark:hover:bg-gray-700 ${
                                  member.role === role
                                    ? 'text-blue-600 dark:text-blue-400 font-medium'
                                    : 'text-gray-700 dark:text-gray-300'
                                }`}
                              >
                                {role.charAt(0).toUpperCase() + role.slice(1)}
                              </button>
                            ))}

                            <div className="border-t border-gray-200 dark:border-gray-700 my-1" />

                            {/* Remove */}
                            <button
                              onClick={() => handleRemoveUser(member.userId, member.user?.name || member.userId)}
                              disabled={removeUserMutation.isPending}
                              className="w-full flex items-center gap-2 px-3 py-2 text-sm text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20"
                            >
                              <TrashIcon />
                              Remove Member
                            </button>
                          </div>
                        </>
                      )}
                    </div>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};

export default OrgUserCard;

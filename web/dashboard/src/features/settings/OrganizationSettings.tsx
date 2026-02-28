import { useState, useEffect, FormEvent } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useAuth } from '@/hooks/useAuth';
import { getOrganization, updateOrganization, getOrganizationMembers, inviteMember, updateMemberRole, removeMember } from './api';
import { useToast } from './useToast';
import type { InviteMemberRequest, UserRole } from './types';

interface OrganizationSettingsProps {
  className?: string;
}

interface InviteModalProps {
  isOpen: boolean;
  onClose: () => void;
  onInvite: (email: string, role: UserRole) => void;
  isLoading: boolean;
}

const InviteModal = ({ isOpen, onClose, onInvite, isLoading }: InviteModalProps) => {
  const [email, setEmail] = useState('');
  const [role, setRole] = useState<UserRole>('user');

  if (!isOpen) return null;

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault();
    if (email) {
      onInvite(email, role);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-xl max-w-md w-full mx-4 p-6">
        <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-2">
          Invite Team Member
        </h3>
        <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
          Send an invitation to join your organization
        </p>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label
              htmlFor="invite-email"
              className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
            >
              Email Address
            </label>
            <input
              id="invite-email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-gray-100"
              placeholder="colleague@example.com"
              required
            />
          </div>

          <div>
            <label
              htmlFor="invite-role"
              className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
            >
              Role
            </label>
            <select
              id="invite-role"
              value={role}
              onChange={(e) => setRole(e.target.value as UserRole)}
              className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-gray-100"
            >
              <option value="user">User - Can view and print</option>
              <option value="admin">Admin - Can manage printers and jobs</option>
              <option value="owner">Owner - Full access to all settings</option>
            </select>
          </div>

          <div className="flex justify-end gap-3 pt-2">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors font-medium"
              disabled={isLoading}
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={isLoading || !email}
              className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors font-medium disabled:opacity-50"
            >
              {isLoading ? 'Sending...' : 'Send Invite'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

const RoleBadge = ({ role }: { role: UserRole }) => {
  const styles = {
    owner: 'bg-purple-100 dark:bg-purple-900/30 text-purple-700 dark:text-purple-300',
    admin: 'bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300',
    user: 'bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300',
  };

  return (
    <span
      className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${styles[role]}`}
    >
      {role.charAt(0).toUpperCase() + role.slice(1)}
    </span>
  );
};

export const OrganizationSettings = ({ className = '' }: OrganizationSettingsProps) => {
  const { user } = useAuth();
  const queryClient = useQueryClient();
  const { showSuccess, showError } = useToast();

  const [showInviteModal, setShowInviteModal] = useState(false);
  const [orgName, setOrgName] = useState('');
  const [isEditingOrg, setIsEditingOrg] = useState(false);

  // Fetch organization
  const { data: organization, isLoading: orgLoading } = useQuery({
    queryKey: ['organization'],
    queryFn: getOrganization,
  });

  // Fetch members
  const { data: members = [], isLoading: membersLoading, refetch: refetchMembers } = useQuery({
    queryKey: ['organization-members'],
    queryFn: getOrganizationMembers,
  });

  useEffect(() => {
    if (organization) {
      setOrgName(organization.name);
    }
  }, [organization]);

  const updateOrgMutation = useMutation({
    mutationFn: (data: { name: string }) => updateOrganization({ name: data.name }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['organization'] });
      showSuccess('Organization updated successfully');
      setIsEditingOrg(false);
    },
    onError: (error: Error) => {
      showError(error.message || 'Failed to update organization');
    },
  });

  const inviteMemberMutation = useMutation({
    mutationFn: (data: InviteMemberRequest) => inviteMember(data),
    onSuccess: () => {
      showSuccess('Invitation sent successfully');
      setShowInviteModal(false);
      refetchMembers();
    },
    onError: (error: Error) => {
      showError(error.message || 'Failed to send invitation');
    },
  });

  const updateRoleMutation = useMutation({
    mutationFn: ({ memberId, role }: { memberId: string; role: string }) =>
      updateMemberRole(memberId, role),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['organization-members'] });
      showSuccess('Member role updated successfully');
    },
    onError: (error: Error) => {
      showError(error.message || 'Failed to update member role');
    },
  });

  const removeMemberMutation = useMutation({
    mutationFn: (memberId: string) => removeMember(memberId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['organization-members'] });
      showSuccess('Member removed successfully');
    },
    onError: (error: Error) => {
      showError(error.message || 'Failed to remove member');
    },
  });

  const handleOrgSave = () => {
    if (orgName.trim()) {
      updateOrgMutation.mutate({ name: orgName });
    }
  };

  const handleOrgCancel = () => {
    if (organization) {
      setOrgName(organization.name);
    }
    setIsEditingOrg(false);
  };

  const handleInvite = (email: string, role: UserRole) => {
    inviteMemberMutation.mutate({ email, role });
  };

  const handleRoleChange = (memberId: string, newRole: string) => {
    updateRoleMutation.mutate({ memberId, role: newRole });
  };

  const handleRemoveMember = (memberId: string) => {
    if (confirm('Are you sure you want to remove this member from the organization?')) {
      removeMemberMutation.mutate(memberId);
    }
  };

  const canManageMembers = user?.role === 'admin' || user?.role === 'owner';
  const memberCount = members.length;
  const maxMembers = organization?.maxUsers || 0;

  return (
    <div className={`space-y-6 ${className}`}>
      {/* Organization Info */}
      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700">
        <div className="p-6 border-b border-gray-200 dark:border-gray-700">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
            Organization Information
          </h2>
          <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
            Manage your organization details
          </p>
        </div>

        <div className="p-6 space-y-6">
          {orgLoading ? (
            <div className="animate-pulse space-y-4">
              <div className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-1/3" />
              <div className="h-10 bg-gray-200 dark:bg-gray-700 rounded" />
            </div>
          ) : (
            <>
              <div>
                <label
                  htmlFor="org-name"
                  className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
                >
                  Organization Name
                </label>
                {isEditingOrg ? (
                  <div className="flex gap-3">
                    <input
                      id="org-name"
                      type="text"
                      value={orgName}
                      onChange={(e) => setOrgName(e.target.value)}
                      className="flex-1 px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-gray-100"
                    />
                    <button
                      onClick={handleOrgSave}
                      disabled={updateOrgMutation.isPending}
                      className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors font-medium disabled:opacity-50"
                    >
                      {updateOrgMutation.isPending ? 'Saving...' : 'Save'}
                    </button>
                    <button
                      onClick={handleOrgCancel}
                      className="px-4 py-2 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors font-medium"
                    >
                      Cancel
                    </button>
                  </div>
                ) : (
                  <div className="flex items-center justify-between">
                    <p className="text-lg text-gray-900 dark:text-gray-100">{organization?.name}</p>
                    {canManageMembers && (
                      <button
                        onClick={() => setIsEditingOrg(true)}
                        className="text-blue-600 dark:text-blue-400 hover:underline text-sm font-medium"
                      >
                        Edit
                      </button>
                    )}
                  </div>
                )}
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div className="p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
                  <p className="text-xs text-gray-500 dark:text-gray-400 uppercase tracking-wide">
                    Plan
                  </p>
                  <p className="text-lg font-semibold text-gray-900 dark:text-gray-100 capitalize">
                    {organization?.plan}
                  </p>
                </div>
                <div className="p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
                  <p className="text-xs text-gray-500 dark:text-gray-400 uppercase tracking-wide">
                    Members
                  </p>
                  <p className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                    {memberCount} / {maxMembers}
                  </p>
                </div>
                <div className="p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
                  <p className="text-xs text-gray-500 dark:text-gray-400 uppercase tracking-wide">
                    Printers
                  </p>
                  <p className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                    {organization?.maxPrinters || 0}
                  </p>
                </div>
                <div className="p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
                  <p className="text-xs text-gray-500 dark:text-gray-400 uppercase tracking-wide">
                    Created
                  </p>
                  <p className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                    {organization?.createdAt
                      ? new Date(organization.createdAt).toLocaleDateString()
                      : 'N/A'}
                  </p>
                </div>
              </div>
            </>
          )}
        </div>
      </div>

      {/* Team Members */}
      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700">
        <div className="p-6 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
          <div>
            <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
              Team Members
            </h2>
            <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
              Manage who has access to your organization
            </p>
          </div>
          {canManageMembers && (
            <button
              onClick={() => setShowInviteModal(true)}
              className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors font-medium text-sm"
            >
              Invite Member
            </button>
          )}
        </div>

        <div className="p-6">
          {membersLoading ? (
            <div className="space-y-4">
              {[1, 2, 3].map((i) => (
                <div key={i} className="animate-pulse flex items-center gap-4">
                  <div className="w-10 h-10 bg-gray-200 dark:bg-gray-700 rounded-full" />
                  <div className="flex-1 space-y-2">
                    <div className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-1/3" />
                    <div className="h-3 bg-gray-200 dark:bg-gray-700 rounded w-1/4" />
                  </div>
                </div>
              ))}
            </div>
          ) : members.length === 0 ? (
            <div className="text-center py-8">
              <p className="text-gray-500 dark:text-gray-400">No members found</p>
            </div>
          ) : (
            <div className="space-y-4">
              {members.map((member) => (
                <div
                  key={member.id}
                  className="flex items-center justify-between p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg"
                >
                  <div className="flex items-center gap-4">
                    <div className="w-10 h-10 bg-gradient-to-br from-blue-500 to-cyan-500 rounded-full flex items-center justify-center text-white font-semibold">
                      {member.name
                        .split(' ')
                        .map((n) => n.charAt(0).toUpperCase())
                        .join('')
                        .slice(0, 2)}
                    </div>
                    <div>
                      <p className="text-sm font-medium text-gray-900 dark:text-gray-100 flex items-center gap-2">
                        {member.name}
                        {member.id === user?.id && (
                          <span className="text-xs text-gray-500 dark:text-gray-400">(You)</span>
                        )}
                      </p>
                      <p className="text-xs text-gray-500 dark:text-gray-400">{member.email}</p>
                    </div>
                  </div>

                  <div className="flex items-center gap-3">
                    <RoleBadge role={member.role} />

                    {canManageMembers && member.id !== user?.id && (
                      <div className="flex items-center gap-2">
                        <select
                          value={member.role}
                          onChange={(e) => handleRoleChange(member.id, e.target.value)}
                          className="text-xs border border-gray-300 dark:border-gray-600 rounded px-2 py-1 bg-white dark:bg-gray-700 dark:text-gray-100 focus:ring-1 focus:ring-blue-500"
                          disabled={updateRoleMutation.isPending}
                        >
                          <option value="user">User</option>
                          <option value="admin">Admin</option>
                          <option value="owner">Owner</option>
                        </select>

                        <button
                          onClick={() => handleRemoveMember(member.id)}
                          className="text-red-600 dark:text-red-400 hover:text-red-700 dark:hover:text-red-300 p-1"
                          title="Remove member"
                          disabled={removeMemberMutation.isPending}
                        >
                          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path
                              strokeLinecap="round"
                              strokeLinejoin="round"
                              strokeWidth={2}
                              d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                            />
                          </svg>
                        </button>
                      </div>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Invite Modal */}
      <InviteModal
        isOpen={showInviteModal}
        onClose={() => setShowInviteModal(false)}
        onInvite={handleInvite}
        isLoading={inviteMemberMutation.isPending}
      />
    </div>
  );
};

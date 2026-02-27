import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useAuth } from '@/hooks/useAuth';
import { organizationApi, userApi, printersApi } from '@/services/api';
import { PlusIcon, TrashIcon, EditIcon, CheckIcon, AlertIcon } from './icons';

export const Organization = () => {
  const { user } = useAuth();
  const queryClient = useQueryClient();
  const [activeTab, setActiveTab] = useState<'overview' | 'users' | 'printers' | 'invites'>('overview');
  const [inviteEmail, setInviteEmail] = useState('');
  const [inviteRole, setInviteRole] = useState<'user' | 'admin'>('user');
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  const { data: organization } = useQuery({
    queryKey: ['organization'],
    queryFn: () => organizationApi.get(),
  });

  const { data: users, isLoading: usersLoading } = useQuery({
    queryKey: ['organization', 'users'],
    queryFn: () => organizationApi.getUsers(),
  });

  const { data: invitations } = useQuery({
    queryKey: ['organization', 'invitations'],
    queryFn: () => organizationApi.getInvitations(),
  });

  const { data: printers } = useQuery({
    queryKey: ['printers'],
    queryFn: () => printersApi.list(),
  });

  const inviteMutation = useMutation({
    mutationFn: (data: { email: string; role: 'user' | 'admin' }) =>
      organizationApi.inviteUser(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['organization', 'invitations'] });
      setInviteEmail('');
      showMessage('Invitation sent successfully', 'success');
    },
    onError: () => {
      showMessage('Failed to send invitation', 'error');
    },
  });

  const removeUserMutation = useMutation({
    mutationFn: (userId: string) => organizationApi.removeUser(userId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['organization', 'users'] });
      showMessage('User removed successfully', 'success');
    },
    onError: () => {
      showMessage('Failed to remove user', 'error');
    },
  });

  const updateRoleMutation = useMutation({
    mutationFn: ({ userId, role }: { userId: string; role: string }) =>
      organizationApi.updateUserRole(userId, role),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['organization', 'users'] });
      showMessage('User role updated', 'success');
    },
    onError: () => {
      showMessage('Failed to update user role', 'error');
    },
  });

  const cancelInviteMutation = useMutation({
    mutationFn: (invitationId: string) => organizationApi.cancelInvitation(invitationId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['organization', 'invitations'] });
      showMessage('Invitation cancelled', 'success');
    },
  });

  const showMessage = (text: string, type: 'success' | 'error') => {
    setMessage({ text, type });
    setTimeout(() => setMessage(null), 3000);
  };

  const handleInvite = (e: React.FormEvent) => {
    e.preventDefault();
    inviteMutation.mutate({ email: inviteEmail, role: inviteRole });
  };

  const adminUsers = users?.filter((u) => u.role === 'admin' || u.role === 'owner').length || 0;
  const regularUsers = users?.filter((u) => u.role === 'user').length || 0;
  const onlinePrinters = printers?.filter((p) => p.isOnline).length || 0;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-900 dark:text-gray-100">Organization</h1>
          <p className="text-gray-600 dark:text-gray-400 mt-1">
            Manage your organization settings and members
          </p>
        </div>
      </div>

      {/* Organization Info Card */}
      <div className="bg-gradient-to-r from-blue-600 to-cyan-600 rounded-xl p-6 text-white">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-2xl font-bold">{organization?.name}</h2>
            <p className="text-blue-100 mt-1">
              Plan:{' '}
              <span className="font-semibold capitalize">{organization?.plan}</span>
            </p>
          </div>
          <button className="px-4 py-2 bg-white/20 hover:bg-white/30 rounded-lg font-medium transition-colors">
            Edit Organization
          </button>
        </div>
        <div className="grid grid-cols-3 gap-4 mt-6 pt-6 border-t border-white/20">
          <div>
            <p className="text-3xl font-bold">{users?.length || 0}</p>
            <p className="text-blue-100 text-sm">Total Users</p>
          </div>
          <div>
            <p className="text-3xl font-bold">{printers?.length || 0}</p>
            <p className="text-blue-100 text-sm">Total Printers</p>
          </div>
          <div>
            <p className="text-3xl font-bold">{onlinePrinters}</p>
            <p className="text-blue-100 text-sm">Online Printers</p>
          </div>
        </div>
      </div>

      {/* Message */}
      {message && (
        <div
          className={`p-4 rounded-lg flex items-center gap-3 ${
            message.type === 'success'
              ? 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400'
              : 'bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400'
          }`}
        >
          {message.type === 'success' ? (
            <CheckIcon className="w-5 h-5 flex-shrink-0" />
          ) : (
            <AlertIcon className="w-5 h-5 flex-shrink-0" />
          )}
          <span>{message.text}</span>
        </div>
      )}

      {/* Tabs */}
      <div className="border-b border-gray-200 dark:border-gray-700">
        <nav className="flex gap-8">
          {[
            { value: 'overview', label: 'Overview' },
            { value: 'users', label: 'Users' },
            { value: 'printers', label: 'Printers' },
            { value: 'invites', label: 'Invitations' },
          ].map((tab) => (
            <button
              key={tab.value}
              onClick={() => setActiveTab(tab.value as typeof activeTab)}
              className={`
                py-4 px-1 border-b-2 font-medium text-sm transition-colors
                ${activeTab === tab.value
                  ? 'border-blue-500 text-blue-600 dark:text-blue-400'
                  : 'border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300'
                }
              `}
            >
              {tab.label}
            </button>
          ))}
        </nav>
      </div>

      {/* Overview Tab */}
      {activeTab === 'overview' && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <div className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700">
            <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">
              Plan Details
            </h3>
            <div className="space-y-4">
              <div className="flex justify-between">
                <span className="text-gray-600 dark:text-gray-400">Current Plan</span>
                <span className="font-medium text-gray-900 dark:text-gray-100 capitalize">
                  {organization?.plan}
                </span>
              </div>
              <div className="flex justify-between">
                <span className="text-gray-600 dark:text-gray-400">Max Users</span>
                <span className="font-medium text-gray-900 dark:text-gray-100">
                  {users?.length || 0} / {organization?.maxUsers || 10}
                </span>
              </div>
              <div className="flex justify-between">
                <span className="text-gray-600 dark:text-gray-400">Max Printers</span>
                <span className="font-medium text-gray-900 dark:text-gray-100">
                  {printers?.length || 0} / {organization?.maxPrinters || 5}
                </span>
              </div>
            </div>
            <button className="w-full mt-4 py-2 px-4 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors font-medium">
              Upgrade Plan
            </button>
          </div>

          <div className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700">
            <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">
              User Distribution
            </h3>
            <div className="space-y-4">
              <div>
                <div className="flex justify-between text-sm mb-1">
                  <span className="text-gray-600 dark:text-gray-400">Admins</span>
                  <span className="font-medium text-gray-900 dark:text-gray-100">
                    {adminUsers}
                  </span>
                </div>
                <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
                  <div
                    className="bg-blue-600 h-2 rounded-full"
                    style={{ width: `${(adminUsers / (users?.length || 1)) * 100}%` }}
                  />
                </div>
              </div>
              <div>
                <div className="flex justify-between text-sm mb-1">
                  <span className="text-gray-600 dark:text-gray-400">Regular Users</span>
                  <span className="font-medium text-gray-900 dark:text-gray-100">
                    {regularUsers}
                  </span>
                </div>
                <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
                  <div
                    className="bg-green-600 h-2 rounded-full"
                    style={{ width: `${(regularUsers / (users?.length || 1)) * 100}%` }}
                  />
                </div>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Users Tab */}
      {activeTab === 'users' && (
        <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700">
          <div className="p-6 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
              Team Members ({users?.length || 0})
            </h2>
            <button
              onClick={() => setActiveTab('invites')}
              className="inline-flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors font-medium"
            >
              <PlusIcon className="w-5 h-5" />
              Invite User
            </button>
          </div>
          {usersLoading ? (
            <div className="p-6 text-center text-gray-500 dark:text-gray-400">
              Loading users...
            </div>
          ) : (
            <div className="divide-y divide-gray-200 dark:divide-gray-700">
              {users?.map((u) => (
                <div key={u.id} className="p-4 hover:bg-gray-50 dark:hover:bg-gray-700/50">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-4">
                      <div className="w-10 h-10 bg-gradient-to-br from-blue-500 to-cyan-500 rounded-full flex items-center justify-center text-white font-semibold">
                        {u.name?.charAt(0).toUpperCase() || u.email.charAt(0).toUpperCase()}
                      </div>
                      <div>
                        <p className="text-sm font-medium text-gray-900 dark:text-gray-100">
                          {u.name}
                        </p>
                        <p className="text-sm text-gray-500 dark:text-gray-400">{u.email}</p>
                      </div>
                    </div>
                    <div className="flex items-center gap-4">
                      <select
                        value={u.role}
                        disabled={u.id === user?.id || u.role === 'owner'}
                        onChange={(e) => updateRoleMutation.mutate({ userId: u.id, role: e.target.value })}
                        className="px-3 py-1 border border-gray-300 dark:border-gray-600 rounded-lg text-sm focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:text-gray-100 disabled:opacity-50"
                      >
                        <option value="user">User</option>
                        <option value="admin">Admin</option>
                        <option value="owner">Owner</option>
                      </select>
                      {u.id !== user?.id && u.role !== 'owner' && (
                        <button
                          onClick={() => removeUserMutation.mutate(u.id)}
                          className="p-2 text-gray-400 hover:text-red-600 dark:hover:text-red-400 transition-colors"
                          title="Remove user"
                        >
                          <TrashIcon className="w-5 h-5" />
                        </button>
                      )}
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Printers Tab */}
      {activeTab === 'printers' && (
        <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700">
          <div className="p-6 border-b border-gray-200 dark:border-gray-700">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
              Organization Printers ({printers?.length || 0})
            </h2>
          </div>
          <div className="divide-y divide-gray-200 dark:divide-gray-700">
            {printers?.map((printer) => (
              <div key={printer.id} className="p-4 hover:bg-gray-50 dark:hover:bg-gray-700/50">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-4">
                    <div
                      className={`p-2 rounded-lg ${
                        printer.isOnline
                          ? 'bg-green-100 dark:bg-green-900/30 text-green-600 dark:text-green-400'
                          : 'bg-gray-100 dark:bg-gray-700 text-gray-400'
                      }`}
                    >
                      <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          strokeWidth={2}
                          d="M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2h2m2 4h6a2 2 0 002-2v-4a2 2 0 00-2-2H9a2 2 0 00-2 2v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z"
                        />
                      </svg>
                    </div>
                    <div>
                      <p className="text-sm font-medium text-gray-900 dark:text-gray-100">
                        {printer.name}
                      </p>
                      <p className="text-sm text-gray-500 dark:text-gray-400">
                        {printer.agentId} • {printer.type}
                      </p>
                    </div>
                  </div>
                  <div className="flex items-center gap-3">
                    <span
                      className={`text-xs px-2 py-1 rounded-full ${
                        printer.isActive
                          ? 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300'
                          : 'bg-gray-100 dark:bg-gray-700 text-gray-500 dark:text-gray-400'
                      }`}
                    >
                      {printer.isActive ? 'Active' : 'Disabled'}
                    </span>
                    <button
                      className="p-2 text-gray-400 hover:text-blue-600 dark:hover:text-blue-400 transition-colors"
                    >
                      <EditIcon className="w-5 h-5" />
                    </button>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Invitations Tab */}
      {activeTab === 'invites' && (
        <div className="space-y-6">
          {/* Invite Form */}
          <div className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">
              Invite New Member
            </h2>
            <form onSubmit={handleInvite} className="flex gap-4">
              <input
                type="email"
                placeholder="colleague@example.com"
                value={inviteEmail}
                onChange={(e) => setInviteEmail(e.target.value)}
                required
                className="flex-1 px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-gray-100"
              />
              <select
                value={inviteRole}
                onChange={(e) => setInviteRole(e.target.value as 'user' | 'admin')}
                className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-gray-100"
              >
                <option value="user">User</option>
                <option value="admin">Admin</option>
              </select>
              <button
                type="submit"
                disabled={inviteMutation.isPending}
                className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors font-medium disabled:opacity-50 inline-flex items-center gap-2"
              >
                <PlusIcon className="w-5 h-5" />
                Send Invite
              </button>
            </form>
          </div>

          {/* Pending Invitations */}
          {invitations && invitations.length > 0 && (
            <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700">
              <div className="p-6 border-b border-gray-200 dark:border-gray-700">
                <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                  Pending Invitations ({invitations.length})
                </h2>
              </div>
              <div className="divide-y divide-gray-200 dark:divide-gray-700">
                {invitations.map((invite) => (
                  <div key={invite.id} className="p-4 hover:bg-gray-50 dark:hover:bg-gray-700/50">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-4">
                        <div className="w-10 h-10 bg-yellow-100 dark:bg-yellow-900/30 text-yellow-600 dark:text-yellow-400 rounded-full flex items-center justify-center">
                          <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path
                              strokeLinecap="round"
                              strokeLinejoin="round"
                              strokeWidth={2}
                              d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
                            />
                          </svg>
                        </div>
                        <div>
                          <p className="text-sm font-medium text-gray-900 dark:text-gray-100">
                            {invite.email}
                          </p>
                          <p className="text-sm text-gray-500 dark:text-gray-400">
                            Role: {invite.role} • Expires{' '}
                            {new Date(invite.expiresAt).toLocaleDateString()}
                          </p>
                        </div>
                      </div>
                      <button
                        onClick={() => cancelInviteMutation.mutate(invite.id)}
                        className="p-2 text-gray-400 hover:text-red-600 dark:hover:text-red-400 transition-colors"
                      >
                        <TrashIcon className="w-5 h-5" />
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
};

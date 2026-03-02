/**
 * OrganizationDetail - Platform admin page for organization details
 *
 * Features:
 * - Overview tab with organization info and quotas
 * - Users tab for managing organization members
 * - Usage tab with usage reports and trends
 * - Settings tab for configuration
 * - Audit trail tab
 */

import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useNavigate, useParams } from 'react-router-dom';
import { platformAdminApi, orgUsersApi, usageReportApi, quotaApi } from '@/services/platformAdminApi';
import type { UsagePeriod } from '@/types';
import { QuotaCard } from '@/components/organization/QuotaCard';
import { OrgUserCard } from '@/components/organization/OrgUserCard';
import { UsageReportChart } from '@/components/organization/UsageReportChart';
import { OrganizationForm } from '@/components/organization/OrganizationForm';

type TabValue = 'overview' | 'users' | 'usage' | 'settings' | 'audit';

// Icons
const BackIcon = () => (
  <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
  </svg>
);

const EditIcon = () => (
  <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
  </svg>
);

const TrashIcon = () => (
  <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
  </svg>
);

const ClockIcon = () => (
  <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
  </svg>
);

const Tab = ({
  active,
  onClick,
  children,
  count,
}: {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
  count?: number;
}) => (
  <button
    onClick={onClick}
    className={`px-4 py-3 text-sm font-medium border-b-2 transition-colors flex items-center gap-2 ${
      active
        ? 'border-blue-500 text-blue-600 dark:text-blue-400'
        : 'border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300'
    }`}
  >
    {children}
    {count !== undefined && (
      <span className={`px-2 py-0.5 rounded-full text-xs ${
        active
          ? 'bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400'
          : 'bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400'
      }`}>
        {count}
      </span>
    )}
  </button>
);

export const OrganizationDetail = () => {
  const { orgId } = useParams<{ orgId: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const [activeTab, setActiveTab] = useState<TabValue>('overview');
  const [showEditModal, setShowEditModal] = useState(false);
  const [usagePeriod, setUsagePeriod] = useState<UsagePeriod>('monthly');

  // Fetch organization details
  const { data: organization, isLoading: orgLoading } = useQuery({
    queryKey: ['organization', orgId],
    queryFn: () => platformAdminApi.getOrganization(orgId!),
    enabled: !!orgId,
  });

  // Fetch quota
  const { data: quota, isLoading: quotaLoading } = useQuery({
    queryKey: ['organizationQuota', orgId],
    queryFn: () => quotaApi.getOrganizationQuota(orgId!),
    enabled: !!orgId,
  });

  // Fetch users
  const { data: users = [], isLoading: usersLoading } = useQuery({
    queryKey: ['organizationUsers', orgId],
    queryFn: () => orgUsersApi.getOrganizationUsers(orgId!),
    enabled: !!orgId && activeTab === 'users',
  });

  // Fetch usage trends
  const { data: usageTrends = [], isLoading: usageLoading } = useQuery({
    queryKey: ['usageTrends', orgId, usagePeriod],
    queryFn: () => usageReportApi.getUsageTrends(orgId!, usagePeriod as 'daily' | 'weekly' | 'monthly'),
    enabled: !!orgId && activeTab === 'usage',
  });

  // Update mutation
  const updateMutation = useMutation({
    mutationFn: (data: any) => platformAdminApi.updateOrganization(orgId!, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['organization', orgId] });
      setShowEditModal(false);
    },
  });

  // Suspend mutation
  const suspendMutation = useMutation({
    mutationFn: (reason: string) => platformAdminApi.suspendOrganization(orgId!, reason),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['organization', orgId] });
    },
  });

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: () => platformAdminApi.deleteOrganization(orgId!),
    onSuccess: () => {
      navigate('/admin/organizations');
    },
  });

  const handleSuspend = () => {
    const reason = prompt('Enter reason for suspending this organization:');
    if (reason) {
      suspendMutation.mutate(reason);
    }
  };

  const handleDelete = () => {
    if (confirm(`Are you sure you want to DELETE ${organization?.name}? This action cannot be undone.`)) {
      deleteMutation.mutate();
    }
  };

  const handleInviteUser = () => {
    // TODO: Implement invite modal
    console.log('Invite user');
  };

  if (orgLoading) {
    return (
      <div className="flex items-center justify-center min-h-[50vh]">
        <div className="animate-spin w-8 h-8 border-4 border-blue-600 border-t-transparent rounded-full" />
      </div>
    );
  }

  if (!organization) {
    return (
      <div className="text-center py-12">
        <p className="text-red-600 dark:text-red-400">Organization not found</p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <button
            onClick={() => navigate('/admin/organizations')}
            className="p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
          >
            <BackIcon />
          </button>
          <div className="w-12 h-12 bg-gradient-to-br from-blue-500 to-cyan-500 rounded-xl flex items-center justify-center text-white font-bold text-lg">
            {organization.name.charAt(0).toUpperCase()}
          </div>
          <div>
            <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">
              {organization.name}
            </h1>
            <p className="text-gray-500 dark:text-gray-400 text-sm">
              {organization.slug} • Created {new Date(organization.createdAt).toLocaleDateString()}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => setShowEditModal(true)}
            className="px-4 py-2 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors font-medium flex items-center gap-2"
          >
            <EditIcon />
            Edit
          </button>
          {organization.status === 'active' ? (
            <button
              onClick={handleSuspend}
              disabled={suspendMutation.isPending}
              className="px-4 py-2 border border-yellow-300 dark:border-yellow-700 text-yellow-700 dark:text-yellow-400 rounded-lg hover:bg-yellow-50 dark:hover:bg-yellow-900/20 transition-colors font-medium"
            >
              Suspend
            </button>
          ) : (
            <button
              onClick={() => {
                platformAdminApi.reactivateOrganization(orgId!).then(() =>
                  queryClient.invalidateQueries({ queryKey: ['organization', orgId] })
                );
              }}
              className="px-4 py-2 border border-green-300 dark:border-green-700 text-green-700 dark:text-green-400 rounded-lg hover:bg-green-50 dark:hover:bg-green-900/20 transition-colors font-medium"
            >
              Reactivate
            </button>
          )}
          <button
            onClick={handleDelete}
            disabled={deleteMutation.isPending}
            className="px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors font-medium flex items-center gap-2"
          >
            <TrashIcon />
            Delete
          </button>
        </div>
      </div>

      {/* Status Banner */}
      {organization.status === 'suspended' && (
        <div className="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-4">
          <div className="flex items-center gap-3">
            <svg className="w-5 h-5 text-yellow-600 dark:text-yellow-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
            <div>
              <p className="font-medium text-yellow-800 dark:text-yellow-300">Organization Suspended</p>
              <p className="text-sm text-yellow-700 dark:text-yellow-400">
                This organization is currently suspended and cannot access the platform.
              </p>
            </div>
          </div>
        </div>
      )}

      {/* Tabs */}
      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700">
        <div className="border-b border-gray-200 dark:border-gray-700 flex overflow-x-auto">
          <Tab active={activeTab === 'overview'} onClick={() => setActiveTab('overview')}>
            Overview
          </Tab>
          <Tab
            active={activeTab === 'users'}
            onClick={() => setActiveTab('users')}
            count={organization.currentUserCount}
          >
            Users
          </Tab>
          <Tab active={activeTab === 'usage'} onClick={() => setActiveTab('usage')}>
            Usage
          </Tab>
          <Tab active={activeTab === 'settings'} onClick={() => setActiveTab('settings')}>
            Settings
          </Tab>
          <Tab active={activeTab === 'audit'} onClick={() => setActiveTab('audit')}>
            Audit Trail
          </Tab>
        </div>

        {/* Tab Content */}
        <div className="p-6">
          {/* Overview Tab */}
          {activeTab === 'overview' && (
            <div className="space-y-6">
              {/* Organization Info */}
              <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                <div className="bg-gray-50 dark:bg-gray-700/50 rounded-lg p-4">
                  <p className="text-xs text-gray-500 dark:text-gray-400 uppercase tracking-wide">Plan</p>
                  <p className="text-lg font-semibold text-gray-900 dark:text-gray-100 capitalize">{organization.plan}</p>
                </div>
                <div className="bg-gray-50 dark:bg-gray-700/50 rounded-lg p-4">
                  <p className="text-xs text-gray-500 dark:text-gray-400 uppercase tracking-wide">Status</p>
                  <p className="text-lg font-semibold text-gray-900 dark:text-gray-100 capitalize">{organization.status}</p>
                </div>
                <div className="bg-gray-50 dark:bg-gray-700/50 rounded-lg p-4">
                  <p className="text-xs text-gray-500 dark:text-gray-400 uppercase tracking-wide">Health Score</p>
                  <p className="text-lg font-semibold text-gray-900 dark:text-gray-100">{organization.healthScore}%</p>
                </div>
              </div>

              {/* Quotas */}
              <QuotaCard quota={quota ?? null} isLoading={quotaLoading} />

              {/* Quick Stats */}
              <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
                <div className="text-center p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
                  <p className="text-2xl font-bold text-blue-600 dark:text-blue-400">{organization.currentUserCount}</p>
                  <p className="text-sm text-gray-600 dark:text-gray-400">Users</p>
                </div>
                <div className="text-center p-4 bg-purple-50 dark:bg-purple-900/20 rounded-lg">
                  <p className="text-2xl font-bold text-purple-600 dark:text-purple-400">{organization.currentPrinterCount}</p>
                  <p className="text-sm text-gray-600 dark:text-gray-400">Printers</p>
                </div>
                <div className="text-center p-4 bg-green-50 dark:bg-green-900/20 rounded-lg">
                  <p className="text-2xl font-bold text-green-600 dark:text-green-400">{organization.usagePercentage}%</p>
                  <p className="text-sm text-gray-600 dark:text-gray-400">Usage</p>
                </div>
                <div className="text-center p-4 bg-orange-50 dark:bg-orange-900/20 rounded-lg">
                  <p className="text-2xl font-bold text-orange-600 dark:text-orange-400">{organization.alertCount}</p>
                  <p className="text-sm text-gray-600 dark:text-gray-400">Alerts</p>
                </div>
              </div>
            </div>
          )}

          {/* Users Tab */}
          {activeTab === 'users' && (
            <OrgUserCard
              users={users}
              organizationId={orgId!}
              isLoading={usersLoading}
              canManage={true}
              onInvite={handleInviteUser}
            />
          )}

          {/* Usage Tab */}
          {activeTab === 'usage' && (
            <div className="space-y-6">
              <div className="flex items-center justify-between">
                <div>
                  <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Usage Trends</h2>
                  <p className="text-sm text-gray-500 dark:text-gray-400">Track organization usage over time</p>
                </div>
                <select
                  value={usagePeriod}
                  onChange={e => setUsagePeriod(e.target.value as UsagePeriod)}
                  className="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg dark:bg-gray-700 dark:text-gray-100 text-sm"
                >
                  <option value="daily">Daily</option>
                  <option value="weekly">Weekly</option>
                  <option value="monthly">Monthly</option>
                  <option value="yearly">Yearly</option>
                </select>
              </div>
              <UsageReportChart data={usageTrends} period={usagePeriod} isLoading={usageLoading} />
            </div>
          )}

          {/* Settings Tab */}
          {activeTab === 'settings' && (
            <div className="space-y-6">
              <div>
                <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">Organization Settings</h2>
                <div className="space-y-4">
                  <div className="flex items-center justify-between p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
                    <div>
                      <p className="font-medium text-gray-900 dark:text-gray-100">Organization Name</p>
                      <p className="text-sm text-gray-500 dark:text-gray-400">{organization.name}</p>
                    </div>
                    <button className="text-blue-600 dark:text-blue-400 text-sm hover:underline">
                      Change
                    </button>
                  </div>
                  <div className="flex items-center justify-between p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
                    <div>
                      <p className="font-medium text-gray-900 dark:text-gray-100">Organization Slug</p>
                      <p className="text-sm text-gray-500 dark:text-gray-400">{organization.slug}</p>
                    </div>
                  </div>
                  <div className="flex items-center justify-between p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
                    <div>
                      <p className="font-medium text-gray-900 dark:text-gray-100">Plan</p>
                      <p className="text-sm text-gray-500 dark:text-gray-400 capitalize">{organization.plan}</p>
                    </div>
                    <button className="text-blue-600 dark:text-blue-400 text-sm hover:underline">
                      Upgrade
                    </button>
                  </div>
                </div>
              </div>
            </div>
          )}

          {/* Audit Tab */}
          {activeTab === 'audit' && (
            <div className="space-y-4">
              <div className="text-center py-12">
                <ClockIcon />
                <p className="text-gray-500 dark:text-gray-400 mt-4">Audit trail coming soon</p>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Edit Modal */}
      {showEditModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
          <div className="bg-white dark:bg-gray-800 rounded-xl shadow-xl max-w-2xl w-full max-h-[90vh] overflow-y-auto">
            <div className="p-6 border-b border-gray-200 dark:border-gray-700">
              <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100">
                Edit Organization
              </h2>
            </div>
            <div className="p-6">
              <OrganizationForm
                mode="edit"
                organization={organization as any}
                onSubmit={async (data) => {
                  await updateMutation.mutateAsync(data);
                }}
                onCancel={() => setShowEditModal(false)}
                isLoading={updateMutation.isPending}
              />
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default OrganizationDetail;

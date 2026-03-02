/**
 * OrganizationsList - Platform admin page for managing all organizations
 *
 * Features:
 * - List all organizations with filtering and search
 * - Create new organization
 * - Organization status management (suspend/reactivate)
 * - Quick actions and navigation to organization details
 */

import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import { platformAdminApi } from '@/services/platformAdminApi';
import type { OrganizationsListFilters, OrganizationStatus, OrganizationPlan } from '@/types';
import { OrganizationForm } from '@/components/organization/OrganizationForm';

// Icons
const SearchIcon = () => (
  <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
  </svg>
);

const PlusIcon = () => (
  <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
  </svg>
);

const ChevronRightIcon = () => (
  <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
  </svg>
);

const StatusBadge = ({ status }: { status: OrganizationStatus }) => {
  const styles: Record<OrganizationStatus, string> = {
    active: 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300',
    suspended: 'bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-300',
    trial: 'bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300',
    deleted: 'bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300',
  };

  return (
    <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${styles[status]}`}>
      {status.charAt(0).toUpperCase() + status.slice(1)}
    </span>
  );
};

const PlanBadge = ({ plan }: { plan: OrganizationPlan }) => {
  const styles: Record<OrganizationPlan, string> = {
    free: 'bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300',
    pro: 'bg-purple-100 dark:bg-purple-900/30 text-purple-700 dark:text-purple-300',
    enterprise: 'bg-yellow-100 dark:bg-yellow-900/30 text-yellow-700 dark:text-yellow-300',
  };

  return (
    <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${styles[plan]}`}>
      {plan.charAt(0).toUpperCase() + plan.slice(1)}
    </span>
  );
};

const UsageBar = ({ percentage }: { percentage: number }) => {
  const color =
    percentage >= 90 ? 'bg-red-500' : percentage >= 70 ? 'bg-yellow-500' : 'bg-green-500';

  return (
    <div className="flex items-center gap-2">
      <div className="w-24 h-2 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
        <div className={`h-full ${color} transition-all`} style={{ width: `${percentage}%` }} />
      </div>
      <span className="text-xs text-gray-500 dark:text-gray-400 w-8">{percentage}%</span>
    </div>
  );
};

export const OrganizationsList = () => {
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const [showCreateModal, setShowCreateModal] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedStatus, setSelectedStatus] = useState<OrganizationStatus | ''>('');
  const [selectedPlan, setSelectedPlan] = useState<OrganizationPlan | ''>('');
  const [sortBy, setSortBy] = useState<'name' | 'createdAt' | 'usage' | 'plan'>('name');
  const [currentPage, setCurrentPage] = useState(0);
  const pageSize = 20;

  const filters: OrganizationsListFilters = {
    status: selectedStatus || undefined,
    plan: selectedPlan || undefined,
    search: searchQuery || undefined,
    sortBy,
    sortOrder: 'asc',
  };

  // Fetch organizations
  const { data, isLoading, error } = useQuery({
    queryKey: ['organizations', filters, currentPage, pageSize],
    queryFn: () => platformAdminApi.listOrganizations(filters, pageSize, currentPage * pageSize),
  });

  // Create organization mutation
  const createMutation = useMutation({
    mutationFn: platformAdminApi.createOrganization,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['organizations'] });
      setShowCreateModal(false);
    },
  });

  // Suspend organization mutation
  const suspendMutation = useMutation({
    mutationFn: ({ orgId, reason }: { orgId: string; reason: string }) =>
      platformAdminApi.suspendOrganization(orgId, reason),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['organizations'] });
    },
  });

  // Reactivate organization mutation
  const reactivateMutation = useMutation({
    mutationFn: (orgId: string) => platformAdminApi.reactivateOrganization(orgId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['organizations'] });
    },
  });

  const handleSearch = (value: string) => {
    setSearchQuery(value);
    setCurrentPage(0);
  };

  const handleStatusFilter = (status: OrganizationStatus | '') => {
    setSelectedStatus(status);
    setCurrentPage(0);
  };

  const handlePlanFilter = (plan: OrganizationPlan | '') => {
    setSelectedPlan(plan);
    setCurrentPage(0);
  };

  const handleSort = (value: 'name' | 'createdAt' | 'usage' | 'plan') => {
    setSortBy(sortBy === value ? (value === 'name' ? 'createdAt' : 'name') : value);
  };

  const handleSuspend = (orgId: string, orgName: string) => {
    const reason = prompt(`Enter reason for suspending ${orgName}:`);
    if (reason) {
      suspendMutation.mutate({ orgId, reason });
    }
  };

  const handleReactivate = (orgId: string) => {
    if (confirm('Are you sure you want to reactivate this organization?')) {
      reactivateMutation.mutate(orgId);
    }
  };

  const handleDelete = (orgId: string, orgName: string) => {
    if (confirm(`Are you sure you want to DELETE ${orgName}? This action cannot be undone.`)) {
      if (confirm(`This will permanently delete ${orgName} and all associated data. Type the organization name to confirm:`)) {
        platformAdminApi.deleteOrganization(orgId)
          .then(() => queryClient.invalidateQueries({ queryKey: ['organizations'] }))
          .catch(err => alert(`Failed to delete: ${err.message}`));
      }
    }
  };

  const organizations = data?.data || [];
  const totalCount = data?.total || 0;
  const totalPages = Math.ceil(totalCount / pageSize);

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">
            Organizations
          </h1>
          <p className="text-gray-600 dark:text-gray-400 mt-1">
            Manage all organizations on the platform
          </p>
        </div>
        <button
          onClick={() => setShowCreateModal(true)}
          className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors font-medium flex items-center gap-2"
        >
          <PlusIcon />
          New Organization
        </button>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 sm:grid-cols-4 gap-4">
        <div className="bg-white dark:bg-gray-800 rounded-lg p-4 border border-gray-200 dark:border-gray-700">
          <p className="text-sm text-gray-500 dark:text-gray-400">Total Organizations</p>
          <p className="text-2xl font-semibold text-gray-900 dark:text-gray-100">{totalCount}</p>
        </div>
        <div className="bg-white dark:bg-gray-800 rounded-lg p-4 border border-gray-200 dark:border-gray-700">
          <p className="text-sm text-gray-500 dark:text-gray-400">Active</p>
          <p className="text-2xl font-semibold text-green-600 dark:text-green-400">
            {organizations.filter(o => o.status === 'active').length}
          </p>
        </div>
        <div className="bg-white dark:bg-gray-800 rounded-lg p-4 border border-gray-200 dark:border-gray-700">
          <p className="text-sm text-gray-500 dark:text-gray-400">Trial</p>
          <p className="text-2xl font-semibold text-blue-600 dark:text-blue-400">
            {organizations.filter(o => o.status === 'trial').length}
          </p>
        </div>
        <div className="bg-white dark:bg-gray-800 rounded-lg p-4 border border-gray-200 dark:border-gray-700">
          <p className="text-sm text-gray-500 dark:text-gray-400">Suspended</p>
          <p className="text-2xl font-semibold text-red-600 dark:text-red-400">
            {organizations.filter(o => o.status === 'suspended').length}
          </p>
        </div>
      </div>

      {/* Filters */}
      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 p-4">
        <div className="flex flex-wrap items-center gap-4">
          {/* Search */}
          <div className="relative flex-1 min-w-[200px]">
            <SearchIcon />
            <input
              type="text"
              placeholder="Search organizations..."
              value={searchQuery}
              onChange={e => handleSearch(e.target.value)}
              className="w-full pl-10 pr-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-gray-100"
            />
            <div className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400">
              <SearchIcon />
            </div>
          </div>

          {/* Status Filter */}
          <select
            value={selectedStatus}
            onChange={e => handleStatusFilter(e.target.value as OrganizationStatus | '')}
            className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-gray-100"
          >
            <option value="">All Statuses</option>
            <option value="active">Active</option>
            <option value="trial">Trial</option>
            <option value="suspended">Suspended</option>
            <option value="deleted">Deleted</option>
          </select>

          {/* Plan Filter */}
          <select
            value={selectedPlan}
            onChange={e => handlePlanFilter(e.target.value as OrganizationPlan | '')}
            className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-gray-100"
          >
            <option value="">All Plans</option>
            <option value="free">Free</option>
            <option value="pro">Pro</option>
            <option value="enterprise">Enterprise</option>
          </select>

          {/* Sort */}
          <select
            value={sortBy}
            onChange={e => handleSort(e.target.value as any)}
            className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-gray-100"
          >
            <option value="name">Sort by Name</option>
            <option value="createdAt">Sort by Created</option>
            <option value="usage">Sort by Usage</option>
            <option value="plan">Sort by Plan</option>
          </select>
        </div>
      </div>

      {/* Organizations List */}
      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700">
        {isLoading ? (
          <div className="p-8 text-center">
            <div className="animate-spin w-8 h-8 border-4 border-blue-600 border-t-transparent rounded-full mx-auto" />
            <p className="text-gray-500 dark:text-gray-400 mt-4">Loading organizations...</p>
          </div>
        ) : error ? (
          <div className="p-8 text-center">
            <p className="text-red-600 dark:text-red-400">Failed to load organizations</p>
          </div>
        ) : organizations.length === 0 ? (
          <div className="p-12 text-center">
            <div className="w-16 h-16 bg-gray-100 dark:bg-gray-700 rounded-full flex items-center justify-center mx-auto mb-4">
              <svg className="w-8 h-8 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4" />
              </svg>
            </div>
            <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-2">
              No organizations found
            </h3>
            <p className="text-gray-500 dark:text-gray-400 mb-4">
              {searchQuery || selectedStatus || selectedPlan
                ? 'Try adjusting your filters'
                : 'Get started by creating your first organization'}
            </p>
            {!searchQuery && !selectedStatus && !selectedPlan && (
              <button
                onClick={() => setShowCreateModal(true)}
                className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors font-medium"
              >
                Create Organization
              </button>
            )}
          </div>
        ) : (
          <div className="divide-y divide-gray-200 dark:divide-gray-700">
            {organizations.map(org => (
              <div
                key={org.id}
                className="p-4 hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors cursor-pointer"
                onClick={() => navigate(`/admin/organizations/${org.id}`)}
              >
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-4 flex-1">
                    {/* Organization Icon */}
                    <div className="w-12 h-12 bg-gradient-to-br from-blue-500 to-cyan-500 rounded-xl flex items-center justify-center text-white font-bold text-lg flex-shrink-0">
                      {org.name.charAt(0).toUpperCase()}
                    </div>

                    {/* Organization Info */}
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2">
                        <h3 className="text-sm font-semibold text-gray-900 dark:text-gray-100 truncate">
                          {org.name}
                        </h3>
                        <StatusBadge status={org.status} />
                        <PlanBadge plan={org.plan} />
                      </div>
                      <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                        {org.slug} • {org.currentUserCount} users • {org.currentPrinterCount} printers
                      </p>
                    </div>

                    {/* Usage */}
                    <div className="hidden md:block">
                      <UsageBar percentage={org.usagePercentage} />
                    </div>

                    {/* Actions */}
                    <div
                      className="flex items-center gap-2"
                      onClick={e => e.stopPropagation()}
                    >
                      {org.status === 'active' ? (
                        <button
                          onClick={() => handleSuspend(org.id, org.name)}
                          disabled={suspendMutation.isPending}
                          className="p-2 text-yellow-600 dark:text-yellow-400 hover:bg-yellow-50 dark:hover:bg-yellow-900/20 rounded-lg"
                          title="Suspend organization"
                        >
                          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 9v6m4-6v6m7-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                          </svg>
                        </button>
                      ) : org.status === 'suspended' ? (
                        <button
                          onClick={() => handleReactivate(org.id)}
                          disabled={reactivateMutation.isPending}
                          className="p-2 text-green-600 dark:text-green-400 hover:bg-green-50 dark:hover:bg-green-900/20 rounded-lg"
                          title="Reactivate organization"
                        >
                          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                          </svg>
                        </button>
                      ) : null}

                      <button
                        onClick={() => handleDelete(org.id, org.name)}
                        className="p-2 text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-lg"
                        title="Delete organization"
                      >
                        <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                        </svg>
                      </button>

                      <button
                        onClick={() => navigate(`/admin/organizations/${org.id}`)}
                        className="p-2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg"
                        title="View details"
                      >
                        <ChevronRightIcon />
                      </button>
                    </div>
                  </div>
                </div>

                {/* Alerts */}
                {org.alertCount > 0 && (
                  <div className="mt-3 flex items-center gap-2 text-xs">
                    <span className="px-2 py-1 bg-orange-100 dark:bg-orange-900/30 text-orange-700 dark:text-orange-300 rounded">
                      {org.alertCount} alert{org.alertCount > 1 ? 's' : ''} requiring attention
                    </span>
                  </div>
                )}
              </div>
            ))}
          </div>
        )}

        {/* Pagination */}
        {totalPages > 1 && (
          <div className="p-4 border-t border-gray-200 dark:border-gray-700 flex items-center justify-between">
            <p className="text-sm text-gray-500 dark:text-gray-400">
              Showing {currentPage * pageSize + 1} to {Math.min((currentPage + 1) * pageSize, totalCount)} of {totalCount} organizations
            </p>
            <div className="flex items-center gap-2">
              <button
                onClick={() => setCurrentPage(Math.max(0, currentPage - 1))}
                disabled={currentPage === 0}
                className="px-3 py-1 border border-gray-300 dark:border-gray-600 rounded-lg disabled:opacity-50 disabled:cursor-not-allowed text-sm"
              >
                Previous
              </button>
              <span className="text-sm text-gray-600 dark:text-gray-400">
                Page {currentPage + 1} of {totalPages}
              </span>
              <button
                onClick={() => setCurrentPage(Math.min(totalPages - 1, currentPage + 1))}
                disabled={currentPage >= totalPages - 1}
                className="px-3 py-1 border border-gray-300 dark:border-gray-600 rounded-lg disabled:opacity-50 disabled:cursor-not-allowed text-sm"
              >
                Next
              </button>
            </div>
          </div>
        )}
      </div>

      {/* Create Modal */}
      {showCreateModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
          <div className="bg-white dark:bg-gray-800 rounded-xl shadow-xl max-w-2xl w-full max-h-[90vh] overflow-y-auto">
            <div className="p-6 border-b border-gray-200 dark:border-gray-700">
              <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100">
                Create New Organization
              </h2>
              <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
                Set up a new organization tenant on the platform
              </p>
            </div>
            <div className="p-6">
              <OrganizationForm
                mode="create"
                onSubmit={async (data) => {
                  await createMutation.mutateAsync(data as any);
                }}
                onCancel={() => setShowCreateModal(false)}
                isLoading={createMutation.isPending}
              />
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default OrganizationsList;

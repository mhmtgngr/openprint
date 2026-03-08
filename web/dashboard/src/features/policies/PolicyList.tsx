import { type FC, useState } from 'react';
import { usePolicies, useDeletePolicy, useTogglePolicy, useDuplicatePolicy } from './usePolicies';
import { PolicyCard } from './PolicyCard';
import type { PolicyStatus, PolicySort, PolicyFilterOptions } from './types';

interface PolicyListProps {
  onCreatePolicy?: () => void;
  onEditPolicy?: (policyId: string) => void;
  onExportPolicy?: (policyId: string) => void;
  onViewHistory?: (policyId: string) => void;
  exportingPolicyId?: string | null;
  filters?: PolicyFilterOptions;
}

export const PolicyList: FC<PolicyListProps> = ({
  onCreatePolicy,
  onEditPolicy,
  onExportPolicy,
  onViewHistory,
  exportingPolicyId,
  filters = {},
}) => {
  const { data: policies, isLoading, error } = usePolicies();
  const deleteMutation = useDeletePolicy();
  const toggleMutation = useTogglePolicy();
  const duplicateMutation = useDuplicatePolicy();

  const [searchTerm, setSearchTerm] = useState(filters.search || '');
  const [statusFilter, setStatusFilter] = useState<PolicyStatus>(filters.status || 'all');
  const [sortBy, setSortBy] = useState<PolicySort>(filters.sortBy || 'priority');

  const filteredAndSortedPolicies = policies
    ?.filter((policy) => {
      // Search filter
      const matchesSearch =
        !searchTerm ||
        policy.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
        policy.description?.toLowerCase().includes(searchTerm.toLowerCase());

      // Status filter
      const matchesStatus =
        statusFilter === 'all' ||
        (statusFilter === 'enabled' && policy.isEnabled) ||
        (statusFilter === 'disabled' && !policy.isEnabled);

      return matchesSearch && matchesStatus;
    })
    .sort((a, b) => {
      switch (sortBy) {
        case 'priority':
          return a.priority - b.priority;
        case 'name':
          return a.name.localeCompare(b.name);
        case 'created':
          return new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime();
        case 'updated':
          return new Date(a.updatedAt).getTime() - new Date(b.updatedAt).getTime();
        default:
          return 0;
      }
    });

  if (isLoading) {
    return (
      <div
        data-testid="policy-list-loading"
        className="text-center py-12 text-gray-500 dark:text-gray-400"
      >
        <div className="inline-block w-8 h-8 border-4 border-blue-600 border-t-transparent rounded-full animate-spin mb-4" />
        <p>Loading policies...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div
        data-testid="policy-list-error"
        className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-6 text-center"
      >
        <p className="text-red-600 dark:text-red-400">
          Failed to load policies. Please try again.
        </p>
      </div>
    );
  }

  if (!policies || policies.length === 0) {
    return (
      <div
        data-testid="empty-state"
        className="bg-white dark:bg-gray-800 rounded-xl p-12 text-center border border-gray-200 dark:border-gray-700"
      >
        <svg
          className="w-16 h-16 mx-auto text-gray-400 mb-4"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
          />
        </svg>
        <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-2">
          No policies configured
        </h3>
        <p className="text-gray-500 dark:text-gray-400 mb-4">
          Create your first print policy to enforce printing rules
        </p>
        {onCreatePolicy && (
          <button
            onClick={onCreatePolicy}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
          >
            Create Policy
          </button>
        )}
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* Filters Bar */}
      <div
        data-testid="policy-filters"
        className="bg-white dark:bg-gray-800 rounded-xl p-4 shadow-sm border border-gray-200 dark:border-gray-700"
      >
        <div className="flex flex-col md:flex-row gap-4">
          {/* Search */}
          <div className="flex-1">
            <div className="relative">
              <svg
                className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-400"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
                />
              </svg>
              <input
                type="text"
                data-testid="policy-search"
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                placeholder="Search policies..."
                className="w-full pl-10 pr-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>
          </div>

          {/* Status Filter */}
          <div className="flex gap-2">
            {(['all', 'enabled', 'disabled'] as const).map((status) => (
              <button
                key={status}
                data-testid={`status-filter-${status}`}
                onClick={() => setStatusFilter(status)}
                className={`px-4 py-2 rounded-lg font-medium transition-colors ${
                  statusFilter === status
                    ? 'bg-blue-600 text-white'
                    : 'bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-600'
                }`}
              >
                {status.charAt(0).toUpperCase() + status.slice(1)}
              </button>
            ))}
          </div>

          {/* Sort */}
          <div>
            <select
              data-testid="policy-sort"
              value={sortBy}
              onChange={(e) => setSortBy(e.target.value as PolicySort)}
              className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            >
              <option value="priority">Sort by Priority</option>
              <option value="name">Sort by Name</option>
              <option value="created">Sort by Created</option>
              <option value="updated">Sort by Updated</option>
            </select>
          </div>
        </div>
      </div>

      {/* Policies List */}
      <div data-testid="policy-list" className="space-y-3">
        {filteredAndSortedPolicies && filteredAndSortedPolicies.length === 0 ? (
          <div className="text-center py-8 text-gray-500 dark:text-gray-400">
            No policies match your filters.
          </div>
        ) : (
          filteredAndSortedPolicies?.map((policy) => (
            <PolicyCard
              key={policy.id}
              policy={policy}
              onEdit={() => onEditPolicy?.(policy.id)}
              onDelete={() => deleteMutation.mutate(policy.id)}
              onToggle={(enabled) => toggleMutation.mutate({ id: policy.id, isEnabled: enabled })}
              onDuplicate={() => duplicateMutation.mutate({ id: policy.id })}
              onExport={() => onExportPolicy?.(policy.id)}
              onViewHistory={() => onViewHistory?.(policy.id)}
              isDeleting={deleteMutation.isPending}
              isToggling={toggleMutation.isPending}
              isExporting={exportingPolicyId === policy.id}
            />
          ))
        )}
      </div>
    </div>
  );
};

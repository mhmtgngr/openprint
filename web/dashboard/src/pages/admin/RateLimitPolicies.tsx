/**
 * RateLimitPolicies - Admin page for managing rate limit policies
 *
 * Features:
 * - List all rate limit policies with filtering and search
 * - Create new rate limit policy
 * - Edit existing policy
 * - Toggle policy status (active/disabled)
 * - Delete policy
 * - Clone policy
 * - Test policy against a request
 * - View policy violations
 */

import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  rateLimitPoliciesApi,
  formatRateLimitWindow,
  calculateUsagePercentage,
  getUsageBarColor,
} from '@/api/ratelimitApi';
import type {
  RateLimitPolicy,
  RateLimitPolicyFilters,
  RateLimitScope,
  RateLimitDimension,
  RateLimitWindow,
  RateLimitAction,
  RateLimitAlgorithm,
} from '@/types/ratelimit';
import { PolicyFormModal } from '@/components/ratelimit/PolicyFormModal';
import { TestPolicyModal } from '@/components/ratelimit/TestPolicyModal';
import { PolicyViolationsPanel } from '@/components/ratelimit/PolicyViolationsPanel';

// ============================================================================
// Icons
// ============================================================================

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

const MoreVerticalIcon = () => (
  <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 5v.01M12 12v.01M12 19v.01M12 6a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2z" />
  </svg>
);

const EditIcon = () => (
  <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
  </svg>
);

const TrashIcon = () => (
  <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
  </svg>
);

const CopyIcon = () => (
  <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
  </svg>
);

const TestIcon = () => (
  <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
  </svg>
);

const ExclamationIcon = () => (
  <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
  </svg>
);

// ============================================================================
// Components
// ============================================================================

const StatusBadge = ({ status }: { status: string }) => {
  const styles: Record<string, string> = {
    active: 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300',
    disabled: 'bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300',
    draft: 'bg-yellow-100 dark:bg-yellow-900/30 text-yellow-700 dark:text-yellow-300',
  };

  return (
    <span
      className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
        styles[status] || styles.disabled
      }`}
    >
      {status === 'active' ? 'Active' : status === 'disabled' ? 'Disabled' : 'Draft'}
    </span>
  );
};

const ScopeBadge = ({ scope }: { scope: RateLimitScope }) => {
  const styles: Record<RateLimitScope, string> = {
    organization: 'bg-purple-100 dark:bg-purple-900/30 text-purple-700 dark:text-purple-300',
    endpoint: 'bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300',
    user: 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300',
    api_key: 'bg-orange-100 dark:bg-orange-900/30 text-orange-700 dark:text-orange-300',
  };

  const labels: Record<RateLimitScope, string> = {
    organization: 'Org',
    endpoint: 'Endpoint',
    user: 'User',
    api_key: 'API Key',
  };

  return (
    <span
      className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${styles[scope]}`}
    >
      {labels[scope]}
    </span>
  );
};

const UsageBar = ({ used, limit }: { used: number; limit: number }) => {
  const percentage = calculateUsagePercentage(used, limit);
  const color = getUsageBarColor(percentage);

  return (
    <div className="flex items-center gap-2">
      <div className="w-20 h-2 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
        <div className={`h-full ${color} transition-all`} style={{ width: `${percentage}%` }} />
      </div>
      <span className="text-xs text-gray-500 dark:text-gray-400 w-12 text-right">
        {used}/{limit}
      </span>
    </div>
  );
};

const ActionMenu = ({ policy, onEdit, onClone, onTest, onDelete, onToggle }: {
  policy: RateLimitPolicy;
  onEdit: () => void;
  onClone: () => void;
  onTest: () => void;
  onDelete: () => void;
  onToggle: () => void;
}) => {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <div className="relative">
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="p-2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg"
      >
        <MoreVerticalIcon />
      </button>

      {isOpen && (
        <>
          <div
            className="fixed inset-0 z-10"
            onClick={() => setIsOpen(false)}
          />
          <div className="absolute right-0 z-20 w-48 bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 py-1">
            <button
              onClick={() => {
                onEdit();
                setIsOpen(false);
              }}
              className="w-full px-4 py-2 text-left text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center gap-2"
            >
              <EditIcon />
              Edit
            </button>
            <button
              onClick={() => {
                onClone();
                setIsOpen(false);
              }}
              className="w-full px-4 py-2 text-left text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center gap-2"
            >
              <CopyIcon />
              Clone
            </button>
            <button
              onClick={() => {
                onTest();
                setIsOpen(false);
              }}
              className="w-full px-4 py-2 text-left text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center gap-2"
            >
              <TestIcon />
              Test
            </button>
            <button
              onClick={() => {
                onToggle();
                setIsOpen(false);
              }}
              className="w-full px-4 py-2 text-left text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center gap-2"
            >
              {policy.status === 'active' ? 'Disable' : 'Enable'}
            </button>
            <div className="border-t border-gray-200 dark:border-gray-700 my-1" />
            <button
              onClick={() => {
                onDelete();
                setIsOpen(false);
              }}
              className="w-full px-4 py-2 text-left text-sm text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 flex items-center gap-2"
            >
              <TrashIcon />
              Delete
            </button>
          </div>
        </>
      )}
    </div>
  );
};

// ============================================================================
// Main Page Component
// ============================================================================

export const RateLimitPolicies = () => {
  const queryClient = useQueryClient();

  const [showCreateModal, setShowCreateModal] = useState(false);
  const [editingPolicy, setEditingPolicy] = useState<RateLimitPolicy | undefined>();
  const [testingPolicy, setTestingPolicy] = useState<RateLimitPolicy | undefined>();
  const [showViolationsFor, setShowViolationsFor] = useState<string | undefined>();
  const [deletingPolicy, setDeletingPolicy] = useState<RateLimitPolicy | undefined>();

  const [searchQuery, setSearchQuery] = useState('');
  const [selectedScope, setSelectedScope] = useState<RateLimitScope | ''>('');
  const [selectedDimension, setSelectedDimension] = useState<RateLimitDimension | ''>('');
  const [selectedStatus, setSelectedStatus] = useState<string>('');
  const [sortBy, setSortBy] = useState<'name' | 'createdAt' | 'limit' | 'priority'>('name');
  const [currentPage, setCurrentPage] = useState(0);
  const pageSize = 20;

  const filters: RateLimitPolicyFilters = {
    scope: selectedScope || undefined,
    dimension: selectedDimension || undefined,
    status: (selectedStatus || undefined) as any,
    search: searchQuery || undefined,
    sortBy,
    sortOrder: 'asc',
  };

  // Fetch policies
  const { data, isLoading, error } = useQuery({
    queryKey: ['rate-limit-policies', filters, currentPage, pageSize],
    queryFn: () => rateLimitPoliciesApi.list(filters, pageSize, currentPage * pageSize),
    refetchInterval: 30000, // Refresh every 30 seconds
  });

  // Toggle policy mutation
  const toggleMutation = useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) =>
      rateLimitPoliciesApi.toggle(id, enabled),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['rate-limit-policies'] });
    },
  });

  // Delete policy mutation
  const deleteMutation = useMutation({
    mutationFn: (id: string) => rateLimitPoliciesApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['rate-limit-policies'] });
      setDeletingPolicy(undefined);
    },
  });

  const handleSearch = (value: string) => {
    setSearchQuery(value);
    setCurrentPage(0);
  };

  const handleSort = (value: typeof sortBy) => {
    setSortBy(sortBy === value ? (value === 'name' ? 'createdAt' : 'name') : value);
  };

  const handleToggle = (policy: RateLimitPolicy) => {
    toggleMutation.mutate({
      id: policy.id,
      enabled: policy.status !== 'active',
    });
  };

  const handleDelete = (policy: RateLimitPolicy) => {
    if (confirm(`Are you sure you want to delete the policy "${policy.name}"? This action cannot be undone.`)) {
      deleteMutation.mutate(policy.id);
    }
  };

  const handleClone = (policy: RateLimitPolicy) => {
    const newName = prompt(`Enter a name for the cloned policy:`, `${policy.name} (Copy)`);
    if (newName) {
      rateLimitPoliciesApi
        .clone(policy.id, newName)
        .then(() => queryClient.invalidateQueries({ queryKey: ['rate-limit-policies'] }))
        .catch((err) => alert(`Failed to clone: ${err.message}`));
    }
  };

  const policies = data?.data || [];
  const totalCount = data?.total || 0;
  const totalPages = Math.ceil(totalCount / pageSize);

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">
            Rate Limit Policies
          </h1>
          <p className="text-gray-600 dark:text-gray-400 mt-1">
            Configure and manage API rate limiting rules
          </p>
        </div>
        <button
          onClick={() => setShowCreateModal(true)}
          className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors font-medium flex items-center gap-2"
        >
          <PlusIcon />
          New Policy
        </button>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 sm:grid-cols-4 gap-4">
        <div className="bg-white dark:bg-gray-800 rounded-lg p-4 border border-gray-200 dark:border-gray-700">
          <p className="text-sm text-gray-500 dark:text-gray-400">Total Policies</p>
          <p className="text-2xl font-semibold text-gray-900 dark:text-gray-100">{totalCount}</p>
        </div>
        <div className="bg-white dark:bg-gray-800 rounded-lg p-4 border border-gray-200 dark:border-gray-700">
          <p className="text-sm text-gray-500 dark:text-gray-400">Active</p>
          <p className="text-2xl font-semibold text-green-600 dark:text-green-400">
            {policies.filter((p) => p.status === 'active').length}
          </p>
        </div>
        <div className="bg-white dark:bg-gray-800 rounded-lg p-4 border border-gray-200 dark:border-gray-700">
          <p className="text-sm text-gray-500 dark:text-gray-400">Violations (24h)</p>
          <p className="text-2xl font-semibold text-orange-600 dark:text-orange-400">
            {policies.reduce((sum, p) => sum + (p.violationCount || 0), 0)}
          </p>
        </div>
        <div className="bg-white dark:bg-gray-800 rounded-lg p-4 border border-gray-200 dark:border-gray-700">
          <p className="text-sm text-gray-500 dark:text-gray-400">Avg Request/Min</p>
          <p className="text-2xl font-semibold text-blue-600 dark:text-blue-400">
            {policies.length > 0
              ? Math.round(
                  policies.reduce((sum, p) => sum + (p.avgRequestsPerWindow || 0), 0) /
                    policies.length
                )
              : 0}
          </p>
        </div>
      </div>

      {/* Filters */}
      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 p-4">
        <div className="flex flex-wrap items-center gap-4">
          {/* Search */}
          <div className="relative flex-1 min-w-[200px]">
            <input
              type="text"
              placeholder="Search policies..."
              value={searchQuery}
              onChange={(e) => handleSearch(e.target.value)}
              className="w-full pl-10 pr-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-gray-100"
            />
            <div className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400">
              <SearchIcon />
            </div>
          </div>

          {/* Scope Filter */}
          <select
            value={selectedScope}
            onChange={(e) => {
              setSelectedScope(e.target.value as RateLimitScope | '');
              setCurrentPage(0);
            }}
            className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-gray-100"
          >
            <option value="">All Scopes</option>
            <option value="organization">Organization</option>
            <option value="endpoint">Endpoint</option>
            <option value="user">User</option>
            <option value="api_key">API Key</option>
          </select>

          {/* Dimension Filter */}
          <select
            value={selectedDimension}
            onChange={(e) => {
              setSelectedDimension(e.target.value as RateLimitDimension | '');
              setCurrentPage(0);
            }}
            className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-gray-100"
          >
            <option value="">All Dimensions</option>
            <option value="global">Global</option>
            <option value="per_ip">Per IP</option>
            <option value="per_user">Per User</option>
            <option value="per_api_key">Per API Key</option>
            <option value="per_endpoint">Per Endpoint</option>
          </select>

          {/* Status Filter */}
          <select
            value={selectedStatus}
            onChange={(e) => {
              setSelectedStatus(e.target.value);
              setCurrentPage(0);
            }}
            className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-gray-100"
          >
            <option value="">All Statuses</option>
            <option value="active">Active</option>
            <option value="disabled">Disabled</option>
            <option value="draft">Draft</option>
          </select>

          {/* Sort */}
          <select
            value={sortBy}
            onChange={(e) => handleSort(e.target.value as any)}
            className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-gray-100"
          >
            <option value="name">Sort by Name</option>
            <option value="createdAt">Sort by Created</option>
            <option value="limit">Sort by Limit</option>
            <option value="priority">Sort by Priority</option>
          </select>
        </div>
      </div>

      {/* Policies List */}
      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700">
        {isLoading ? (
          <div className="p-8 text-center">
            <div className="animate-spin w-8 h-8 border-4 border-blue-600 border-t-transparent rounded-full mx-auto" />
            <p className="text-gray-500 dark:text-gray-400 mt-4">Loading policies...</p>
          </div>
        ) : error ? (
          <div className="p-8 text-center">
            <p className="text-red-600 dark:text-red-400">Failed to load policies</p>
          </div>
        ) : policies.length === 0 ? (
          <div className="p-12 text-center">
            <div className="w-16 h-16 bg-gray-100 dark:bg-gray-700 rounded-full flex items-center justify-center mx-auto mb-4">
              <svg className="w-8 h-8 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
              </svg>
            </div>
            <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-2">
              No rate limit policies found
            </h3>
            <p className="text-gray-500 dark:text-gray-400 mb-4">
              {searchQuery || selectedScope || selectedDimension || selectedStatus
                ? 'Try adjusting your filters'
                : 'Get started by creating your first rate limit policy'}
            </p>
            {!searchQuery && !selectedScope && !selectedDimension && !selectedStatus && (
              <button
                onClick={() => setShowCreateModal(true)}
                className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors font-medium"
              >
                Create Policy
              </button>
            )}
          </div>
        ) : (
          <div className="divide-y divide-gray-200 dark:divide-gray-700">
            {policies.map((policy) => (
              <div
                key={policy.id}
                className="p-4 hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors"
              >
                <div className="flex items-start gap-4">
                  {/* Policy Icon */}
                  <div className="w-12 h-12 bg-gradient-to-br from-orange-500 to-red-500 rounded-xl flex items-center justify-center text-white flex-shrink-0">
                    <ExclamationIcon />
                  </div>

                  {/* Policy Info */}
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-1">
                      <h3 className="text-sm font-semibold text-gray-900 dark:text-gray-100">
                        {policy.name}
                      </h3>
                      <StatusBadge status={policy.status} />
                      <ScopeBadge scope={policy.scope} />
                    </div>

                    {policy.description && (
                      <p className="text-xs text-gray-500 dark:text-gray-400 mb-2">
                        {policy.description}
                      </p>
                    )}

                    <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-gray-600 dark:text-gray-400">
                      <span>
                        <span className="font-medium">{policy.limit}</span> requests /{' '}
                        {formatRateLimitWindow(policy.window, policy.windowSize)}
                      </span>
                      <span>•</span>
                      <span className="capitalize">{policy.dimension.replace('_', ' ')}</span>
                      {policy.endpoint && (
                        <>
                          <span>•</span>
                          <span className="font-mono">{policy.endpoint}</span>
                        </>
                      )}
                      <span>•</span>
                      <span className="capitalize">{policy.actionOnLimit.replace('_', ' ')}</span>
                    </div>

                    {/* Usage Bar */}
                    {policy.currentUsage !== undefined && (
                      <div className="mt-3">
                        <UsageBar used={policy.currentUsage} limit={policy.limit} />
                      </div>
                    )}
                  </div>

                  {/* Violation Count */}
                  <div className="text-center px-4">
                    <p className="text-2xl font-semibold text-orange-600 dark:text-orange-400">
                      {policy.violationCount || 0}
                    </p>
                    <p className="text-xs text-gray-500 dark:text-gray-400">violations</p>
                    <button
                      onClick={() => setShowViolationsFor(policy.id)}
                      className="text-xs text-blue-600 dark:text-blue-400 hover:underline mt-1"
                    >
                      View
                    </button>
                  </div>

                  {/* Actions */}
                  <ActionMenu
                    policy={policy}
                    onEdit={() => setEditingPolicy(policy)}
                    onClone={() => handleClone(policy)}
                    onTest={() => setTestingPolicy(policy)}
                    onDelete={() => handleDelete(policy)}
                    onToggle={() => handleToggle(policy)}
                  />
                </div>
              </div>
            ))}
          </div>
        )}

        {/* Pagination */}
        {totalPages > 1 && (
          <div className="p-4 border-t border-gray-200 dark:border-gray-700 flex items-center justify-between">
            <p className="text-sm text-gray-500 dark:text-gray-400">
              Showing {currentPage * pageSize + 1} to{' '}
              {Math.min((currentPage + 1) * pageSize, totalCount)} of {totalCount} policies
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

      {/* Create/Edit Modal */}
      {(showCreateModal || editingPolicy) && (
        <PolicyFormModal
          policy={editingPolicy}
          onClose={() => {
            setShowCreateModal(false);
            setEditingPolicy(undefined);
          }}
          onSave={() => {
            setShowCreateModal(false);
            setEditingPolicy(undefined);
            queryClient.invalidateQueries({ queryKey: ['rate-limit-policies'] });
          }}
        />
      )}

      {/* Test Modal */}
      {testingPolicy && (
        <TestPolicyModal
          policy={testingPolicy}
          onClose={() => setTestingPolicy(undefined)}
        />
      )}

      {/* Violations Panel */}
      {showViolationsFor && (
        <PolicyViolationsPanel
          policyId={showViolationsFor}
          onClose={() => setShowViolationsFor(undefined)}
        />
      )}
    </div>
  );
};

export default RateLimitPolicies;

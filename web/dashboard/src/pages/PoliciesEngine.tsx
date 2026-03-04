import { useState } from 'react';
import { usePolicies, useDeletePolicy, useTogglePolicy, useDuplicatePolicy } from '@/features/policies';
import { PolicyTemplates, PolicyBuilder, PolicyEvaluation } from '@/features/policies';

export const PoliciesEngine = () => {
  const [activeTab, setActiveTab] = useState<'policies' | 'templates' | 'builder' | 'evaluation'>('policies');
  const [searchTerm, setSearchTerm] = useState('');
  const [filterStatus, setFilterStatus] = useState<'all' | 'enabled' | 'disabled'>('all');
  const [sortBy, setSortBy] = useState<'priority' | 'name' | 'created' | 'updated'>('priority');

  const { data: policies, isLoading } = usePolicies();
  const deleteMutation = useDeletePolicy();
  const toggleMutation = useTogglePolicy();
  const duplicateMutation = useDuplicatePolicy();

  const filteredAndSortedPolicies = policies
    ?.filter((policy) => {
      const matchesSearch =
        !searchTerm ||
        policy.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
        policy.description?.toLowerCase().includes(searchTerm.toLowerCase());

      const matchesStatus =
        filterStatus === 'all' ||
        (filterStatus === 'enabled' && policy.isEnabled) ||
        (filterStatus === 'disabled' && !policy.isEnabled);

      return matchesSearch && matchesStatus;
    })
    .sort((a, b) => {
      switch (sortBy) {
        case 'priority':
          return a.priority - b.priority;
        case 'name':
          return a.name.localeCompare(b.name);
        case 'created':
          return new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime();
        case 'updated':
          return new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime();
        default:
          return 0;
      }
    });

  const getPriorityColor = (priority: number) => {
    if (priority <= 2) return 'bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400 border-red-200 dark:border-red-800';
    if (priority <= 5) return 'bg-orange-100 dark:bg-orange-900/30 text-orange-700 dark:text-orange-400 border-orange-200 dark:border-orange-800';
    if (priority <= 8) return 'bg-amber-100 dark:bg-amber-900/30 text-amber-700 dark:text-amber-400 border-amber-200 dark:border-amber-800';
    return 'bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-400 border-gray-200 dark:border-gray-600';
  };

  const getPriorityLabel = (priority: number) => {
    if (priority <= 2) return 'Critical';
    if (priority <= 5) return 'High';
    if (priority <= 8) return 'Medium';
    return 'Low';
  };

  const tabs = [
    { id: 'policies', label: 'Policies', count: policies?.length || 0 },
    { id: 'templates', label: 'Templates', count: 8 },
    { id: 'builder', label: 'Policy Builder' },
    { id: 'evaluation', label: 'Test & Evaluate' },
  ] as const;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-900 dark:text-gray-100">
            Policy Engine
          </h1>
          <p className="text-gray-600 dark:text-gray-400 mt-1">
            Create and manage print policies with a powerful rules engine
          </p>
        </div>
        <button
          onClick={() => {
            setActiveTab('builder');
          }}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-medium transition-colors flex items-center gap-2"
        >
          <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
          New Policy
        </button>
      </div>

      {/* Tabs */}
      <div className="border-b border-gray-200 dark:border-gray-700">
        <nav className="flex gap-8 overflow-x-auto">
          {tabs.map((tab) => (
            <button
              key={tab.id}
              onClick={() => {
                setActiveTab(tab.id as any);
              }}
              className={`
                py-4 px-1 border-b-2 font-medium text-sm transition-colors whitespace-nowrap
                ${activeTab === tab.id
                  ? 'border-blue-500 text-blue-600 dark:text-blue-400'
                  : 'border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300'
                }
              `}
              aria-current={activeTab === tab.id ? 'page' : undefined}
            >
              {tab.label}
              {'count' in tab && (
                <span className={`ml-2 px-2 py-0.5 rounded-full text-xs ${
                  activeTab === tab.id
                    ? 'bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400'
                    : 'bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400'
                }`}>
                  {tab.count}
                </span>
              )}
            </button>
          ))}
        </nav>
      </div>

      {/* Policies Tab */}
      {activeTab === 'policies' && (
        <div className="space-y-4">
          {/* Filters */}
          <div className="bg-white dark:bg-gray-800 rounded-xl p-4 shadow-sm border border-gray-200 dark:border-gray-700">
            <div className="flex flex-col md:flex-row gap-4">
              <div className="flex-1">
                <div className="relative">
                  <svg
                    className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-400"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                  >
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                  </svg>
                  <input
                    type="text"
                    value={searchTerm}
                    onChange={(e) => setSearchTerm(e.target.value)}
                    placeholder="Search policies..."
                    className="w-full pl-10 pr-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  />
                </div>
              </div>
              <div className="flex gap-2">
                {(['all', 'enabled', 'disabled'] as const).map((status) => (
                  <button
                    key={status}
                    onClick={() => setFilterStatus(status)}
                    className={`px-4 py-2 rounded-lg font-medium transition-colors ${
                      filterStatus === status
                        ? 'bg-blue-600 text-white'
                        : 'bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-600'
                    }`}
                  >
                    {status.charAt(0).toUpperCase() + status.slice(1)}
                  </button>
                ))}
              </div>
              <div>
                <select
                  value={sortBy}
                  onChange={(e) => setSortBy(e.target.value as any)}
                  className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
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
          <div className="space-y-3">
            {isLoading ? (
              <div className="text-center py-12 text-gray-500 dark:text-gray-400">
                <div className="inline-block w-8 h-8 border-4 border-blue-600 border-t-transparent rounded-full animate-spin mb-4" />
                <p>Loading policies...</p>
              </div>
            ) : filteredAndSortedPolicies && filteredAndSortedPolicies.length === 0 ? (
              <div className="text-center py-12 text-gray-500 dark:text-gray-400">
                <p>No policies match your filters.</p>
              </div>
            ) : (
              filteredAndSortedPolicies?.map((policy) => (
                <div
                  key={policy.id}
                  data-policy-id={policy.id}
                  className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700 hover:shadow-md transition-shadow"
                >
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <div className="flex items-center gap-3 mb-2">
                        <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                          {policy.name}
                        </h3>
                        <span className={`px-2 py-0.5 rounded-full text-xs font-medium border ${getPriorityColor(policy.priority)}`}>
                          {getPriorityLabel(policy.priority)}
                        </span>
                        {!policy.isEnabled && (
                          <span className="px-2 py-0.5 rounded-full text-xs font-medium bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400">
                            Disabled
                          </span>
                        )}
                      </div>
                      <p className="text-gray-600 dark:text-gray-400 mb-4">{policy.description}</p>

                      {/* Display conditions summary */}
                      <div className="mb-4">
                        <p className="text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider mb-2">
                          Conditions
                        </p>
                        <div className="flex flex-wrap gap-2 text-sm">
                          {policy.conditions.maxPagesPerJob && (
                            <span className="inline-flex items-center gap-1 px-3 py-1 bg-blue-50 dark:bg-blue-900/20 text-blue-700 dark:text-blue-300 rounded-lg">
                              <span className="font-medium">Max Pages:</span>
                              <span>{policy.conditions.maxPagesPerJob}</span>
                            </span>
                          )}
                          {policy.conditions.maxPagesPerMonth && (
                            <span className="inline-flex items-center gap-1 px-3 py-1 bg-blue-50 dark:bg-blue-900/20 text-blue-700 dark:text-blue-300 rounded-lg">
                              <span className="font-medium">Max Pages/Month:</span>
                              <span>{policy.conditions.maxPagesPerMonth}</span>
                            </span>
                          )}
                          {policy.conditions.allowedFileTypes && (
                            <span className="inline-flex items-center gap-1 px-3 py-1 bg-blue-50 dark:bg-blue-900/20 text-blue-700 dark:text-blue-300 rounded-lg">
                              <span className="font-medium">File Types:</span>
                              <span>{policy.conditions.allowedFileTypes.join(', ')}</span>
                            </span>
                          )}
                        </div>
                      </div>

                      {/* Display actions summary */}
                      <div>
                        <p className="text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider mb-2">
                          Actions
                        </p>
                        <div className="flex flex-wrap gap-2">
                          {policy.actions.forceDuplex && (
                            <span className="inline-flex items-center gap-1 px-3 py-1 bg-green-50 dark:bg-green-900/20 text-green-700 dark:text-green-300 rounded-lg">
                              Force Duplex
                            </span>
                          )}
                          {policy.actions.forceColor === 'grayscale' && (
                            <span className="inline-flex items-center gap-1 px-3 py-1 bg-green-50 dark:bg-green-900/20 text-green-700 dark:text-green-300 rounded-lg">
                              Force Grayscale
                            </span>
                          )}
                          {policy.actions.requireApproval && (
                            <span className="inline-flex items-center gap-1 px-3 py-1 bg-green-50 dark:bg-green-900/20 text-green-700 dark:text-green-300 rounded-lg">
                              Require Approval
                            </span>
                          )}
                          {policy.actions.maxCopies && (
                            <span className="inline-flex items-center gap-1 px-3 py-1 bg-green-50 dark:bg-green-900/20 text-green-700 dark:text-green-300 rounded-lg">
                              Max Copies: {policy.actions.maxCopies}
                            </span>
                          )}
                        </div>
                      </div>
                    </div>

                    {/* Actions */}
                    <div className="flex items-center gap-2 ml-4">
                      <button
                        onClick={() => toggleMutation.mutate({ id: policy.id, isEnabled: !policy.isEnabled })}
                        className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                          policy.isEnabled ? 'bg-blue-600' : 'bg-gray-300 dark:bg-gray-600'
                        }`}
                        role="switch"
                        aria-checked={policy.isEnabled}
                      >
                        <span
                          className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                            policy.isEnabled ? 'translate-x-6' : 'translate-x-1'
                          }`}
                        />
                      </button>
                      <button className="p-2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300">
                        <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                        </svg>
                      </button>
                      <button
                        onClick={() => duplicateMutation.mutate({ id: policy.id })}
                        className="p-2 text-gray-400 hover:text-purple-600 dark:hover:text-purple-400"
                      >
                        <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                        </svg>
                      </button>
                      <button
                        onClick={() => deleteMutation.mutate(policy.id)}
                        className="p-2 text-gray-400 hover:text-red-600 dark:hover:text-red-400"
                      >
                        <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                        </svg>
                      </button>
                    </div>
                  </div>
                </div>
              ))
            )}
          </div>
        </div>
      )}

      {/* Templates Tab */}
      {activeTab === 'templates' && (
        <PolicyTemplates
          onSelectTemplate={() => {
            setActiveTab('builder');
          }}
        />
      )}

      {/* Builder Tab */}
      {activeTab === 'builder' && (
        <PolicyBuilder
          onClose={() => {
            setActiveTab('policies');
          }}
          onSave={() => {
            setActiveTab('policies');
          }}
        />
      )}

      {/* Evaluation Tab */}
      {activeTab === 'evaluation' && (
        <PolicyEvaluation
          onResult={(result) => {
            console.log('Evaluation result:', result);
          }}
        />
      )}
    </div>
  );
};

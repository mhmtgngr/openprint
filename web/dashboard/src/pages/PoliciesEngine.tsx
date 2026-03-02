import { useState } from 'react';

type PolicyCondition = {
  type: 'user' | 'group' | 'printer' | 'time' | 'document_type' | 'page_count';
  operator: 'equals' | 'contains' | 'not_equals' | 'greater_than' | 'less_than' | 'between';
  value: string;
};

type PolicyAction = {
  type: 'allow' | 'deny' | 'redirect' | 'modify' | 'notify';
  parameter?: string;
};

type PolicyPriority = 'low' | 'medium' | 'high' | 'critical';

interface PrintPolicy {
  id: string;
  name: string;
  description: string;
  priority: PolicyPriority;
  enabled: boolean;
  conditions: PolicyCondition[];
  actions: PolicyAction[];
  createdAt: string;
  lastModified: string;
}

interface PolicyTemplate {
  id: string;
  name: string;
  description: string;
  category: 'security' | 'cost_control' | 'access_control' | 'quality';
}

export const PoliciesEngine = () => {
  const [activeTab, setActiveTab] = useState<'policies' | 'templates' | 'builder'>('policies');
  const [searchTerm, setSearchTerm] = useState('');
  const [filterStatus, setFilterStatus] = useState<'all' | 'enabled' | 'disabled'>('all');

  // Mock policies data
  const policies: PrintPolicy[] = [
    {
      id: '1',
      name: 'Block Color Printing for Students',
      description: 'Prevent students from printing in color unless authorized',
      priority: 'high',
      enabled: true,
      conditions: [
        { type: 'group', operator: 'equals', value: 'students' },
        { type: 'document_type', operator: 'not_equals', value: 'image' },
      ],
      actions: [
        { type: 'modify', parameter: 'force_grayscale' },
        { type: 'notify', parameter: 'admin@openprint.test' },
      ],
      createdAt: '2024-01-01T00:00:00Z',
      lastModified: '2024-01-10T00:00:00Z',
    },
    {
      id: '2',
      name: 'Limit Large Print Jobs',
      description: 'Require approval for print jobs over 100 pages',
      priority: 'medium',
      enabled: true,
      conditions: [
        { type: 'page_count', operator: 'greater_than', value: '100' },
      ],
      actions: [
        { type: 'notify', parameter: 'manager@openprint.test' },
        { type: 'allow' },
      ],
      createdAt: '2024-01-02T00:00:00Z',
      lastModified: '2024-01-05T00:00:00Z',
    },
    {
      id: '3',
      name: 'After-Hours Print Restriction',
      description: 'Restrict printing to authorized personnel outside business hours',
      priority: 'high',
      enabled: true,
      conditions: [
        { type: 'time', operator: 'between', value: '18:00-08:00' },
      ],
      actions: [
        { type: 'deny' },
      ],
      createdAt: '2024-01-03T00:00:00Z',
      lastModified: '2024-01-08T00:00:00Z',
    },
    {
      id: '4',
      name: 'Duplex Default for Reports',
      description: 'Force double-sided printing for report documents',
      priority: 'low',
      enabled: false,
      conditions: [
        { type: 'document_type', operator: 'contains', value: 'report' },
      ],
      actions: [
        { type: 'modify', parameter: 'force_duplex' },
      ],
      createdAt: '2024-01-04T00:00:00Z',
      lastModified: '2024-01-04T00:00:00Z',
    },
  ];

  // Policy templates
  const templates: PolicyTemplate[] = [
    {
      id: 't1',
      name: 'Color Restriction',
      description: 'Restrict color printing to specific users or groups',
      category: 'cost_control',
    },
    {
      id: 't2',
      name: 'Quota Enforcement',
      description: 'Enforce page quotas per user or group',
      category: 'cost_control',
    },
    {
      id: 't3',
      name: 'Time-Based Access',
      description: 'Control printer access by time of day',
      category: 'access_control',
    },
    {
      id: 't4',
      name: 'Secure Document Routing',
      description: 'Route sensitive documents to secure printers only',
      category: 'security',
    },
    {
      id: 't5',
      name: 'Default Duplex',
      description: 'Enable double-sided printing by default',
      category: 'quality',
    },
  ];

  const filteredPolicies = policies.filter((policy) => {
    const matchesSearch =
      !searchTerm ||
      policy.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      policy.description.toLowerCase().includes(searchTerm.toLowerCase());

    const matchesStatus =
      filterStatus === 'all' ||
      (filterStatus === 'enabled' && policy.enabled) ||
      (filterStatus === 'disabled' && !policy.enabled);

    return matchesSearch && matchesStatus;
  });

  const getPriorityColor = (priority: PolicyPriority) => {
    const colors = {
      critical: 'bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400 border-red-200 dark:border-red-800',
      high: 'bg-orange-100 dark:bg-orange-900/30 text-orange-700 dark:text-orange-400 border-orange-200 dark:border-orange-800',
      medium: 'bg-amber-100 dark:bg-amber-900/30 text-amber-700 dark:text-amber-400 border-amber-200 dark:border-amber-800',
      low: 'bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-400 border-gray-200 dark:border-gray-600',
    };
    return colors[priority];
  };

  const getCategoryColor = (category: PolicyTemplate['category']) => {
    const colors = {
      security: 'bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400',
      cost_control: 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400',
      access_control: 'bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-400',
      quality: 'bg-purple-100 dark:bg-purple-900/30 text-purple-700 dark:text-purple-400',
    };
    return colors[category];
  };

  const tabs = [
    { id: 'policies', label: 'Policies', count: policies.length },
    { id: 'templates', label: 'Templates', count: templates.length },
    { id: 'builder', label: 'Policy Builder' },
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
            </div>
          </div>

          {/* Policies List */}
          <div className="space-y-3">
            {filteredPolicies.map((policy) => (
              <div
                key={policy.id}
                className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700 hover:shadow-md transition-shadow"
              >
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <div className="flex items-center gap-3 mb-2">
                      <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                        {policy.name}
                      </h3>
                      <span className={`px-2 py-0.5 rounded-full text-xs font-medium border ${getPriorityColor(policy.priority)}`}>
                        {policy.priority}
                      </span>
                      {!policy.enabled && (
                        <span className="px-2 py-0.5 rounded-full text-xs font-medium bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400">
                          Disabled
                        </span>
                      )}
                    </div>
                    <p className="text-gray-600 dark:text-gray-400 mb-4">{policy.description}</p>

                    {/* Conditions */}
                    <div className="mb-4">
                      <p className="text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider mb-2">
                        Conditions
                      </p>
                      <div className="flex flex-wrap gap-2">
                        {policy.conditions.map((condition, index) => (
                          <span
                            key={index}
                            className="inline-flex items-center gap-1 px-3 py-1 bg-blue-50 dark:bg-blue-900/20 text-blue-700 dark:text-blue-300 rounded-lg text-sm"
                          >
                            <span className="font-medium">{condition.type}</span>
                            <span className="text-blue-400 dark:text-blue-500">{condition.operator}</span>
                            <span className="font-mono">{condition.value}</span>
                          </span>
                        ))}
                      </div>
                    </div>

                    {/* Actions */}
                    <div>
                      <p className="text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider mb-2">
                        Actions
                      </p>
                      <div className="flex flex-wrap gap-2">
                        {policy.actions.map((action, index) => (
                          <span
                            key={index}
                            className="inline-flex items-center gap-1 px-3 py-1 bg-green-50 dark:bg-green-900/20 text-green-700 dark:text-green-300 rounded-lg text-sm"
                          >
                            <span className="font-medium">{action.type}</span>
                            {action.parameter && (
                              <>
                                <span className="text-green-400 dark:text-green-500">:</span>
                                <span className="font-mono">{action.parameter}</span>
                              </>
                            )}
                          </span>
                        ))}
                      </div>
                    </div>
                  </div>

                  {/* Actions */}
                  <div className="flex items-center gap-2 ml-4">
                    <button className="p-2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300">
                      <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                      </svg>
                    </button>
                    <button className="p-2 text-gray-400 hover:text-red-600 dark:hover:text-red-400">
                      <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                      </svg>
                    </button>
                    <button
                      className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                        policy.enabled ? 'bg-blue-600' : 'bg-gray-300 dark:bg-gray-600'
                      }`}
                      role="switch"
                      aria-checked={policy.enabled}
                    >
                      <span
                        className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                          policy.enabled ? 'translate-x-6' : 'translate-x-1'
                        }`}
                      />
                    </button>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Templates Tab */}
      {activeTab === 'templates' && (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {templates.map((template) => (
            <div
              key={template.id}
              className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700 hover:shadow-md transition-shadow cursor-pointer"
              onClick={() => {
                setActiveTab('builder');
              }}
            >
              <div className="flex items-start justify-between mb-4">
                <div className="p-3 bg-blue-100 dark:bg-blue-900/30 rounded-lg">
                  <svg className="w-6 h-6 text-blue-600 dark:text-blue-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                  </svg>
                </div>
                <span className={`px-2 py-1 rounded-full text-xs font-medium ${getCategoryColor(template.category)}`}>
                  {template.category.replace('_', ' ')}
                </span>
              </div>
              <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-2">
                {template.name}
              </h3>
              <p className="text-gray-600 dark:text-gray-400 text-sm">{template.description}</p>
            </div>
          ))}

          {/* Add Custom Template */}
          <div className="bg-gray-50 dark:bg-gray-700/50 rounded-xl p-6 border-2 border-dashed border-gray-300 dark:border-gray-600 flex flex-col items-center justify-center text-center cursor-pointer hover:border-blue-500 dark:hover:border-blue-500 transition-colors">
            <div className="p-3 bg-gray-200 dark:bg-gray-600 rounded-full mb-3">
              <svg className="w-6 h-6 text-gray-600 dark:text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
              </svg>
            </div>
            <p className="font-medium text-gray-900 dark:text-gray-100">Create Custom Template</p>
            <p className="text-sm text-gray-600 dark:text-gray-400">Build a policy from scratch</p>
          </div>
        </div>
      )}

      {/* Builder Tab */}
      {activeTab === 'builder' && (
        <div className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700">
          <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-6">Policy Builder</h2>

          <div className="space-y-6">
            {/* Basic Info */}
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  Policy Name
                </label>
                <input
                  type="text"
                  placeholder="Enter policy name"
                  className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  Priority
                </label>
                <select className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100">
                  <option value="low">Low</option>
                  <option value="medium">Medium</option>
                  <option value="high">High</option>
                  <option value="critical">Critical</option>
                </select>
              </div>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                Description
              </label>
              <textarea
                rows={2}
                placeholder="Describe what this policy does"
                className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>

            {/* Conditions Builder */}
            <div>
              <div className="flex items-center justify-between mb-3">
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                  Conditions
                </label>
                <button className="text-sm text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300">
                  + Add Condition
                </button>
              </div>
              <div className="p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg space-y-3">
                <div className="flex items-center gap-3 p-3 bg-white dark:bg-gray-800 rounded-lg">
                  <select className="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 text-sm">
                    <option value="user">User</option>
                    <option value="group">Group</option>
                    <option value="printer">Printer</option>
                    <option value="time">Time</option>
                    <option value="document_type">Document Type</option>
                    <option value="page_count">Page Count</option>
                  </select>
                  <select className="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 text-sm">
                    <option value="equals">equals</option>
                    <option value="contains">contains</option>
                    <option value="not_equals">not equals</option>
                    <option value="greater_than">greater than</option>
                    <option value="less_than">less than</option>
                    <option value="between">between</option>
                  </select>
                  <input
                    type="text"
                    placeholder="Value"
                    className="flex-1 px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 text-sm"
                  />
                  <button className="p-2 text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-lg">
                    <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                    </svg>
                  </button>
                </div>
                <p className="text-sm text-gray-500 dark:text-gray-400 text-center">
                  Add conditions to define when this policy should apply
                </p>
              </div>
            </div>

            {/* Actions Builder */}
            <div>
              <div className="flex items-center justify-between mb-3">
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                  Actions
                </label>
                <button className="text-sm text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300">
                  + Add Action
                </button>
              </div>
              <div className="p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg space-y-3">
                <div className="flex items-center gap-3 p-3 bg-white dark:bg-gray-800 rounded-lg">
                  <select className="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 text-sm">
                    <option value="allow">Allow</option>
                    <option value="deny">Deny</option>
                    <option value="redirect">Redirect</option>
                    <option value="modify">Modify</option>
                    <option value="notify">Notify</option>
                  </select>
                  <input
                    type="text"
                    placeholder="Optional parameter"
                    className="flex-1 px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 text-sm"
                  />
                  <button className="p-2 text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-lg">
                    <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                    </svg>
                  </button>
                </div>
                <p className="text-sm text-gray-500 dark:text-gray-400 text-center">
                  Define what actions to take when conditions are met
                </p>
              </div>
            </div>

            {/* Actions */}
            <div className="flex items-center gap-3 pt-4 border-t border-gray-200 dark:border-gray-700">
              <button className="px-6 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-medium transition-colors">
                Create Policy
              </button>
              <button
                onClick={() => {
                  setActiveTab('policies');
                }}
                className="px-6 py-2 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700 rounded-lg font-medium transition-colors"
              >
                Cancel
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

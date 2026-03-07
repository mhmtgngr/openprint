/**
 * AgentDetail Component
 * Detailed view of an agent with discovered printers, job history, and health metrics
 */

import { useState } from 'react';
import { formatDistanceToNow } from 'date-fns';
import type { AgentDetail as AgentDetailType } from '@/types/agents';
import { AgentStatusBadge } from '@/components/AgentStatusBadge';
import { AgentPrinters } from './AgentPrinters';
import { AgentHealthChart } from './AgentHealthChart';
import { useTriggerDiscovery } from './useAgents';

interface AgentDetailProps {
  agent: AgentDetailType;
  isLoading?: boolean;
}

export const AgentDetail = ({ agent, isLoading }: AgentDetailProps) => {
  const [activeTab, setActiveTab] = useState<'overview' | 'printers' | 'jobs' | 'health'>(
    'overview'
  );
  const triggerDiscoveryMutation = useTriggerDiscovery();

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="bg-gray-100 dark:bg-gray-800 rounded-lg h-48 animate-pulse" />
        <div className="bg-gray-100 dark:bg-gray-800 rounded-lg h-64 animate-pulse" />
      </div>
    );
  }

  const handleTriggerDiscovery = async () => {
    await triggerDiscoveryMutation.mutateAsync(agent.id);
  };

  const getPlatformIcon = (platform: string) => {
    switch (platform.toLowerCase()) {
      case 'windows':
        return (
          <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
            <path d="M0 3.449L9.75 2.1v9.451H0m10.949-9.602L24 0v11.4H10.949M0 12.6h9.75v9.451L0 20.699M10.949 12.6H24V24l-12.9-1.801" />
          </svg>
        );
      case 'linux':
        return (
          <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
            <path d="M19.78 2.2c-.29-.14-.63-.06-.86.17l-2.32 2.32c-.14.14-.22.34-.22.54v.72c-1.34.66-2.29 1.95-2.45 3.47h-.18c-.18 0-.34.08-.46.2l-1.16 1.16c-.12.12-.2.29-.2.46v1.39h-1.39c-.18 0-.34.08-.46.2l-1.16 1.16c-.12.12-.2.29-.2.46v2.08H6.92c-.18 0-.34.08-.46.2l-1.16 1.16c-.12.12-.2.29-.2.46v.69H2.77c-.18 0-.34.08-.46.2L.2 19.94c-.12.12-.2.29-.2.46v2.08c0 .36.29.65.65.65h2.08c.18 0 .34-.08.46-.2l1.16-1.16c.12-.12.2-.29.2-.46v-.69h1.39c.18 0 .34-.08.46-.2l1.16-1.16c.12-.12.2-.29.2-.46v-2.08h1.39c.18 0 .34-.08.46-.2l1.16-1.16c.12-.12.2-.29.2-.46v-1.39h1.39c.18 0 .34-.08.46-.2l1.16-1.16c.12-.12.2-.29.2-.46v-.72c1.52-.16 2.81-1.11 3.47-2.45h.72c.2 0 .39-.08.54-.22l2.32-2.32c.23-.23.31-.57.17-.86-.14-.29-.45-.43-.74-.35zM16.5 8c-.83 0-1.5-.67-1.5-1.5S15.67 5 16.5 5s1.5.67 1.5 1.5S17.33 8 16.5 8z" />
          </svg>
        );
      case 'darwin':
      case 'macos':
        return (
          <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
            <path d="M18.71 19.5c-.83 1.24-1.71 2.45-3.05 2.47-1.34.03-1.77-.79-3.29-.79-1.53 0-2 .77-3.27.82-1.31.05-2.3-1.32-3.14-2.53C4.25 17 2.94 12.45 4.7 9.39c.87-1.52 2.43-2.48 4.12-2.51 1.28-.02 2.5.87 3.29.87.78 0 2.26-1.07 3.81-.91.65.03 2.47.26 3.64 1.98-.09.06-2.17 1.28-2.15 3.81.03 3.02 2.65 4.03 2.68 4.04-.03.07-.42 1.44-1.38 2.83M13 3.5c.73-.83 1.94-1.46 2.94-1.5.13 1.17-.34 2.35-1.04 3.19-.69.85-1.83 1.51-2.95 1.42-.15-1.15.41-2.35 1.05-3.11z" />
          </svg>
        );
      default:
        return (
          <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
            />
          </svg>
        );
    }
  };

  const tabs = [
    { id: 'overview' as const, label: 'Overview', count: null },
    { id: 'printers' as const, label: 'Printers', count: agent.printers.length },
    { id: 'jobs' as const, label: 'Jobs', count: agent.jobHistory.length },
    { id: 'health' as const, label: 'Health', count: null },
  ];

  const renderOverview = () => (
    <div className="space-y-6">
      {/* Agent info cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <div className="bg-white dark:bg-gray-800 rounded-lg p-4 border border-gray-200 dark:border-gray-700">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-600 dark:text-gray-400">Status</p>
              <p className="mt-1 text-lg font-semibold text-gray-900 dark:text-gray-100">
                <AgentStatusBadge status={agent.status} />
              </p>
            </div>
            <div className="text-gray-400">{getPlatformIcon(agent.platform)}</div>
          </div>
        </div>

        <div className="bg-white dark:bg-gray-800 rounded-lg p-4 border border-gray-200 dark:border-gray-700">
          <p className="text-sm text-gray-600 dark:text-gray-400">Printers</p>
          <p className="mt-1 text-lg font-semibold text-gray-900 dark:text-gray-100">
            {agent.printers.length}
          </p>
        </div>

        <div className="bg-white dark:bg-gray-800 rounded-lg p-4 border border-gray-200 dark:border-gray-700">
          <p className="text-sm text-gray-600 dark:text-gray-400">Jobs Processed</p>
          <p className="mt-1 text-lg font-semibold text-gray-900 dark:text-gray-100">
            {agent.healthMetrics.totalJobsProcessed}
          </p>
        </div>

        <div className="bg-white dark:bg-gray-800 rounded-lg p-4 border border-gray-200 dark:border-gray-700">
          <p className="text-sm text-gray-600 dark:text-gray-400">Success Rate</p>
          <p className="mt-1 text-lg font-semibold text-gray-900 dark:text-gray-100">
            {agent.healthMetrics.successRate.toFixed(1)}%
          </p>
        </div>
      </div>

      {/* Detailed info */}
      <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
        <div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
          <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100">
            Agent Information
          </h3>
        </div>
        <dl className="divide-y divide-gray-200 dark:divide-gray-700">
          <div className="px-6 py-4 grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <dt className="text-sm text-gray-600 dark:text-gray-400">Agent Name</dt>
              <dd className="mt-1 text-sm font-medium text-gray-900 dark:text-gray-100">
                {agent.name}
              </dd>
            </div>
            <div>
              <dt className="text-sm text-gray-600 dark:text-gray-400">Platform</dt>
              <dd className="mt-1 text-sm font-medium text-gray-900 dark:text-gray-100">
                {agent.platform} {agent.platformVersion && `(${agent.platformVersion})`}
              </dd>
            </div>
            <div>
              <dt className="text-sm text-gray-600 dark:text-gray-400">Agent Version</dt>
              <dd className="mt-1 text-sm font-medium text-gray-900 dark:text-gray-100">
                {agent.agentVersion || 'Unknown'}
              </dd>
            </div>
            <div>
              <dt className="text-sm text-gray-600 dark:text-gray-400">IP Address</dt>
              <dd className="mt-1 text-sm font-medium text-gray-900 dark:text-gray-100">
                {agent.ipAddress || 'Unknown'}
              </dd>
            </div>
            <div>
              <dt className="text-sm text-gray-600 dark:text-gray-400">Last Heartbeat</dt>
              <dd className="mt-1 text-sm font-medium text-gray-900 dark:text-gray-100">
                {agent.lastHeartbeat
                  ? formatDistanceToNow(new Date(agent.lastHeartbeat), { addSuffix: true })
                  : 'Never'}
              </dd>
            </div>
            <div>
              <dt className="text-sm text-gray-600 dark:text-gray-400">Session State</dt>
              <dd className="mt-1 text-sm font-medium text-gray-900 dark:text-gray-100">
                {agent.sessionState || 'Unknown'}
              </dd>
            </div>
            {agent.associatedUser && (
              <>
                <div>
                  <dt className="text-sm text-gray-600 dark:text-gray-400">Associated User</dt>
                  <dd className="mt-1 text-sm font-medium text-gray-900 dark:text-gray-100">
                    {agent.associatedUser.name}
                  </dd>
                </div>
                <div>
                  <dt className="text-sm text-gray-600 dark:text-gray-400">User Email</dt>
                  <dd className="mt-1 text-sm font-medium text-gray-900 dark:text-gray-100">
                    {agent.associatedUser.email}
                  </dd>
                </div>
              </>
            )}
          </div>
        </dl>
      </div>

      {/* Capabilities */}
      <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
        <div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
          <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100">
            Capabilities
          </h3>
        </div>
        <div className="p-6">
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div className="flex items-center gap-2">
              {agent.capabilities?.supportsColor ? (
                <svg className="w-5 h-5 text-green-500" fill="currentColor" viewBox="0 0 20 20">
                  <path
                    fillRule="evenodd"
                    d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
                    clipRule="evenodd"
                  />
                </svg>
              ) : (
                <svg className="w-5 h-5 text-gray-400" fill="currentColor" viewBox="0 0 20 20">
                  <path
                    fillRule="evenodd"
                    d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
                    clipRule="evenodd"
                  />
                </svg>
              )}
              <span className="text-sm text-gray-700 dark:text-gray-300">Color Printing</span>
            </div>
            <div className="flex items-center gap-2">
              {agent.capabilities?.supportsDuplex ? (
                <svg className="w-5 h-5 text-green-500" fill="currentColor" viewBox="0 0 20 20">
                  <path
                    fillRule="evenodd"
                    d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
                    clipRule="evenodd"
                  />
                </svg>
              ) : (
                <svg className="w-5 h-5 text-gray-400" fill="currentColor" viewBox="0 0 20 20">
                  <path
                    fillRule="evenodd"
                    d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
                    clipRule="evenodd"
                  />
                </svg>
              )}
              <span className="text-sm text-gray-700 dark:text-gray-300">Duplex Printing</span>
            </div>
            <div className="flex items-center gap-2">
              {agent.capabilities?.supportsLargeFormat ? (
                <svg className="w-5 h-5 text-green-500" fill="currentColor" viewBox="0 0 20 20">
                  <path
                    fillRule="evenodd"
                    d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
                    clipRule="evenodd"
                  />
                </svg>
              ) : (
                <svg className="w-5 h-5 text-gray-400" fill="currentColor" viewBox="0 0 20 20">
                  <path
                    fillRule="evenodd"
                    d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
                    clipRule="evenodd"
                  />
                </svg>
              )}
              <span className="text-sm text-gray-700 dark:text-gray-300">Large Format</span>
            </div>
            <div>
              <span className="text-sm text-gray-600 dark:text-gray-400">Max Job Size: </span>
              <span className="text-sm font-medium text-gray-900 dark:text-gray-100">
                {(agent.capabilities.maxJobSize / (1024 * 1024)).toFixed(0)} MB
              </span>
            </div>
          </div>
          {agent.capabilities.supportedFormats.length > 0 && (
            <div className="mt-4">
              <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">Supported Formats:</p>
              <div className="flex flex-wrap gap-2">
                {agent.capabilities.supportedFormats.map((format) => (
                  <span
                    key={format}
                    className="px-2 py-1 text-xs font-medium bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded"
                  >
                    {format}
                  </span>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );

  const renderJobs = () => (
    <div className="space-y-4">
      {agent.jobHistory.length === 0 ? (
        <div className="text-center py-12">
          <svg
            className="mx-auto h-12 w-12 text-gray-400"
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
          <h3 className="mt-2 text-sm font-medium text-gray-900 dark:text-gray-100">
            No job history
          </h3>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            This agent hasn't processed any jobs yet.
          </p>
        </div>
      ) : (
        <div className="overflow-hidden">
          <ul className="divide-y divide-gray-200 dark:divide-gray-700">
            {agent.jobHistory.map((entry) => (
              <li key={entry.id} className="py-4 px-4">
                <div className="flex items-center justify-between">
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-3">
                      <p className="text-sm font-medium text-gray-900 dark:text-gray-100 truncate">
                        {entry.documentName}
                      </p>
                      <span
                        className={`inline-flex items-center px-2 py-1 rounded text-xs font-medium ${
                          entry.status === 'completed'
                            ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300'
                            : entry.status === 'failed'
                            ? 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300'
                            : 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300'
                        }`}
                      >
                        {entry.status}
                      </span>
                    </div>
                    <div className="mt-1 flex items-center gap-4 text-sm text-gray-500 dark:text-gray-400">
                      <span>{entry.printerName}</span>
                      <span>{entry.pages} pages</span>
                      <span>
                        {formatDistanceToNow(new Date(entry.timestamp), { addSuffix: true })}
                      </span>
                    </div>
                    {entry.errorMessage && (
                      <p className="mt-1 text-sm text-red-600 dark:text-red-400">
                        {entry.errorMessage}
                      </p>
                    )}
                  </div>
                </div>
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );

  const renderHealth = () => <AgentHealthChart metrics={agent.healthMetrics} />;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <div className="text-gray-500">{getPlatformIcon(agent.platform)}</div>
          <div>
            <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100">
              {agent.name}
            </h2>
            <p className="text-sm text-gray-600 dark:text-gray-400">
              {agent.platform} {agent.platformVersion && `(${agent.platformVersion})`}
              {agent.agentVersion && ` • Agent v${agent.agentVersion}`}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-3">
          <AgentStatusBadge status={agent.status} />
          {agent.status === 'online' && (
            <button
              onClick={handleTriggerDiscovery}
              disabled={triggerDiscoveryMutation.isPending}
              className="px-3 py-2 text-sm font-medium text-blue-600 bg-blue-50 hover:bg-blue-100 dark:bg-blue-900/20 dark:text-blue-400 dark:hover:bg-blue-900/30 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {triggerDiscoveryMutation.isPending ? 'Scanning...' : 'Discover Printers'}
            </button>
          )}
        </div>
      </div>

      {/* Tabs */}
      <div className="border-b border-gray-200 dark:border-gray-700">
        <nav className="flex gap-8 -mb-px">
          {tabs.map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={`
                flex items-center gap-2 px-1 py-4 text-sm font-medium border-b-2 transition-colors
                ${
                  activeTab === tab.id
                    ? 'border-blue-500 text-blue-600 dark:text-blue-400'
                    : 'border-transparent text-gray-600 hover:text-gray-900 hover:border-gray-300 dark:text-gray-400 dark:hover:text-gray-100'
                }
              `}
            >
              {tab.label}
              {tab.count !== null && (
                <span
                  className={`px-2 py-0.5 text-xs rounded-full ${
                    activeTab === tab.id
                      ? 'bg-blue-100 text-blue-600 dark:bg-blue-900/30 dark:text-blue-300'
                      : 'bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400'
                  }`}
                >
                  {tab.count}
                </span>
              )}
            </button>
          ))}
        </nav>
      </div>

      {/* Tab content */}
      <div className="mt-6">
        {activeTab === 'overview' && renderOverview()}
        {activeTab === 'printers' && <AgentPrinters agentId={agent.id} />}
        {activeTab === 'jobs' && renderJobs()}
        {activeTab === 'health' && renderHealth()}
      </div>
    </div>
  );
};

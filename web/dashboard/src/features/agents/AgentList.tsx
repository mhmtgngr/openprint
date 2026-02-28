/**
 * AgentList Component
 * Displays all registered agents with status, last heartbeat, and associated user
 */

import { formatDistanceToNow } from 'date-fns';
import type { Agent, AgentListParams } from '@/types/agents';
import { AgentStatusBadge } from '@/components/AgentStatusBadge';
import { useDeleteAgent, useRestartAgent } from './useAgents';

interface AgentListProps {
  agents: Agent[];
  isLoading?: boolean;
  onAgentClick?: (agent: Agent) => void;
  onFilterChange?: (params: AgentListParams) => void;
  currentFilter?: AgentListParams;
}

export const AgentList = ({
  agents,
  isLoading,
  onAgentClick,
  onFilterChange,
  currentFilter,
}: AgentListProps) => {
  const deleteAgentMutation = useDeleteAgent();
  const restartAgentMutation = useRestartAgent();

  if (isLoading) {
    return (
      <div className="space-y-3">
        {[...Array(5)].map((_, i) => (
          <div
            key={i}
            className="bg-gray-100 dark:bg-gray-800 rounded-lg h-28 animate-pulse"
          />
        ))}
      </div>
    );
  }

  if (agents.length === 0) {
    return (
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
            d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
          />
        </svg>
        <h3 className="mt-2 text-sm font-medium text-gray-900 dark:text-gray-100">
          No agents found
        </h3>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          Install the OpenPrint Agent on Windows machines to get started.
        </p>
      </div>
    );
  }

  const handleDelete = async (e: React.MouseEvent, agentId: string, agentName: string) => {
    e.stopPropagation();
    if (
      confirm(
        `Are you sure you want to delete agent "${agentName}"? This action cannot be undone.`
      )
    ) {
      await deleteAgentMutation.mutateAsync(agentId);
    }
  };

  const handleRestart = async (e: React.MouseEvent, agentId: string) => {
    e.stopPropagation();
    await restartAgentMutation.mutateAsync(agentId);
  };

  const getPlatformIcon = (platform: string) => {
    switch (platform.toLowerCase()) {
      case 'windows':
        return (
          <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
            <path d="M0 3.449L9.75 2.1v9.451H0m10.949-9.602L24 0v11.4H10.949M0 12.6h9.75v9.451L0 20.699M10.949 12.6H24V24l-12.9-1.801" />
          </svg>
        );
      case 'linux':
        return (
          <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
            <path d="M19.78 2.2c-.29-.14-.63-.06-.86.17l-2.32 2.32c-.14.14-.22.34-.22.54v.72c-1.34.66-2.29 1.95-2.45 3.47h-.18c-.18 0-.34.08-.46.2l-1.16 1.16c-.12.12-.2.29-.2.46v1.39h-1.39c-.18 0-.34.08-.46.2l-1.16 1.16c-.12.12-.2.29-.2.46v2.08H6.92c-.18 0-.34.08-.46.2l-1.16 1.16c-.12.12-.2.29-.2.46v.69H2.77c-.18 0-.34.08-.46.2L.2 19.94c-.12.12-.2.29-.2.46v2.08c0 .36.29.65.65.65h2.08c.18 0 .34-.08.46-.2l1.16-1.16c.12-.12.2-.29.2-.46v-.69h1.39c.18 0 .34-.08.46-.2l1.16-1.16c.12-.12.2-.29.2-.46v-2.08h1.39c.18 0 .34-.08.46-.2l1.16-1.16c.12-.12.2-.29.2-.46v-1.39h1.39c.18 0 .34-.08.46-.2l1.16-1.16c.12-.12.2-.29.2-.46v-.72c1.52-.16 2.81-1.11 3.47-2.45h.72c.2 0 .39-.08.54-.22l2.32-2.32c.23-.23.31-.57.17-.86-.14-.29-.45-.43-.74-.35zM16.5 8c-.83 0-1.5-.67-1.5-1.5S15.67 5 16.5 5s1.5.67 1.5 1.5S17.33 8 16.5 8z" />
          </svg>
        );
      case 'darwin':
      case 'macos':
        return (
          <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
            <path d="M18.71 19.5c-.83 1.24-1.71 2.45-3.05 2.47-1.34.03-1.77-.79-3.29-.79-1.53 0-2 .77-3.27.82-1.31.05-2.3-1.32-3.14-2.53C4.25 17 2.94 12.45 4.7 9.39c.87-1.52 2.43-2.48 4.12-2.51 1.28-.02 2.5.87 3.29.87.78 0 2.26-1.07 3.81-.91.65.03 2.47.26 3.64 1.98-.09.06-2.17 1.28-2.15 3.81.03 3.02 2.65 4.03 2.68 4.04-.03.07-.42 1.44-1.38 2.83M13 3.5c.73-.83 1.94-1.46 2.94-1.5.13 1.17-.34 2.35-1.04 3.19-.69.85-1.83 1.51-2.95 1.42-.15-1.15.41-2.35 1.05-3.11z" />
          </svg>
        );
      default:
        return (
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
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

  const getStatusCounts = () => {
    const online = agents.filter((a) => a.status === 'online').length;
    const offline = agents.filter((a) => a.status === 'offline').length;
    const error = agents.filter((a) => a.status === 'error').length;
    return { online, offline, error };
  };

  const statusCounts = getStatusCounts();

  return (
    <div className="space-y-4">
      {/* Status summary */}
      <div className="flex items-center gap-4 text-sm">
        <span className="text-gray-600 dark:text-gray-400">
          {agents.length} {agents.length === 1 ? 'agent' : 'agents'}
        </span>
        <span className="flex items-center gap-1">
          <span className="w-2 h-2 rounded-full bg-green-500" />
          {statusCounts.online} online
        </span>
        <span className="flex items-center gap-1">
          <span className="w-2 h-2 rounded-full bg-gray-400" />
          {statusCounts.offline} offline
        </span>
        {statusCounts.error > 0 && (
          <span className="flex items-center gap-1">
            <span className="w-2 h-2 rounded-full bg-red-500" />
            {statusCounts.error} errors
          </span>
        )}
      </div>

      {/* Filter buttons */}
      {onFilterChange && (
        <div className="flex items-center gap-2">
          <button
            onClick={() => onFilterChange({ ...currentFilter, status: 'all' })}
            className={`px-3 py-1 text-xs font-medium rounded-full transition-colors ${
              !currentFilter?.status || currentFilter.status === 'all'
                ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300'
                : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-gray-800 dark:text-gray-400 dark:hover:bg-gray-700'
            }`}
          >
            All
          </button>
          <button
            onClick={() => onFilterChange({ ...currentFilter, status: 'online' })}
            className={`px-3 py-1 text-xs font-medium rounded-full transition-colors ${
              currentFilter?.status === 'online'
                ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300'
                : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-gray-800 dark:text-gray-400 dark:hover:bg-gray-700'
            }`}
          >
            Online
          </button>
          <button
            onClick={() => onFilterChange({ ...currentFilter, status: 'offline' })}
            className={`px-3 py-1 text-xs font-medium rounded-full transition-colors ${
              currentFilter?.status === 'offline'
                ? 'bg-gray-200 text-gray-700 dark:bg-gray-700 dark:text-gray-300'
                : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-gray-800 dark:text-gray-400 dark:hover:bg-gray-700'
            }`}
          >
            Offline
          </button>
          <button
            onClick={() => onFilterChange({ ...currentFilter, status: 'error' })}
            className={`px-3 py-1 text-xs font-medium rounded-full transition-colors ${
              currentFilter?.status === 'error'
                ? 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300'
                : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-gray-800 dark:text-gray-400 dark:hover:bg-gray-700'
            }`}
          >
            Errors
          </button>
        </div>
      )}

      {/* Agent list */}
      <div className="overflow-hidden">
        <ul className="divide-y divide-gray-200 dark:divide-gray-700">
          {agents.map((agent) => (
            <li
              key={agent.id}
              className={`
                py-4 px-4 hover:bg-gray-50 dark:hover:bg-gray-800/50 transition-colors
                ${onAgentClick ? 'cursor-pointer' : ''}
              `}
              onClick={() => onAgentClick?.(agent)}
            >
              <div className="flex items-center justify-between">
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-3">
                    <div className="flex items-center gap-2 text-gray-500 dark:text-gray-400">
                      {getPlatformIcon(agent.platform)}
                      <span className="text-xs">{agent.platform}</span>
                    </div>
                    <p className="text-sm font-medium text-gray-900 dark:text-gray-100 truncate">
                      {agent.name}
                    </p>
                    <AgentStatusBadge status={agent.status} />
                  </div>

                  <div className="mt-1 flex flex-wrap items-center gap-4 text-sm text-gray-500 dark:text-gray-400">
                    {agent.associatedUser && (
                      <span className="flex items-center gap-1">
                        <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path
                            strokeLinecap="round"
                            strokeLinejoin="round"
                            strokeWidth={2}
                            d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"
                          />
                        </svg>
                        {agent.associatedUser.name}
                      </span>
                    )}
                    {agent.printerCount !== undefined && (
                      <span className="flex items-center gap-1">
                        <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path
                            strokeLinecap="round"
                            strokeLinejoin="round"
                            strokeWidth={2}
                            d="M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2h2m2 4h6a2 2 0 002-2v-4a2 2 0 00-2-2H9a2 2 0 00-2 2v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z"
                          />
                        </svg>
                        {agent.printerCount} {agent.printerCount === 1 ? 'printer' : 'printers'}
                      </span>
                    )}
                    {agent.jobQueueDepth !== undefined && agent.jobQueueDepth > 0 && (
                      <span className="flex items-center gap-1">
                        <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path
                            strokeLinecap="round"
                            strokeLinejoin="round"
                            strokeWidth={2}
                            d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2"
                          />
                        </svg>
                        {agent.jobQueueDepth} queued
                      </span>
                    )}
                    <span className="flex items-center gap-1">
                      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          strokeWidth={2}
                          d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
                        />
                      </svg>
                      {agent.lastHeartbeat
                        ? formatDistanceToNow(new Date(agent.lastHeartbeat), { addSuffix: true })
                        : 'Never'}
                    </span>
                    {agent.agentVersion && (
                      <span className="text-xs">v{agent.agentVersion}</span>
                    )}
                  </div>
                </div>

                <div className="ml-4 flex items-center gap-2">
                  {agent.status === 'error' && (
                    <button
                      onClick={(e) => handleRestart(e, agent.id)}
                      className="p-2 text-gray-400 hover:text-blue-600 dark:hover:text-blue-400 transition-colors"
                      title="Restart agent"
                    >
                      <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          strokeWidth={2}
                          d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
                        />
                      </svg>
                    </button>
                  )}
                  <button
                    onClick={(e) => handleDelete(e, agent.id, agent.name)}
                    className="p-2 text-gray-400 hover:text-red-600 dark:hover:text-red-400 transition-colors"
                    title="Delete agent"
                  >
                    <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                      />
                    </svg>
                  </button>
                  <svg
                    className="w-5 h-5 text-gray-400"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                  >
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                  </svg>
                </div>
              </div>
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
};

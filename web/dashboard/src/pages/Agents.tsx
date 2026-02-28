/**
 * Agents Page
 * Main page for viewing and managing all agents
 */

import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAgents } from '@/features/agents';
import { AgentList } from '@/features/agents';
import type { AgentListParams } from '@/types/agents';
import type { Agent } from '@/types/agents';

export const Agents = () => {
  const navigate = useNavigate();
  const [filterParams, setFilterParams] = useState<AgentListParams>({});

  const { data: agents, isLoading, error } = useAgents(filterParams);

  const handleAgentClick = (agent: Agent) => {
    navigate(`/agents/${agent.id}`);
  };

  if (error) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center">
          <svg
            className="mx-auto h-12 w-12 text-red-400"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
            />
          </svg>
          <h2 className="mt-4 text-lg font-medium text-gray-900 dark:text-gray-100">
            Error loading agents
          </h2>
          <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">
            {(error as Error).message}
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <div className="md:flex md:items-center md:justify-between mb-6">
        <div className="flex-1 min-w-0">
          <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">Agents</h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Manage Windows print agents and monitor their status
          </p>
        </div>
      </div>

      <AgentList
        agents={agents || []}
        isLoading={isLoading}
        onAgentClick={handleAgentClick}
        onFilterChange={setFilterParams}
        currentFilter={filterParams}
      />
    </div>
  );
};

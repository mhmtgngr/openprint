import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { ActivityIcon, CheckIcon, XCircleIcon, ClockIcon } from '@/components/icons';

interface AgentHealth {
  agent_id: string;
  name: string;
  status: 'online' | 'offline' | 'error';
  last_heartbeat: string;
  uptime_seconds: number;
  cpu_usage?: number;
  memory_usage?: number;
  memory_total?: number;
  job_success_rate?: number;
  queue_depth?: number;
  active_connections?: number;
}

export const AgentHealth = () => {
  const [selectedTimeRange, setSelectedTimeRange] = useState('1h');

  // Fetch all agents
  const { data: agentsData } = useQuery({
    queryKey: ['agents'],
    queryFn: async () => {
      const res = await fetch('/api/v1/agents');
      if (!res.ok) throw new Error('Failed to fetch agents');
      return res.json();
    },
    refetchInterval: 30000, // Refresh every 30 seconds
  });

  const agents = agentsData?.agents || [];

  // Calculate health metrics
  const healthMetrics = {
    total: agents.length,
    online: agents.filter((a: AgentHealth) => a.status === 'online').length,
    offline: agents.filter((a: AgentHealth) => a.status === 'offline').length,
    error: agents.filter((a: AgentHealth) => a.status === 'error').length,
    healthPercentage: agents.length > 0
      ? Math.round((agents.filter((a: AgentHealth) => a.status === 'online').length / agents.length) * 100)
      : 0,
  };

  // Get recent alerts (would be fetched from API in production)
  const recentAlerts = [
    { id: '1', type: 'warning', message: 'Agent prod-print-01 is offline', time: '5 minutes ago' },
    { id: '2', type: 'info', message: 'Agent dev-print-03 needs update', time: '1 hour ago' },
    { id: '3', type: 'critical', message: 'High failure rate on agent prod-print-02', time: '2 hours ago' },
  ];

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'online':
        return 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400 border-green-200 dark:border-green-800';
      case 'offline':
        return 'bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-400 border-gray-200 dark:border-gray-700';
      case 'error':
        return 'bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400 border-red-200 dark:border-red-800';
      default:
        return 'bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-400 border-gray-200 dark:border-gray-700';
    }
  };

  const getHealthBarColor = (percentage: number) => {
    if (percentage >= 90) return 'bg-green-500';
    if (percentage >= 70) return 'bg-yellow-500';
    if (percentage >= 50) return 'bg-orange-500';
    return 'bg-red-500';
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Agent Health</h2>
          <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
            Real-time monitoring of your print agent fleet
          </p>
        </div>
        <select
          value={selectedTimeRange}
          onChange={(e) => setSelectedTimeRange(e.target.value)}
          className="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 text-sm"
        >
          <option value="1h">Last Hour</option>
          <option value="24h">Last 24 Hours</option>
          <option value="7d">Last 7 Days</option>
        </select>
      </div>

      {/* Health Overview Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <div className="bg-white dark:bg-gray-800 rounded-lg p-4 border border-gray-200 dark:border-gray-700">
          <div className="flex items-center gap-3">
            <div className="p-2 bg-blue-100 dark:bg-blue-900/30 rounded-lg text-blue-600 dark:text-blue-400">
              <ActivityIcon className="w-5 h-5" />
            </div>
            <div>
              <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">{healthMetrics.total}</p>
              <p className="text-xs text-gray-500 dark:text-gray-400">Total Agents</p>
            </div>
          </div>
        </div>

        <div className="bg-white dark:bg-gray-800 rounded-lg p-4 border border-gray-200 dark:border-gray-700">
          <div className="flex items-center gap-3">
            <div className="p-2 bg-green-100 dark:bg-green-900/30 rounded-lg text-green-600 dark:text-green-400">
              <CheckIcon className="w-5 h-5" />
            </div>
            <div>
              <p className="text-2xl font-bold text-green-600 dark:text-green-400">{healthMetrics.online}</p>
              <p className="text-xs text-gray-500 dark:text-gray-400">Online</p>
            </div>
          </div>
        </div>

        <div className="bg-white dark:bg-gray-800 rounded-lg p-4 border border-gray-200 dark:border-gray-700">
          <div className="flex items-center gap-3">
            <div className="p-2 bg-gray-100 dark:bg-gray-800 rounded-lg text-gray-500 dark:text-gray-400">
              <XCircleIcon className="w-5 h-5" />
            </div>
            <div>
              <p className="text-2xl font-bold text-gray-600 dark:text-gray-400">{healthMetrics.offline}</p>
              <p className="text-xs text-gray-500 dark:text-gray-400">Offline</p>
            </div>
          </div>
        </div>

        <div className="bg-white dark:bg-gray-800 rounded-lg p-4 border border-gray-200 dark:border-gray-700">
          <div className="flex items-center gap-3">
            <div className="p-2 bg-red-100 dark:bg-red-900/30 rounded-lg text-red-600 dark:text-red-400">
              <ClockIcon className="w-5 h-5" />
            </div>
            <div>
              <p className="text-2xl font-bold text-red-600 dark:text-red-400">{healthMetrics.error}</p>
              <p className="text-xs text-gray-500 dark:text-gray-400">Errors</p>
            </div>
          </div>
        </div>
      </div>

      {/* Fleet Health Bar */}
      <div className="bg-white dark:bg-gray-800 rounded-lg p-4 border border-gray-200 dark:border-gray-700">
        <div className="flex items-center justify-between mb-2">
          <h3 className="font-medium text-gray-900 dark:text-gray-100">Fleet Health</h3>
          <span className="text-2xl font-bold text-gray-900 dark:text-gray-100">{healthMetrics.healthPercentage}%</span>
        </div>
        <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-3">
          <div
            className={`h-3 rounded-full transition-all ${getHealthBarColor(healthMetrics.healthPercentage)}`}
            style={{ width: `${healthMetrics.healthPercentage}%` }}
          />
        </div>
        <p className="text-xs text-gray-500 dark:text-gray-400 mt-2">
          {healthMetrics.online} of {healthMetrics.total} agents operational
        </p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Agent Status List */}
        <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
          <div className="p-4 border-b border-gray-200 dark:border-gray-700">
            <h3 className="font-semibold text-gray-900 dark:text-gray-100">Agent Status</h3>
          </div>
          <div className="divide-y divide-gray-200 dark:divide-gray-700">
            {agents.length === 0 ? (
              <div className="p-6 text-center text-gray-500 dark:text-gray-400">
                No agents found. Install the OpenPrint Agent to get started.
              </div>
            ) : (
              agents.slice(0, 8).map((agent: AgentHealth) => {
                const uptime = agent.uptime_seconds
                  ? `${Math.floor(agent.uptime_seconds / 86400)}d`
                  : 'N/A';
                return (
                  <div key={agent.agent_id} className="p-4 hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors">
                    <div className="flex items-center justify-between">
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2">
                          <p className="text-sm font-medium text-gray-900 dark:text-gray-100 truncate">
                            {agent.name}
                          </p>
                          <span className={`text-xs px-2 py-0.5 rounded border ${getStatusColor(agent.status)}`}>
                            {agent.status}
                          </span>
                        </div>
                        <div className="flex items-center gap-4 mt-1 text-xs text-gray-500 dark:text-gray-400">
                          <span>Uptime: {uptime}</span>
                          {agent.queue_depth !== undefined && (
                            <span>Queue: {agent.queue_depth}</span>
                          )}
                          {agent.job_success_rate !== undefined && (
                            <span>Success: {agent.job_success_rate}%</span>
                          )}
                        </div>
                      </div>
                      <div className="ml-4 flex items-center gap-2">
                        {agent.cpu_usage !== undefined && (
                          <div className="text-right">
                            <p className="text-xs text-gray-500 dark:text-gray-400">CPU</p>
                            <p className="text-sm font-medium text-gray-900 dark:text-gray-100">{agent.cpu_usage}%</p>
                          </div>
                        )}
                        {agent.memory_usage !== undefined && agent.memory_total !== undefined && (
                          <div className="text-right">
                            <p className="text-xs text-gray-500 dark:text-gray-400">RAM</p>
                            <p className="text-sm font-medium text-gray-900 dark:text-gray-100">
                              {Math.round((agent.memory_usage / agent.memory_total) * 100)}%
                            </p>
                          </div>
                        )}
                      </div>
                    </div>
                  </div>
                );
              })
            )}
            {agents.length > 8 && (
              <div className="p-4 text-center">
                <a
                  href="/agents"
                  className="text-sm text-blue-600 dark:text-blue-400 hover:underline"
                >
                  View all {agents.length} agents →
                </a>
              </div>
            )}
          </div>
        </div>

        {/* Recent Alerts */}
        <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
          <div className="p-4 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
            <h3 className="font-semibold text-gray-900 dark:text-gray-100">Recent Alerts</h3>
            <a href="/audit-logs" className="text-sm text-blue-600 dark:text-blue-400 hover:underline">
              View All
            </a>
          </div>
          <div className="divide-y divide-gray-200 dark:divide-gray-700">
            {recentAlerts.length === 0 ? (
              <div className="p-6 text-center text-gray-500 dark:text-gray-400">
                No recent alerts
              </div>
            ) : (
              recentAlerts.map((alert) => (
                <div key={alert.id} className="p-4 flex items-start gap-3">
                  <div className={`p-1.5 rounded ${
                    alert.type === 'critical'
                      ? 'bg-red-100 dark:bg-red-900/30 text-red-600 dark:text-red-400'
                      : alert.type === 'warning'
                      ? 'bg-yellow-100 dark:bg-yellow-900/30 text-yellow-600 dark:text-yellow-400'
                      : 'bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400'
                  }`}>
                    <ActivityIcon className="w-4 h-4" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="text-sm text-gray-900 dark:text-gray-100">{alert.message}</p>
                    <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">{alert.time}</p>
                  </div>
                </div>
              ))
            )}
          </div>
        </div>
      </div>

      {/* Quick Actions */}
      <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
        <h3 className="font-semibold text-gray-900 dark:text-gray-100 mb-4">Quick Actions</h3>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <button className="flex items-center gap-3 p-4 border border-gray-200 dark:border-gray-700 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors text-left">
            <div className="p-2 bg-blue-100 dark:bg-blue-900/30 rounded-lg text-blue-600 dark:text-blue-400">
              <ActivityIcon className="w-5 h-5" />
            </div>
            <div>
              <p className="text-sm font-medium text-gray-900 dark:text-gray-100">Restart Offline Agents</p>
              <p className="text-xs text-gray-500 dark:text-gray-400">Send restart command</p>
            </div>
          </button>

          <button className="flex items-center gap-3 p-4 border border-gray-200 dark:border-gray-700 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors text-left">
            <div className="p-2 bg-green-100 dark:bg-green-900/30 rounded-lg text-green-600 dark:text-green-400">
              <CheckIcon className="w-5 h-5" />
            </div>
            <div>
              <p className="text-sm font-medium text-gray-900 dark:text-gray-100">Check Connectivity</p>
              <p className="text-xs text-gray-500 dark:text-gray-400">Ping all agents</p>
            </div>
          </button>

          <button className="flex items-center gap-3 p-4 border border-gray-200 dark:border-gray-700 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors text-left">
            <div className="p-2 bg-purple-100 dark:bg-purple-900/30 rounded-lg text-purple-600 dark:text-purple-400">
              <ClockIcon className="w-5 h-5" />
            </div>
            <div>
              <p className="text-sm font-medium text-gray-900 dark:text-gray-100">Export Health Report</p>
              <p className="text-xs text-gray-500 dark:text-gray-400">Download CSV</p>
            </div>
          </button>
        </div>
      </div>
    </div>
  );
};

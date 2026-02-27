import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useAuth } from '@/hooks/useAuth';
import { analyticsApi } from '@/services/api';
import { EnvironmentReport } from '@/components/EnvironmentReport';
import {
  BarChart,
  Bar,
  LineChart,
  Line,
  PieChart,
  Pie,
  Cell,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts';

const COLORS = ['#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6', '#ec4899'];

export const Analytics = () => {
  const { user } = useAuth();
  const [period, setPeriod] = useState<'7d' | '30d' | '90d' | '12m'>('30d');
  // TODO: Implement groupBy selector
  // const [groupBy, setGroupBy] = useState<'day' | 'week' | 'month'>('day');
  const groupBy: 'day' | 'week' | 'month' = 'day';

  const { data: usageStats, isLoading: statsLoading } = useQuery({
    queryKey: ['analytics', 'usage', period, groupBy],
    queryFn: () =>
      analyticsApi.getUsage({
        groupBy,
        startDate: getStartDate(period),
        endDate: new Date().toISOString().split('T')[0],
      }),
  });

  const { data: environment } = useQuery({
    queryKey: ['analytics', 'environment', period],
    queryFn: () => analyticsApi.getEnvironment(period),
  });

  const { data: auditLogs } = useQuery({
    queryKey: ['analytics', 'audit-logs'],
    queryFn: () => analyticsApi.getAuditLogs({ limit: 20 }),
  });

  function getStartDate(p: typeof period): string {
    const date = new Date();
    switch (p) {
      case '7d':
        date.setDate(date.getDate() - 7);
        break;
      case '30d':
        date.setDate(date.getDate() - 30);
        break;
      case '90d':
        date.setDate(date.getDate() - 90);
        break;
      case '12m':
        date.setFullYear(date.getFullYear() - 1);
        break;
    }
    return date.toISOString().split('T')[0];
  }

  // Prepare chart data
  const chartData = usageStats?.map((stat) => ({
    date: new Date(stat.statDate).toLocaleDateString('en-US', { month: 'short', day: 'numeric' }),
    pages: stat.pagesPrinted,
    jobs: stat.jobsCount,
    color: stat.colorPages,
    co2: stat.co2Grams,
  })) || [];

  // Status distribution data
  const statusData = usageStats
    ? [
        { name: 'Completed', value: usageStats.reduce((sum, s) => sum + s.jobsCompleted, 0) },
        { name: 'Failed', value: usageStats.reduce((sum, s) => sum + s.jobsFailed, 0) },
      ]
    : [];

  // Totals
  const totalJobs = usageStats?.reduce((sum, s) => sum + s.jobsCount, 0) || 0;
  const totalPages = usageStats?.reduce((sum, s) => sum + s.pagesPrinted, 0) || 0;
  const totalCost = usageStats?.reduce((sum, s) => sum + s.estimatedCost, 0) || 0;
  const successRate =
    totalJobs > 0
      ? ((usageStats?.reduce((sum, s) => sum + s.jobsCompleted, 0) || 0) / totalJobs) * 100
      : 0;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-900 dark:text-gray-100">Analytics</h1>
          <p className="text-gray-600 dark:text-gray-400 mt-1">
            Track your printing usage and environmental impact
          </p>
        </div>
        <div className="flex items-center gap-2">
          {(['7d', '30d', '90d', '12m'] as const).map((p) => (
            <button
              key={p}
              onClick={() => setPeriod(p)}
              className={`px-4 py-2 rounded-lg font-medium transition-colors ${
                period === p
                  ? 'bg-blue-600 text-white'
                  : 'bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-300 dark:hover:bg-gray-600'
              }`}
            >
              {p === '12m' ? '12 Months' : p === '7d' ? '7 Days' : p === '30d' ? '30 Days' : '90 Days'}
            </button>
          ))}
        </div>
      </div>

      {/* Key Metrics */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <MetricCard
          label="Total Jobs"
          value={totalJobs.toLocaleString()}
          change="+12.5%"
          changeType="positive"
          icon={
            <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
            </svg>
          }
          color="blue"
        />
        <MetricCard
          label="Pages Printed"
          value={totalPages.toLocaleString()}
          change="+8.2%"
          changeType="positive"
          icon={
            <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z" />
            </svg>
          }
          color="green"
        />
        <MetricCard
          label="Success Rate"
          value={`${successRate.toFixed(1)}%`}
          change="+2.1%"
          changeType="positive"
          icon={
            <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
          }
          color="purple"
        />
        <MetricCard
          label="Estimated Cost"
          value={`$${totalCost.toFixed(2)}`}
          change="-5.3%"
          changeType="negative"
          icon={
            <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
          }
          color="amber"
        />
      </div>

      {/* Environmental Report */}
      {environment && <EnvironmentReport report={environment} isLoading={statsLoading} />}

      {/* Charts */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Jobs Over Time */}
        <div className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">
            Print Volume Over Time
          </h3>
          <ResponsiveContainer width="100%" height={300}>
            <BarChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" className="stroke-gray-200 dark:stroke-gray-700" />
              <XAxis dataKey="date" className="text-gray-500 dark:text-gray-400" />
              <YAxis className="text-gray-500 dark:text-gray-400" />
              <Tooltip
                contentStyle={{
                  backgroundColor: 'rgb(31 41 55)',
                  border: 'none',
                  borderRadius: '0.5rem',
                }}
              />
              <Legend />
              <Bar dataKey="jobs" fill="#3b82f6" name="Jobs" radius={[4, 4, 0, 0]} />
              <Bar dataKey="pages" fill="#10b981" name="Pages" radius={[4, 4, 0, 0]} />
            </BarChart>
          </ResponsiveContainer>
        </div>

        {/* Status Distribution */}
        <div className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">
            Job Status Distribution
          </h3>
          <ResponsiveContainer width="100%" height={300}>
            <PieChart>
              <Pie
                data={statusData}
                cx="50%"
                cy="50%"
                labelLine={false}
                label={({ name, percent }) => `${name}: ${(percent * 100).toFixed(0)}%`}
                outerRadius={80}
                fill="#8884d8"
                dataKey="value"
              >
                {statusData.map((_entry, index) => (
                  <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                ))}
              </Pie>
              <Tooltip />
            </PieChart>
          </ResponsiveContainer>
        </div>
      </div>

      {/* CO2 Trend */}
      <div className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700">
        <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">
          CO₂ Emissions Trend
        </h3>
        <ResponsiveContainer width="100%" height={200}>
          <LineChart data={chartData}>
            <CartesianGrid strokeDasharray="3 3" className="stroke-gray-200 dark:stroke-gray-700" />
            <XAxis dataKey="date" className="text-gray-500 dark:text-gray-400" />
            <YAxis className="text-gray-500 dark:text-gray-400" />
            <Tooltip
              contentStyle={{
                backgroundColor: 'rgb(31 41 55)',
                border: 'none',
                borderRadius: '0.5rem',
              }}
            />
            <Line
              type="monotone"
              dataKey="co2"
              stroke="#10b981"
              strokeWidth={2}
              dot={{ fill: '#10b981' }}
              name="CO₂ (grams)"
            />
          </LineChart>
        </ResponsiveContainer>
      </div>

      {/* Audit Logs */}
      {user?.role === 'admin' || user?.role === 'owner' ? (
        <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700">
          <div className="p-6 border-b border-gray-200 dark:border-gray-700">
            <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
              Recent Activity
            </h3>
          </div>
          <div className="divide-y divide-gray-200 dark:divide-gray-700">
            {auditLogs?.data.slice(0, 10).map((log) => (
              <div key={log.id} className="p-4 hover:bg-gray-50 dark:hover:bg-gray-700/50">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium text-gray-900 dark:text-gray-100">
                      {log.action}
                    </p>
                    <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                      {log.resourceType && `${log.resourceType} • `}
                      {new Date(log.timestamp).toLocaleString()}
                    </p>
                  </div>
                  {log.ipAddress && (
                    <span className="text-xs text-gray-400">{log.ipAddress}</span>
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>
      ) : null}
    </div>
  );
};

interface MetricCardProps {
  label: string;
  value: string;
  change: string;
  changeType: 'positive' | 'negative';
  icon: React.ReactNode;
  color: 'blue' | 'green' | 'purple' | 'amber';
}

const MetricCard = ({ label, value, change, changeType, icon, color }: MetricCardProps) => {
  const colorClasses = {
    blue: 'bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400',
    green: 'bg-green-100 dark:bg-green-900/30 text-green-600 dark:text-green-400',
    purple: 'bg-purple-100 dark:bg-purple-900/30 text-purple-600 dark:text-purple-400',
    amber: 'bg-amber-100 dark:bg-amber-900/30 text-amber-600 dark:text-amber-400',
  };

  return (
    <div className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700">
      <div className="flex items-center justify-between">
        <div className={colorClasses[color]}>{icon}</div>
        <span
          className={`text-xs font-medium ${
            changeType === 'positive'
              ? 'text-green-600 dark:text-green-400'
              : 'text-red-600 dark:text-red-400'
          }`}
        >
          {change}
        </span>
      </div>
      <div className="mt-4">
        <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">{value}</p>
        <p className="text-sm text-gray-500 dark:text-gray-400">{label}</p>
      </div>
    </div>
  );
};

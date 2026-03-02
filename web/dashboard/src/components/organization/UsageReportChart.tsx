/**
 * UsageReportChart - Component displaying organization usage trends
 *
 * Features:
 * - Visual chart of usage over time
 * - Bar and line chart options
 * - Multiple metrics (jobs, pages, storage, users)
 * - Period selection (daily, weekly, monthly, yearly)
 */

import { useState } from 'react';
import { AreaChart, Area, BarChart, Bar, LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import type { UsageTrend, UsagePeriod } from '@/types';

interface UsageReportChartProps {
  data: UsageTrend[];
  period?: UsagePeriod;
  metric?: 'jobs' | 'pages' | 'storage' | 'users' | 'all';
  chartType?: 'area' | 'bar' | 'line';
  isLoading?: boolean;
  onPeriodChange?: (period: UsagePeriod) => void;
  className?: string;
  compact?: boolean;
}

// Icons
const TrendUpIcon = () => (
  <svg className="w-5 h-5 text-green-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 7h8m0 0v8m0-8l-8 8-4-4-6 6" />
  </svg>
);

const TrendDownIcon = () => (
  <svg className="w-5 h-5 text-red-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 17h8m0 0V9m0 8l-8-8-4 4-6-6" />
  </svg>
);

const CustomTooltip = ({ active, payload, label }: any) => {
  if (!active || !payload || !payload.length) return null;

  return (
    <div className="bg-white dark:bg-gray-800 p-3 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700">
      <p className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-2">
        {new Date(label).toLocaleDateString('en-US', { month: 'short', day: 'numeric' })}
      </p>
      {payload.map((entry: any, index: number) => (
        <div key={index} className="flex items-center gap-2 text-sm">
          <div
            className="w-3 h-3 rounded-full"
            style={{ backgroundColor: entry.color }}
          />
          <span className="text-gray-600 dark:text-gray-400">{entry.name}:</span>
          <span className="font-medium text-gray-900 dark:text-gray-100">
            {entry.value.toLocaleString()}
          </span>
        </div>
      ))}
    </div>
  );
};

const calculateTrend = (data: UsageTrend[], metric: 'jobs' | 'pages' | 'storage' | 'users'): {
  percentage: number;
  isPositive: boolean;
  current: number;
  previous: number;
} => {
  if (data.length < 2) {
    return { percentage: 0, isPositive: true, current: data[0]?.[metric] || 0, previous: 0 };
  }

  const current = data[data.length - 1][metric];
  const previous = data[data.length - 2][metric];

  if (previous === 0) {
    return { percentage: 0, isPositive: true, current, previous };
  }

  const percentage = ((current - previous) / previous) * 100;
  return {
    percentage: Math.abs(percentage),
    isPositive: percentage >= 0,
    current,
    previous,
  };
};

const MetricBadge = ({ label, value, color, trend }: { label: string; value: number; color: string; trend?: { percentage: number; isPositive: boolean } }) => (
  <div className={`p-3 rounded-lg bg-${color}-50 dark:bg-${color}-900/20 border border-${color}-200 dark:border-${color}-800`}>
    <p className="text-xs text-gray-500 dark:text-gray-400 uppercase tracking-wide">{label}</p>
    <div className="flex items-center gap-2 mt-1">
      <p className="text-lg font-semibold text-gray-900 dark:text-gray-100">{value.toLocaleString()}</p>
      {trend && (
        <div className={`flex items-center gap-1 text-xs ${trend.isPositive ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400'}`}>
          {trend.isPositive ? <TrendUpIcon /> : <TrendDownIcon />}
          {trend.percentage.toFixed(1)}%
        </div>
      )}
    </div>
  </div>
);

export const UsageReportChart = ({
  data,
  period = 'monthly',
  metric = 'all',
  chartType = 'area',
  isLoading = false,
  onPeriodChange,
  className = '',
  compact = false,
}: UsageReportChartProps) => {
  const [selectedPeriod, setSelectedPeriod] = useState<UsagePeriod>(period);
  const [selectedMetric, setSelectedMetric] = useState(metric);

  const handlePeriodChange = (newPeriod: UsagePeriod) => {
    setSelectedPeriod(newPeriod);
    onPeriodChange?.(newPeriod);
  };

  const formatChartData = () => {
    return data.map(item => ({
      ...item,
      date: new Date(item.date).toLocaleDateString('en-US', { month: 'short', day: 'numeric' }),
    }));
  };

  const chartData = formatChartData();

  const jobsTrend = calculateTrend(data, 'jobs');
  const pagesTrend = calculateTrend(data, 'pages');
  const storageTrend = calculateTrend(data, 'storage');
  const usersTrend = calculateTrend(data, 'users');

  const renderChart = () => {
    const colors = {
      jobs: '#3b82f6',
      pages: '#8b5cf6',
      storage: '#06b6d4',
      users: '#10b981',
    };

    const commonProps = {
      data: chartData,
      margin: { top: 10, right: 10, left: 0, bottom: 0 },
    };

    if (chartType === 'bar') {
      return (
        <ResponsiveContainer width="100%" height={compact ? 200 : 300}>
          <BarChart {...commonProps}>
            <CartesianGrid strokeDasharray="3 3" stroke="#374151" strokeOpacity={0.2} />
            <XAxis
              dataKey="date"
              tick={{ fill: '#9ca3af', fontSize: 12 }}
              stroke="#4b5563"
            />
            <YAxis
              tick={{ fill: '#9ca3af', fontSize: 12 }}
              stroke="#4b5563"
            />
            <Tooltip content={<CustomTooltip />} />
            {selectedMetric === 'all' || selectedMetric === 'jobs' ? (
              <Bar dataKey="jobs" name="Jobs" fill={colors.jobs} radius={[4, 4, 0, 0]} />
            ) : null}
            {selectedMetric === 'all' || selectedMetric === 'pages' ? (
              <Bar dataKey="pages" name="Pages" fill={colors.pages} radius={[4, 4, 0, 0]} />
            ) : null}
            {selectedMetric === 'all' || selectedMetric === 'storage' ? (
              <Bar dataKey="storage" name="Storage (GB)" fill={colors.storage} radius={[4, 4, 0, 0]} />
            ) : null}
            {selectedMetric === 'all' || selectedMetric === 'users' ? (
              <Bar dataKey="users" name="Users" fill={colors.users} radius={[4, 4, 0, 0]} />
            ) : null}
          </BarChart>
        </ResponsiveContainer>
      );
    }

    if (chartType === 'line') {
      return (
        <ResponsiveContainer width="100%" height={compact ? 200 : 300}>
          <LineChart {...commonProps}>
            <CartesianGrid strokeDasharray="3 3" stroke="#374151" strokeOpacity={0.2} />
            <XAxis
              dataKey="date"
              tick={{ fill: '#9ca3af', fontSize: 12 }}
              stroke="#4b5563"
            />
            <YAxis
              tick={{ fill: '#9ca3af', fontSize: 12 }}
              stroke="#4b5563"
            />
            <Tooltip content={<CustomTooltip />} />
            <Legend />
            {selectedMetric === 'all' || selectedMetric === 'jobs' ? (
              <Line
                type="monotone"
                dataKey="jobs"
                name="Jobs"
                stroke={colors.jobs}
                strokeWidth={2}
                dot={{ fill: colors.jobs, r: 4 }}
                activeDot={{ r: 6 }}
              />
            ) : null}
            {selectedMetric === 'all' || selectedMetric === 'pages' ? (
              <Line
                type="monotone"
                dataKey="pages"
                name="Pages"
                stroke={colors.pages}
                strokeWidth={2}
                dot={{ fill: colors.pages, r: 4 }}
                activeDot={{ r: 6 }}
              />
            ) : null}
            {selectedMetric === 'all' || selectedMetric === 'storage' ? (
              <Line
                type="monotone"
                dataKey="storage"
                name="Storage (GB)"
                stroke={colors.storage}
                strokeWidth={2}
                dot={{ fill: colors.storage, r: 4 }}
                activeDot={{ r: 6 }}
              />
            ) : null}
            {selectedMetric === 'all' || selectedMetric === 'users' ? (
              <Line
                type="monotone"
                dataKey="users"
                name="Users"
                stroke={colors.users}
                strokeWidth={2}
                dot={{ fill: colors.users, r: 4 }}
                activeDot={{ r: 6 }}
              />
            ) : null}
          </LineChart>
        </ResponsiveContainer>
      );
    }

    // Default: area chart
    return (
      <ResponsiveContainer width="100%" height={compact ? 200 : 300}>
        <AreaChart {...commonProps}>
          <defs>
            <linearGradient id="colorJobs" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor={colors.jobs} stopOpacity={0.3} />
              <stop offset="95%" stopColor={colors.jobs} stopOpacity={0} />
            </linearGradient>
            <linearGradient id="colorPages" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor={colors.pages} stopOpacity={0.3} />
              <stop offset="95%" stopColor={colors.pages} stopOpacity={0} />
            </linearGradient>
            <linearGradient id="colorStorage" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor={colors.storage} stopOpacity={0.3} />
              <stop offset="95%" stopColor={colors.storage} stopOpacity={0} />
            </linearGradient>
            <linearGradient id="colorUsers" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor={colors.users} stopOpacity={0.3} />
              <stop offset="95%" stopColor={colors.users} stopOpacity={0} />
            </linearGradient>
          </defs>
          <CartesianGrid strokeDasharray="3 3" stroke="#374151" strokeOpacity={0.2} />
          <XAxis
            dataKey="date"
            tick={{ fill: '#9ca3af', fontSize: 12 }}
            stroke="#4b5563"
          />
          <YAxis
            tick={{ fill: '#9ca3af', fontSize: 12 }}
            stroke="#4b5563"
          />
          <Tooltip content={<CustomTooltip />} />
          <Legend />
          {selectedMetric === 'all' || selectedMetric === 'jobs' ? (
            <Area
              type="monotone"
              dataKey="jobs"
              name="Jobs"
              stroke={colors.jobs}
              fill="url(#colorJobs)"
              strokeWidth={2}
            />
          ) : null}
          {selectedMetric === 'all' || selectedMetric === 'pages' ? (
            <Area
              type="monotone"
              dataKey="pages"
              name="Pages"
              stroke={colors.pages}
              fill="url(#colorPages)"
              strokeWidth={2}
            />
          ) : null}
          {selectedMetric === 'all' || selectedMetric === 'storage' ? (
            <Area
              type="monotone"
              dataKey="storage"
              name="Storage (GB)"
              stroke={colors.storage}
              fill="url(#colorStorage)"
              strokeWidth={2}
            />
          ) : null}
          {selectedMetric === 'all' || selectedMetric === 'users' ? (
            <Area
              type="monotone"
              dataKey="users"
              name="Users"
              stroke={colors.users}
              fill="url(#colorUsers)"
              strokeWidth={2}
            />
          ) : null}
        </AreaChart>
      </ResponsiveContainer>
    );
  };

  if (isLoading) {
    return (
      <div className={`bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 ${className}`}>
        <div className="p-6">
          <div className="animate-pulse space-y-4">
            <div className="h-6 bg-gray-200 dark:bg-gray-700 rounded w-1/3" />
            <div className="h-64 bg-gray-200 dark:bg-gray-700 rounded" />
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className={`bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 ${className}`}>
      {/* Header */}
      <div className="p-6 border-b border-gray-200 dark:border-gray-700">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
              Usage Report
            </h2>
            <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
              Track your organization's usage trends
            </p>
          </div>

          {/* Controls */}
          <div className="flex items-center gap-2">
            {/* Period Selector */}
            <select
              value={selectedPeriod}
              onChange={e => handlePeriodChange(e.target.value as UsagePeriod)}
              className="px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-gray-100"
            >
              <option value="daily">Daily</option>
              <option value="weekly">Weekly</option>
              <option value="monthly">Monthly</option>
              <option value="yearly">Yearly</option>
            </select>

            {/* Chart Type Selector */}
            <select
              value={chartType}
              onChange={e => setSelectedMetric(e.target.value as any)}
              className="px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-gray-100"
            >
              <option value="all">All Metrics</option>
              <option value="jobs">Jobs</option>
              <option value="pages">Pages</option>
              <option value="storage">Storage</option>
              <option value="users">Users</option>
            </select>
          </div>
        </div>
      </div>

      {/* Summary Cards */}
      {!compact && (
        <div className="p-6 grid grid-cols-2 md:grid-cols-4 gap-4">
          <MetricBadge
            label="Total Jobs"
            value={data.reduce((sum, d) => sum + d.jobs, 0)}
            color="blue"
            trend={jobsTrend}
          />
          <MetricBadge
            label="Total Pages"
            value={data.reduce((sum, d) => sum + d.pages, 0)}
            color="purple"
            trend={pagesTrend}
          />
          <MetricBadge
            label="Avg Storage (GB)"
            value={Math.round(data.reduce((sum, d) => sum + d.storage, 0) / data.length)}
            color="cyan"
            trend={storageTrend}
          />
          <MetricBadge
            label="Active Users"
            value={data[data.length - 1]?.users || 0}
            color="green"
            trend={usersTrend}
          />
        </div>
      )}

      {/* Chart */}
      <div className="p-6">
        {data.length === 0 ? (
          <div className="text-center py-12">
            <p className="text-gray-500 dark:text-gray-400">No usage data available</p>
          </div>
        ) : (
          renderChart()
        )}
      </div>
    </div>
  );
};

export default UsageReportChart;

/**
 * JobChart Component - Displays job statistics as a bar chart
 */

import type { JobStatistics } from './types';

export interface JobChartProps {
  statistics: JobStatistics;
  loading?: boolean;
  className?: string;
}

export const JobChart = ({
  statistics,
  loading = false,
  className = '',
}: JobChartProps) => {
  if (loading) {
    return (
      <div
        className={`
          bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700
          p-5 animate-pulse
          ${className}
        `}
      >
        <div className="h-6 bg-gray-200 dark:bg-gray-700 rounded w-1/3 mb-6" />
        <div className="h-48 bg-gray-200 dark:bg-gray-700 rounded" />
      </div>
    );
  }

  const maxValue = Math.max(...statistics.data.map((d) => d.value), 1);

  const getBarHeight = (value: number) => {
    return (value / maxValue) * 100;
  };

  const getPeriodLabel = () => {
    switch (statistics.period) {
      case 'day':
        return 'Last 24 Hours';
      case 'week':
        return 'Last 7 Days';
      case 'month':
        return 'Last 30 Days';
      default:
        return 'Job Statistics';
    }
  };

  return (
    <div
      className={`
        bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700
        ${className}
      `}
    >
      {/* Header */}
      <div className="p-5 border-b border-gray-200 dark:border-gray-700">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
            Job Statistics
          </h2>
          <span className="text-sm text-gray-500 dark:text-gray-400">
            {getPeriodLabel()}
          </span>
        </div>
        {/* Summary Stats */}
        <div className="mt-4 flex items-center gap-6">
          <div>
            <span className="text-2xl font-bold text-gray-900 dark:text-gray-100">
              {statistics.total.toLocaleString()}
            </span>
            <span className="text-sm text-gray-500 dark:text-gray-400 ml-1">
              total jobs
            </span>
          </div>
          <div className="flex gap-4">
            <div className="flex items-center gap-1.5">
              <span className="w-2 h-2 rounded-full bg-green-500" />
              <span className="text-sm text-gray-600 dark:text-gray-400">
                {statistics.completed} completed
              </span>
            </div>
            <div className="flex items-center gap-1.5">
              <span className="w-2 h-2 rounded-full bg-red-500" />
              <span className="text-sm text-gray-600 dark:text-gray-400">
                {statistics.failed} failed
              </span>
            </div>
          </div>
        </div>
      </div>

      {/* Chart */}
      <div className="p-5">
        {statistics.data.length === 0 ? (
          <div className="h-48 flex items-center justify-center text-gray-500 dark:text-gray-400">
            No data available
          </div>
        ) : (
          <div className="h-48">
            {/* Y-axis labels */}
            <div className="relative h-full">
              {/* Grid lines */}
              <div className="absolute inset-0 flex flex-col justify-between text-xs text-gray-400 dark:text-gray-600 pointer-events-none">
                <div className="border-b border-gray-200 dark:border-gray-700 pb-1">
                  {maxValue.toLocaleString()}
                </div>
                <div className="border-b border-gray-200 dark:border-gray-700 pb-1">
                  {Math.round(maxValue / 2).toLocaleString()}
                </div>
                <div>0</div>
              </div>

              {/* Bars */}
              <div className="ml-10 h-full flex items-end justify-between gap-1">
                {statistics.data.map((point, index) => {
                  const height = getBarHeight(point.value);
                  return (
                    <div
                      key={`${point.label}-${index}`}
                      className="flex-1 flex flex-col items-center group"
                    >
                      {/* Tooltip */}
                      <div className="opacity-0 group-hover:opacity-100 transition-opacity absolute -top-10 bg-gray-900 dark:bg-gray-700 text-white text-xs px-2 py-1 rounded pointer-events-none whitespace-nowrap z-10">
                        {point.label}: {point.value}
                      </div>

                      {/* Bar */}
                      <div
                        className={`
                          w-full max-w-8 bg-blue-500 dark:bg-blue-600 rounded-t
                          hover:bg-blue-600 dark:hover:bg-blue-500 transition-colors
                        `}
                        style={{ height: `${height}%` }}
                      />

                      {/* X-axis label */}
                      <span className="text-xs text-gray-500 dark:text-gray-400 mt-2 truncate w-full text-center">
                        {point.label}
                      </span>
                    </div>
                  );
                })}
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Footer with average */}
      <div className="px-5 pb-5">
        <div className="pt-4 border-t border-gray-200 dark:border-gray-700">
          <div className="flex items-center justify-between text-sm">
            <span className="text-gray-600 dark:text-gray-400">
              Average per {statistics.period === 'day' ? 'hour' : statistics.period === 'week' ? 'day' : 'day'}
            </span>
            <span className="font-semibold text-gray-900 dark:text-gray-100">
              {statistics.averagePerDay.toLocaleString()} jobs
            </span>
          </div>
        </div>
      </div>
    </div>
  );
};

export default JobChart;

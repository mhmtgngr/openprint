/**
 * StatCard Component for Compliance
 * Simple stat card component for compliance statistics
 */

import type { SVGProps } from 'react';

export interface StatCardProps {
  title: string;
  value: string | number;
  icon?: React.ComponentType<SVGProps<SVGSVGElement>>;
  color?: 'blue' | 'green' | 'purple' | 'orange' | 'red';
  trend?: {
    value: number;
    label: string;
    isPositive: boolean;
  };
  loading?: boolean;
  className?: string;
}

const colorConfig = {
  blue: {
    bg: 'bg-blue-100 dark:bg-blue-900/30',
    text: 'text-blue-600 dark:text-blue-400',
  },
  green: {
    bg: 'bg-green-100 dark:bg-green-900/30',
    text: 'text-green-600 dark:text-green-400',
  },
  purple: {
    bg: 'bg-purple-100 dark:bg-purple-900/30',
    text: 'text-purple-600 dark:text-purple-400',
  },
  orange: {
    bg: 'bg-orange-100 dark:bg-orange-900/30',
    text: 'text-orange-600 dark:text-orange-400',
  },
  red: {
    bg: 'bg-red-100 dark:bg-red-900/30',
    text: 'text-red-600 dark:text-red-400',
  },
};

export const StatCard = ({
  title,
  value,
  icon: Icon,
  color = 'blue',
  trend,
  loading = false,
  className = '',
}: StatCardProps) => {
  const config = colorConfig[color];

  if (loading) {
    return (
      <div
        className={`
          bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700
          p-5 animate-pulse
          ${className}
        `}
      >
        <div className="flex items-center justify-between">
          <div className="space-y-2 flex-1">
            <div className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-1/3" />
            <div className="h-8 bg-gray-200 dark:bg-gray-700 rounded w-1/2" />
          </div>
          <div className="h-12 w-12 bg-gray-200 dark:bg-gray-700 rounded-lg" />
        </div>
      </div>
    );
  }

  return (
    <div
      className={`
        bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700
        p-5 transition-all hover:shadow-md dark:hover:shadow-gray-900/50
        ${className}
      `}
      data-testid="stat-card"
    >
      <div className="flex items-start justify-between">
        {/* Value and Title */}
        <div className="flex-1">
          <p className="text-sm font-medium text-gray-600 dark:text-gray-400 mb-1">
            {title}
          </p>
          <h3 className="text-2xl font-bold text-gray-900 dark:text-gray-100">
            {typeof value === 'number' ? value.toLocaleString() : value}
          </h3>

          {/* Trend Indicator */}
          {trend && (
            <div className="mt-2 flex items-center gap-1 text-sm">
              <span
                className={`
                  flex items-center gap-0.5 font-medium
                  ${trend.isPositive
                    ? 'text-green-600 dark:text-green-400'
                    : 'text-red-600 dark:text-red-400'
                  }
                `}
              >
                {trend.isPositive ? (
                  <TrendUpIcon className="w-3 h-3" />
                ) : (
                  <TrendDownIcon className="w-3 h-3" />
                )}
                {Math.abs(trend.value)}%
              </span>
              <span className="text-gray-500 dark:text-gray-400">
                {trend.label}
              </span>
            </div>
          )}
        </div>

        {/* Icon */}
        {Icon && (
          <div
            className={`
              w-12 h-12 rounded-lg flex items-center justify-center
              ${config.bg}
            `}
          >
            <Icon className={`w-6 h-6 ${config.text}`} />
          </div>
        )}
      </div>
    </div>
  );
};

// Icons
const TrendUpIcon = ({ className }: { className?: string }) => (
  <svg
    className={className}
    fill="none"
    viewBox="0 0 24 24"
    stroke="currentColor"
  >
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M13 7h8m0 0v8m0-8l-8 8-4-4-6 6"
    />
  </svg>
);

const TrendDownIcon = ({ className }: { className?: string }) => (
  <svg
    className={className}
    fill="none"
    viewBox="0 0 24 24"
    stroke="currentColor"
  >
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M13 17h8m0 0V9m0 8l-8-8-4 4-6-6"
    />
  </svg>
);

export default StatCard;

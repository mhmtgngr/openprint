import { ReactNode } from 'react';

interface MetricCardProps {
  label: string;
  value: string | number;
  unit?: string;
  change?: string;
  changeType?: 'positive' | 'negative' | 'neutral';
  icon?: ReactNode;
  color?: 'blue' | 'green' | 'purple' | 'amber' | 'red' | 'cyan';
  loading?: boolean;
  sparkline?: number[];
  onClick?: () => void;
}

const colorClasses = {
  blue: {
    bg: 'bg-blue-100 dark:bg-blue-900/30',
    text: 'text-blue-600 dark:text-blue-400',
    border: 'border-blue-200 dark:border-blue-800',
  },
  green: {
    bg: 'bg-green-100 dark:bg-green-900/30',
    text: 'text-green-600 dark:text-green-400',
    border: 'border-green-200 dark:border-green-800',
  },
  purple: {
    bg: 'bg-purple-100 dark:bg-purple-900/30',
    text: 'text-purple-600 dark:text-purple-400',
    border: 'border-purple-200 dark:border-purple-800',
  },
  amber: {
    bg: 'bg-amber-100 dark:bg-amber-900/30',
    text: 'text-amber-600 dark:text-amber-400',
    border: 'border-amber-200 dark:border-amber-800',
  },
  red: {
    bg: 'bg-red-100 dark:bg-red-900/30',
    text: 'text-red-600 dark:text-red-400',
    border: 'border-red-200 dark:border-red-800',
  },
  cyan: {
    bg: 'bg-cyan-100 dark:bg-cyan-900/30',
    text: 'text-cyan-600 dark:text-cyan-400',
    border: 'border-cyan-200 dark:border-cyan-800',
  },
};

export const MetricCard = ({
  label,
  value,
  unit,
  change,
  changeType = 'neutral',
  icon,
  color = 'blue',
  loading = false,
  sparkline,
  onClick,
}: MetricCardProps) => {
  const colors = colorClasses[color];

  if (loading) {
    return (
      <div className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700 animate-pulse">
        <div className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-1/3 mb-4" />
        <div className="h-8 bg-gray-200 dark:bg-gray-700 rounded w-1/2 mb-2" />
        <div className="h-3 bg-gray-200 dark:bg-gray-700 rounded w-1/4" />
      </div>
    );
  }

  return (
    <div
      className={`bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border transition-all cursor-pointer hover:shadow-md ${
        onClick ? 'hover:border-blue-300 dark:hover:border-blue-700' : 'border-gray-200 dark:border-gray-700'
      }`}
      onClick={onClick}
      role={onClick ? 'button' : undefined}
      tabIndex={onClick ? 0 : undefined}
    >
      <div className="flex items-center justify-between mb-4">
        {icon && <div className={`${colors.bg} ${colors.text} p-2 rounded-lg`}>{icon}</div>}
        {change && (
          <span
            className={`text-xs font-medium ${
              changeType === 'positive'
                ? 'text-green-600 dark:text-green-400'
                : changeType === 'negative'
                  ? 'text-red-600 dark:text-red-400'
                  : 'text-gray-500 dark:text-gray-400'
            }`}
          >
            {changeType === 'positive' && '↑'}
            {changeType === 'negative' && '↓'}
            {change}
          </span>
        )}
      </div>

      <div className="space-y-1">
        <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">
          {value}
          {unit && <span className="text-sm font-normal text-gray-500 dark:text-gray-400 ml-1">{unit}</span>}
        </p>
        <p className="text-sm text-gray-500 dark:text-gray-400">{label}</p>
      </div>

      {sparkline && sparkline.length > 0 && (
        <div className="mt-4 h-10 flex items-end gap-0.5">
          {sparkline.map((point, i) => {
            const max = Math.max(...sparkline);
            const min = Math.min(...sparkline);
            const height = max > min ? ((point - min) / (max - min)) * 100 : 50;
            return (
              <div
                key={i}
                className={`${colors.bg} ${colors.text} flex-1 rounded-sm transition-all hover:opacity-80`}
                style={{ height: `${height}%` }}
                title={`${point.toFixed(2)}`}
              />
            );
          })}
        </div>
      )}
    </div>
  );
};

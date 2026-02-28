/**
 * RecentActivity Component - Displays recent job events and activities
 */

import type { Activity } from './types';
import { ACTIVITY_CONFIG } from './types';

export interface RecentActivityProps {
  activities: Activity[];
  loading?: boolean;
  maxItems?: number;
  onViewAll?: () => void;
  className?: string;
}

export const RecentActivity = ({
  activities,
  loading = false,
  maxItems = 5,
  onViewAll,
  className = '',
}: RecentActivityProps) => {
  const displayActivities = activities.slice(0, maxItems);

  const formatTimestamp = (timestamp: string) => {
    const date = new Date(timestamp);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);
    const diffDays = Math.floor(diffMs / 86400000);

    if (diffMins < 1) return 'Just now';
    if (diffMins < 60) return `${diffMins}m ago`;
    if (diffHours < 24) return `${diffHours}h ago`;
    if (diffDays < 7) return `${diffDays}d ago`;
    return date.toLocaleDateString();
  };

  if (loading) {
    return (
      <div
        className={`
          bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700
          ${className}
        `}
      >
        <div className="p-5 border-b border-gray-200 dark:border-gray-700">
          <div className="h-6 bg-gray-200 dark:bg-gray-700 rounded w-1/3 animate-pulse" />
        </div>
        <div className="p-5 space-y-4">
          {[...Array(maxItems)].map((_, i) => (
            <div key={i} className="flex items-start gap-3 animate-pulse">
              <div className="h-10 w-10 bg-gray-200 dark:bg-gray-700 rounded-full flex-shrink-0" />
              <div className="flex-1 space-y-2">
                <div className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-3/4" />
                <div className="h-3 bg-gray-200 dark:bg-gray-700 rounded w-1/2" />
              </div>
            </div>
          ))}
        </div>
      </div>
    );
  }

  return (
    <div
      className={`
        bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700
        ${className}
      `}
    >
      {/* Header */}
      <div className="flex items-center justify-between p-5 border-b border-gray-200 dark:border-gray-700">
        <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
          Recent Activity
        </h2>
        {onViewAll && activities.length > maxItems && (
          <button
            onClick={onViewAll}
            className="text-sm text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300 font-medium"
          >
            View all
          </button>
        )}
      </div>

      {/* Activity List */}
      {displayActivities.length === 0 ? (
        <div className="p-8 text-center">
          <EmptyStateIcon className="w-12 h-12 mx-auto text-gray-400 dark:text-gray-600 mb-3" />
          <p className="text-gray-500 dark:text-gray-400">No recent activity</p>
        </div>
      ) : (
        <ul className="divide-y divide-gray-200 dark:divide-gray-700">
          {displayActivities.map((activity) => {
            const config = ACTIVITY_CONFIG[activity.type];
            return (
              <li
                key={activity.id}
                className="p-4 hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors"
              >
                <div className="flex items-start gap-3">
                  {/* Icon */}
                  <div
                    className={`
                      w-10 h-10 rounded-full flex items-center justify-center flex-shrink-0
                      ${config.bgColor}
                    `}
                  >
                    <span className={`w-2 h-2 rounded-full ${config.dotColor}`} />
                  </div>

                  {/* Content */}
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium text-gray-900 dark:text-gray-100">
                      {activity.jobName || activity.printerName || activity.userName || 'Activity'}
                    </p>
                    <p className={`text-xs ${config.textColor} mt-0.5`}>
                      {config.label}
                      {activity.printerName && activity.type.startsWith('job_') && (
                        <span className="text-gray-600 dark:text-gray-400">
                          {' '}
                          on {activity.printerName}
                        </span>
                      )}
                    </p>
                    {activity.details && (
                      <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                        {activity.details}
                      </p>
                    )}
                  </div>

                  {/* Timestamp */}
                  <span className="text-xs text-gray-500 dark:text-gray-400 flex-shrink-0">
                    {formatTimestamp(activity.timestamp)}
                  </span>
                </div>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
};

// Icons
const EmptyStateIcon = ({ className }: { className?: string }) => (
  <svg
    className={className}
    fill="none"
    viewBox="0 0 24 24"
    stroke="currentColor"
  >
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={1.5}
      d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
    />
  </svg>
);

export default RecentActivity;

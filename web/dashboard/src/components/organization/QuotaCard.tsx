/**
 * QuotaCard - Component displaying organization resource quotas
 *
 * Shows current usage vs limits for:
 * - Users
 * - Printers
 * - Storage
 * - Monthly jobs
 */

import { useMemo } from 'react';
import type { ResourceQuota } from '@/types';

interface QuotaCardProps {
  quota: ResourceQuota | null;
  isLoading?: boolean;
  className?: string;
  compact?: boolean;
  showAll?: boolean;
}

interface QuotaItem {
  label: string;
  current: number;
  max: number;
  unit: string;
  color: string;
  icon: React.ReactNode;
  warningThreshold?: number;
}

// Icons
const UsersIcon = () => (
  <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />
  </svg>
);

const PrinterIcon = () => (
  <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2h2m2 4h6a2 2 0 002-2v-4a2 2 0 00-2-2H9a2 2 0 00-2 2v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z" />
  </svg>
);

const StorageIcon = () => (
  <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4" />
  </svg>
);

const JobsIcon = () => (
  <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
  </svg>
);

const WarningIcon = () => (
  <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
  </svg>
);

const CheckCircleIcon = () => (
  <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
  </svg>
);

const calculatePercentage = (current: number, max: number): number => {
  if (!max || max <= 0) return 0;
  return Math.min(100, Math.round((current / max) * 100));
};

const getColorForPercentage = (percentage: number): string => {
  if (percentage >= 90) return 'red';
  if (percentage >= 75) return 'orange';
  if (percentage >= 50) return 'yellow';
  return 'green';
};

const QuotaProgressBar = ({
  current,
  max,
  unit,
  warningThreshold = 80,
}: {
  current: number;
  max: number;
  unit: string;
  warningThreshold?: number;
}) => {
  const percentage = calculatePercentage(current, max);
  const actualColor = getColorForPercentage(percentage);
  const isNearLimit = percentage >= warningThreshold;

  const colorClasses = {
    green: 'bg-green-500 dark:bg-green-600',
    yellow: 'bg-yellow-500 dark:bg-yellow-600',
    orange: 'bg-orange-500 dark:bg-orange-600',
    red: 'bg-red-500 dark:bg-red-600',
  };

  return (
    <div className="flex-1">
      <div className="flex justify-between items-center mb-1">
        <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
          {current.toLocaleString()} / {max.toLocaleString()} {unit}
        </span>
        <div className="flex items-center gap-1">
          {isNearLimit ? (
            <span className={`text-xs font-medium text-${actualColor}-600 dark:text-${actualColor}-400 flex items-center gap-1`}>
              <WarningIcon />
              {percentage}%
            </span>
          ) : (
            <span className={`text-xs font-medium text-${actualColor}-600 dark:text-${actualColor}-400 flex items-center gap-1`}>
              <CheckCircleIcon />
              {percentage}%
            </span>
          )}
        </div>
      </div>
      <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
        <div
          className={`h-2 rounded-full transition-all duration-300 ${colorClasses[actualColor as keyof typeof colorClasses]}`}
          style={{ width: `${percentage}%` }}
        />
      </div>
    </div>
  );
};

export const QuotaCard = ({
  quota,
  isLoading = false,
  className = '',
  compact = false,
  showAll = true,
}: QuotaCardProps) => {
  const quotaItems = useMemo((): QuotaItem[] => {
    if (!quota) return [];

    const items: QuotaItem[] = [
      {
        label: 'Users',
        current: quota.currentUserCount,
        max: quota.maxUsers,
        unit: 'users',
        color: 'blue',
        icon: <UsersIcon />,
        warningThreshold: 85,
      },
      {
        label: 'Printers',
        current: quota.currentPrinterCount,
        max: quota.maxPrinters,
        unit: 'printers',
        color: 'purple',
        icon: <PrinterIcon />,
        warningThreshold: 85,
      },
    ];

    if (showAll) {
      items.push(
        {
          label: 'Storage',
          current: quota.currentStorageGB,
          max: quota.maxStorageGB,
          unit: 'GB',
          color: 'cyan',
          icon: <StorageIcon />,
          warningThreshold: 80,
        },
        {
          label: 'Monthly Jobs',
          current: quota.currentJobsThisMonth,
          max: quota.maxJobsPerMonth,
          unit: 'jobs',
          color: 'green',
          icon: <JobsIcon />,
          warningThreshold: 80,
        }
      );
    }

    return items;
  }, [quota, showAll]);

  const hasWarnings = useMemo(() => {
    return quotaItems.some(item => {
      const percentage = calculatePercentage(item.current, item.max);
      return percentage >= (item.warningThreshold || 80);
    });
  }, [quotaItems]);

  const resetDate = quota?.quotaResetDate
    ? new Date(quota.quotaResetDate).toLocaleDateString()
    : null;

  if (isLoading) {
    return (
      <div className={`bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 ${className}`}>
        <div className="p-6">
          <div className="animate-pulse space-y-4">
            <div className="h-6 bg-gray-200 dark:bg-gray-700 rounded w-1/3" />
            <div className="space-y-3">
              {[1, 2, 3, 4].map(i => (
                <div key={i} className="h-12 bg-gray-200 dark:bg-gray-700 rounded" />
              ))}
            </div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className={`bg-white dark:bg-gray-800 rounded-xl shadow-sm border ${hasWarnings ? 'border-orange-300 dark:border-orange-700' : 'border-gray-200 dark:border-gray-700'} ${className}`}>
      <div className="p-6 border-b border-gray-200 dark:border-gray-700">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className={`p-2 rounded-lg ${hasWarnings ? 'bg-orange-100 dark:bg-orange-900/30' : 'bg-blue-100 dark:bg-blue-900/30'}`}>
              {hasWarnings ? (
                <WarningIcon />
              ) : (
                <svg className="w-5 h-5 text-blue-600 dark:text-blue-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
                </svg>
              )}
            </div>
            <div>
              <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                Resource Quotas
              </h2>
              {resetDate && !compact && (
                <p className="text-xs text-gray-500 dark:text-gray-400">
                  Resets on {resetDate}
                </p>
              )}
            </div>
          </div>
          {hasWarnings && (
            <span className="px-3 py-1 bg-orange-100 dark:bg-orange-900/30 text-orange-700 dark:text-orange-300 text-xs font-medium rounded-full">
              At capacity
            </span>
          )}
        </div>
      </div>

      <div className={`p-6 ${compact ? 'py-4' : ''} space-y-4`}>
        {quotaItems.length === 0 ? (
          <p className="text-sm text-gray-500 dark:text-gray-400 text-center py-4">
            No quota information available
          </p>
        ) : (
          quotaItems.map(item => (
            <div key={item.label} className="flex items-center gap-3">
              <div className={`p-2 rounded-lg bg-${item.color}-100 dark:bg-${item.color}-900/30 text-${item.color}-600 dark:text-${item.color}-400 flex-shrink-0`}>
                {item.icon}
              </div>
              <QuotaProgressBar
                current={item.current}
                max={item.max}
                unit={item.unit}
                warningThreshold={item.warningThreshold}
              />
            </div>
          ))
        )}
      </div>

      {hasWarnings && !compact && (
        <div className="px-6 py-4 bg-orange-50 dark:bg-orange-900/20 border-t border-orange-200 dark:border-orange-800">
          <div className="flex items-start gap-3">
            <WarningIcon />
            <div>
              <p className="text-sm font-medium text-orange-800 dark:text-orange-300">
                Approaching resource limits
              </p>
              <p className="text-xs text-orange-700 dark:text-orange-400 mt-1">
                Consider upgrading your plan or reducing usage to avoid service interruption.
              </p>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default QuotaCard;

/**
 * ComplianceChecklist Component
 * Displays and runs compliance checklist items
 */

import type { ChecklistStatus } from '../types';

export interface ChecklistItem {
  name: string;
  status: ChecklistStatus;
  description?: string;
}

export interface ComplianceChecklistProps {
  checklist?: ChecklistItem[];
  isLoading?: boolean;
  onRun?: () => void;
  isRunning?: boolean;
}

export const ComplianceChecklist = ({
  checklist = [],
  isLoading = false,
  onRun,
  isRunning = false,
}: ComplianceChecklistProps) => {
  const statusConfig: Record<
    ChecklistStatus,
    { label: string; className: string; icon: React.ReactNode }
  > = {
    pass: {
      label: 'Pass',
      className: 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400',
      icon: (
        <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
          <path
            fillRule="evenodd"
            d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
            clipRule="evenodd"
          />
        </svg>
      ),
    },
    fail: {
      label: 'Fail',
      className: 'bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400',
      icon: (
        <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
          <path
            fillRule="evenodd"
            d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
            clipRule="evenodd"
          />
        </svg>
      ),
    },
    warning: {
      label: 'Warning',
      className: 'bg-amber-100 dark:bg-amber-900/30 text-amber-700 dark:text-amber-400',
      icon: (
        <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
          <path
            fillRule="evenodd"
            d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
            clipRule="evenodd"
          />
        </svg>
      ),
    },
    pending: {
      label: 'Pending',
      className: 'bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-400',
      icon: (
        <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
          <path
            fillRule="evenodd"
            d="M10 18a8 8 0 100-16 8 8 0 000 16zm1-12a1 1 0 10-2 0v4a1 1 0 00.293.707l2.828 2.829a1 1 0 101.415-1.415L11 9.586V6z"
            clipRule="evenodd"
          />
        </svg>
      ),
    },
  };

  const passedCount = checklist.filter((item) => item.status === 'pass').length;
  const failedCount = checklist.filter((item) => item.status === 'fail').length;
  const warningCount = checklist.filter((item) => item.status === 'warning').length;
  const pendingCount = checklist.filter((item) => item.status === 'pending').length;

  return (
    <div
      className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700"
      data-testid="compliance-checklist"
    >
      <div className="flex items-center justify-between mb-4">
        <div>
          <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
            Compliance Checklist
          </h2>
          {checklist.length > 0 && (
            <div className="flex items-center gap-3 mt-1 text-sm">
              <span className="text-green-600 dark:text-green-400">
                {passedCount} passed
              </span>
              <span className="text-red-600 dark:text-red-400">
                {failedCount} failed
              </span>
              <span className="text-amber-600 dark:text-amber-400">
                {warningCount} warnings
              </span>
              <span className="text-gray-600 dark:text-gray-400">
                {pendingCount} pending
              </span>
            </div>
          )}
        </div>
        {onRun && (
          <button
            onClick={onRun}
            disabled={isRunning}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-blue-400 text-white rounded-lg text-sm font-medium transition-colors flex items-center gap-2"
            data-testid="run-checklist-button"
          >
            {isRunning ? (
              <>
                <div className="animate-spin rounded-full h-4 w-4 border-2 border-white border-t-transparent" />
                Running...
              </>
            ) : (
              <>
                <PlayIcon className="w-4 h-4" />
                Run Checklist
              </>
            )}
          </button>
        )}
      </div>

      {isLoading ? (
        <div className="space-y-2">
          {[...Array(5)].map((_, i) => (
            <div
              key={i}
              className="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-700/50 rounded-lg animate-pulse"
            >
              <div className="h-5 bg-gray-200 dark:bg-gray-600 rounded w-1/3" />
              <div className="h-6 bg-gray-200 dark:bg-gray-600 rounded w-6" />
            </div>
          ))}
        </div>
      ) : checklist.length === 0 ? (
        <div
          className="text-center py-8"
          data-testid="checklist-empty"
        >
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
              d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4"
            />
          </svg>
          <p className="mt-2 text-sm text-gray-500 dark:text-gray-400">
            Click "Run Checklist" to verify compliance status
          </p>
        </div>
      ) : (
        <div className="space-y-2">
          {checklist.map((item, index) => (
            <div
              key={index}
              className="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-700/50 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
              data-testid="checklist-item"
            >
              <div className="flex-1">
                <span className="text-sm font-medium text-gray-900 dark:text-gray-100">
                  {item.name}
                </span>
                {item.description && (
                  <p className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">
                    {item.description}
                  </p>
                )}
              </div>
              <div
                className={`inline-flex items-center gap-2 px-3 py-1.5 rounded-lg ${
                  statusConfig[item.status]?.className || ''
                }`}
              >
                {statusConfig[item.status]?.icon}
                <span className="text-xs font-medium">
                  {statusConfig[item.status]?.label}
                </span>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

const PlayIcon = ({ className }: { className?: string }) => (
  <svg
    className={className}
    fill="currentColor"
    viewBox="0 0 20 20"
  >
    <path
      fillRule="evenodd"
      d="M10 18a8 8 0 100-16 8 8 0 000 16zM9.555 7.168A1 1 0 008 8v4a1 1 0 001.555.832l3-2a1 1 0 000-1.664l-3-2z"
      clipRule="evenodd"
    />
  </svg>
);

export default ComplianceChecklist;

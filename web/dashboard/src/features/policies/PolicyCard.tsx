import { type FC, memo } from 'react';
import type { PrintPolicy } from './types';

interface PolicyCardProps {
  policy: PrintPolicy;
  onEdit?: () => void;
  onDelete?: () => void;
  onToggle?: (enabled: boolean) => void;
  onDuplicate?: () => void;
  isDeleting?: boolean;
  isToggling?: boolean;
}

export const PolicyCard: FC<PolicyCardProps> = memo(({
  policy,
  onEdit,
  onDelete,
  onToggle,
  onDuplicate,
  isDeleting = false,
  isToggling = false,
}) => {
  const getPriorityColor = (priority: number) => {
    if (priority <= 2) return 'bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400 border-red-200 dark:border-red-800';
    if (priority <= 5) return 'bg-orange-100 dark:bg-orange-900/30 text-orange-700 dark:text-orange-400 border-orange-200 dark:border-orange-800';
    if (priority <= 8) return 'bg-amber-100 dark:bg-amber-900/30 text-amber-700 dark:text-amber-400 border-amber-200 dark:border-amber-800';
    return 'bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-400 border-gray-200 dark:border-gray-600';
  };

  const getPriorityLabel = (priority: number) => {
    if (priority <= 2) return 'Critical';
    if (priority <= 5) return 'High';
    if (priority <= 8) return 'Medium';
    return 'Low';
  };

  return (
    <div
      data-testid="policy-card"
      data-policy-id={policy.id}
      className={`bg-white dark:bg-gray-800 rounded-xl shadow-sm border transition-all ${
        policy.isEnabled
          ? 'border-gray-200 dark:border-gray-700'
          : 'border-gray-300 dark:border-gray-600 opacity-75'
      }`}
    >
      <div className="p-6">
        <div className="flex items-start justify-between">
          <div className="flex items-start gap-4">
            <div
              className={`p-3 rounded-lg ${
                policy.isEnabled
                  ? 'bg-blue-100 dark:bg-blue-900/30'
                  : 'bg-gray-100 dark:bg-gray-700'
              }`}
            >
              <svg
                className={`w-6 h-6 ${
                  policy.isEnabled
                    ? 'text-blue-600 dark:text-blue-400'
                    : 'text-gray-500'
                }`}
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"
                />
              </svg>
            </div>
            <div>
              <h3
                data-testid="policy-name"
                className="text-lg font-semibold text-gray-900 dark:text-gray-100"
              >
                {policy.name}
              </h3>
              {policy.description && (
                <p
                  data-testid="policy-description"
                  className="text-sm text-gray-500 dark:text-gray-400 mt-1"
                >
                  {policy.description}
                </p>
              )}
              <div className="flex items-center gap-3 mt-2">
                <span
                  data-testid="policy-priority"
                  className={`text-xs px-2 py-1 rounded-full border font-medium ${getPriorityColor(
                    policy.priority
                  )}`}
                >
                  {getPriorityLabel(policy.priority)} ({policy.priority})
                </span>
                <span
                  className={`text-xs px-2 py-1 rounded-full ${
                    policy.isEnabled
                      ? 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400'
                      : 'bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400'
                  }`}
                >
                  {policy.isEnabled ? 'Active' : 'Disabled'}
                </span>
              </div>
            </div>
          </div>
          <div className="flex items-center gap-2">
            {/* Toggle Switch */}
            <button
              data-testid="policy-toggle"
              onClick={() => onToggle?.(!policy.isEnabled)}
              disabled={isToggling}
              className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                policy.isEnabled ? 'bg-blue-600' : 'bg-gray-300 dark:bg-gray-600'
              }`}
              role="switch"
              aria-checked={policy.isEnabled}
              title={policy.isEnabled ? 'Disable' : 'Enable'}
            >
              <span
                className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                  policy.isEnabled ? 'translate-x-6' : 'translate-x-1'
                }`}
              />
            </button>

            {/* Edit Button */}
            {onEdit && (
              <button
                onClick={onEdit}
                className="p-2 text-gray-500 hover:text-blue-600 dark:hover:text-blue-400 transition-colors"
                title="Edit"
              >
                <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
                  />
                </svg>
              </button>
            )}

            {/* Duplicate Button */}
            {onDuplicate && (
              <button
                onClick={onDuplicate}
                className="p-2 text-gray-500 hover:text-purple-600 dark:hover:text-purple-400 transition-colors"
                title="Duplicate"
              >
                <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"
                  />
                </svg>
              </button>
            )}

            {/* Delete Button */}
            {onDelete && (
              <button
                onClick={onDelete}
                disabled={isDeleting}
                className="p-2 text-gray-500 hover:text-red-600 dark:hover:text-red-400 transition-colors disabled:opacity-50"
                title="Delete"
              >
                <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                  />
                </svg>
              </button>
            )}
          </div>
        </div>

        {/* Policy Details */}
        <div className="mt-4 grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
          <PolicyCondition
            label="Max Pages"
            value={
              policy.conditions.maxPagesPerJob
                ? `${policy.conditions.maxPagesPerJob} per job`
                : 'No limit'
            }
          />
          <PolicyCondition
            label="Duplex"
            value={policy.actions.forceDuplex ? 'Forced' : 'Optional'}
          />
          <PolicyCondition
            label="Color Mode"
            value={
              policy.actions.forceColor === true
                ? 'Color only'
                : policy.actions.forceColor === 'grayscale'
                ? 'Grayscale only'
                : 'Any'
            }
          />
          <PolicyCondition
            label="Approval"
            value={policy.actions.requireApproval ? 'Required' : 'Not required'}
          />
        </div>
      </div>
    </div>
  );
});

PolicyCard.displayName = 'PolicyCard';

interface PolicyConditionProps {
  label: string;
  value: string;
}

const PolicyCondition: FC<PolicyConditionProps> = ({ label, value }) => (
  <div>
    <span className="text-gray-500 dark:text-gray-400">{label}:</span>
    <span className="ml-2 text-gray-900 dark:text-gray-100 font-medium">{value}</span>
  </div>
);

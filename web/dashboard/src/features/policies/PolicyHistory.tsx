import { type FC } from 'react';
import { usePolicyHistory, useRestorePolicy } from './usePolicyEvaluation';

interface PolicyHistoryProps {
  policyId: string;
  onClose?: () => void;
}

export const PolicyHistory: FC<PolicyHistoryProps> = ({ policyId, onClose }) => {
  const { data: history, isLoading } = usePolicyHistory(policyId);
  const restoreMutation = useRestorePolicy();

  const handleRestore = async (versionId: string) => {
    if (confirm('Are you sure you want to restore this version? Current settings will be replaced.')) {
      try {
        await restoreMutation.mutateAsync({ policyId, versionId });
        onClose?.();
      } catch (error) {
        console.error('Restore failed:', error);
      }
    }
  };

  if (isLoading) {
    return (
      <div className="text-center py-8">
        <div className="inline-block w-8 h-8 border-4 border-blue-600 border-t-transparent rounded-full animate-spin" />
      </div>
    );
  }

  if (!history || history.length === 0) {
    return (
      <div className="text-center py-8 text-gray-500 dark:text-gray-400">
        <svg className="w-12 h-12 mx-auto mb-3 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
        <p>No history available for this policy</p>
      </div>
    );
  }

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString();
  };

  return (
    <div className="space-y-4" data-testid="policy-history">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
          Version History
        </h3>
        <button
          onClick={onClose}
          className="text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
        >
          <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <div className="space-y-3">
        {history.map((entry, index) => (
          <div
            key={entry.id}
            data-version-id={entry.id}
            className={`p-4 rounded-lg border ${
              index === 0
                ? 'bg-blue-50 dark:bg-blue-900/20 border-blue-200 dark:border-blue-800'
                : 'bg-gray-50 dark:bg-gray-700/50 border-gray-200 dark:border-gray-600'
            }`}
          >
            <div className="flex items-start justify-between">
              <div className="flex-1">
                <div className="flex items-center gap-2 mb-1">
                  <span className="text-sm font-medium text-gray-900 dark:text-gray-100">
                    Version {entry.version}
                  </span>
                  {index === 0 && (
                    <span className="text-xs bg-blue-600 text-white px-2 py-0.5 rounded-full">
                      Current
                    </span>
                  )}
                </div>
                <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">
                  {entry.changes}
                </p>
                <div className="flex items-center gap-4 text-xs text-gray-500 dark:text-gray-400">
                  <span>By {entry.changedBy}</span>
                  <span>{formatDate(entry.changedAt)}</span>
                </div>
              </div>
              {index !== 0 && (
                <button
                  onClick={() => handleRestore(entry.id)}
                  data-testid="restore-policy-button"
                  disabled={restoreMutation.isPending}
                  className="px-3 py-1 text-sm bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50"
                >
                  Restore
                </button>
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};

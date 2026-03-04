/**
 * PolicyFormModal - Modal for creating/editing rate limit policies
 */

import type { RateLimitPolicy } from '@/types/ratelimit';

export interface PolicyFormModalProps {
  policy?: RateLimitPolicy;
  onClose: () => void;
  onSave: () => void;
}

export const PolicyFormModal = ({ policy, onClose, onSave }: PolicyFormModalProps) => {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="bg-white dark:bg-gray-800 rounded-lg p-6 w-full max-w-lg">
        <h2 className="text-xl font-bold mb-4">
          {policy ? 'Edit Policy' : 'Create New Policy'}
        </h2>
        <p className="text-gray-600 dark:text-gray-400 mb-4">
          Policy form placeholder - implement form fields here
        </p>
        <div className="flex justify-end gap-2">
          <button
            onClick={onClose}
            className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg"
          >
            Cancel
          </button>
          <button
            onClick={onSave}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg"
          >
            Save
          </button>
        </div>
      </div>
    </div>
  );
};

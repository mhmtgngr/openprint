/**
 * TestPolicyModal - Modal for testing rate limit policies
 */

import type { RateLimitPolicy } from '@/types/ratelimit';

export interface TestPolicyModalProps {
  policy: RateLimitPolicy;
  onClose: () => void;
}

export const TestPolicyModal = ({ policy, onClose }: TestPolicyModalProps) => {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="bg-white dark:bg-gray-800 rounded-lg p-6 w-full max-w-lg">
        <h2 className="text-xl font-bold mb-4">Test Policy: {policy.name}</h2>
        <p className="text-gray-600 dark:text-gray-400 mb-4">
          Policy testing form placeholder - implement test fields here
        </p>
        <div className="flex justify-end">
          <button
            onClick={onClose}
            className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
};

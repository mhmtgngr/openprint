/**
 * PolicyViolationsPanel - Panel for viewing policy violations
 */

export interface PolicyViolationsPanelProps {
  policyId: string;
  onClose: () => void;
}

export const PolicyViolationsPanel = ({ policyId, onClose }: PolicyViolationsPanelProps) => {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="bg-white dark:bg-gray-800 rounded-lg p-6 w-full max-w-2xl max-h-[80vh] overflow-auto">
        <div className="flex justify-between items-center mb-4">
          <h2 className="text-xl font-bold">Policy Violations</h2>
          <button
            onClick={onClose}
            className="text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
          >
            ✕
          </button>
        </div>
        <p className="text-gray-600 dark:text-gray-400 mb-4">
          Violations panel for policy: {policyId}
        </p>
        <p className="text-gray-600 dark:text-gray-400">
          Violations data placeholder - implement violations list here
        </p>
      </div>
    </div>
  );
};

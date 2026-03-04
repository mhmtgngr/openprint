import { type FC, useState, useEffect } from 'react';
import { useCreatePolicy, useUpdatePolicy } from './usePolicies';
import type { PrintPolicy, CreatePolicyRequest } from './types';

interface PolicyFormProps {
  policy?: PrintPolicy | null;
  onClose: () => void;
  onSave?: () => void;
  mode?: 'create' | 'edit';
}

export const PolicyForm: FC<PolicyFormProps> = ({
  policy,
  onClose,
  onSave,
  mode = policy ? 'edit' : 'create',
}) => {
  const createMutation = useCreatePolicy();
  const updateMutation = useUpdatePolicy();

  const [name, setName] = useState(policy?.name || '');
  const [description, setDescription] = useState(policy?.description || '');
  const [maxPages, setMaxPages] = useState(policy?.conditions.maxPagesPerJob || 0);
  const [maxPagesPerMonth, setMaxPagesPerMonth] = useState(policy?.conditions.maxPagesPerMonth || 0);
  const [forceDuplex, setForceDuplex] = useState(policy?.actions.forceDuplex || false);
  const [forceGrayscale, setForceGrayscale] = useState(policy?.actions.forceColor === 'grayscale');
  const [requireApproval, setRequireApproval] = useState(
    policy?.actions.requireApproval || policy?.conditions.requireApproval || false
  );
  const [maxCopies, setMaxCopies] = useState(policy?.actions.maxCopies || 1);
  const [priority, setPriority] = useState(policy?.priority || 10);
  const [isEnabled, setIsEnabled] = useState(policy?.isEnabled ?? true);
  const [appliesTo, setAppliesTo] = useState(policy?.appliesTo || 'all');

  const [errors, setErrors] = useState<Record<string, string>>({});

  useEffect(() => {
    if (policy) {
      setName(policy.name);
      setDescription(policy.description || '');
      setMaxPages(policy.conditions.maxPagesPerJob || 0);
      setMaxPagesPerMonth(policy.conditions.maxPagesPerMonth || 0);
      setForceDuplex(policy.actions.forceDuplex || false);
      setForceGrayscale(policy.actions.forceColor === 'grayscale');
      setRequireApproval(
        policy.actions.requireApproval || policy.conditions.requireApproval || false
      );
      setMaxCopies(policy.actions.maxCopies || 1);
      setPriority(policy.priority);
      setIsEnabled(policy.isEnabled);
      setAppliesTo(policy.appliesTo || 'all');
    }
  }, [policy]);

  const validate = (): boolean => {
    const newErrors: Record<string, string> = {};

    if (!name.trim()) {
      newErrors.name = 'Policy name is required';
    }

    if (priority < 1 || priority > 100) {
      newErrors.priority = 'Priority must be between 1 and 100';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!validate()) {
      return;
    }

    const data: CreatePolicyRequest = {
      name: name.trim(),
      description: description.trim(),
      conditions: {
        maxPagesPerJob: maxPages > 0 ? maxPages : undefined,
        maxPagesPerMonth: maxPagesPerMonth > 0 ? maxPagesPerMonth : undefined,
        requireApproval,
      },
      actions: {
        forceDuplex,
        forceColor: forceGrayscale ? 'grayscale' : undefined,
        maxCopies,
      },
      appliesTo: appliesTo as 'all' | 'users' | 'groups' | 'printers',
    };

    try {
      if (mode === 'create') {
        await createMutation.mutateAsync(data);
      } else if (policy) {
        await updateMutation.mutateAsync({ id: policy.id, data });
      }
      onSave?.();
      onClose();
    } catch (error) {
      // Error handling is done by the mutation hooks
    }
  };

  const isLoading = createMutation.isPending || updateMutation.isPending;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 overflow-y-auto">
      <div
        data-testid="policy-editor"
        className="bg-white dark:bg-gray-800 rounded-xl shadow-xl max-w-2xl w-full mx-4 my-8"
      >
        <div className="p-6 border-b border-gray-200 dark:border-gray-700">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
            {mode === 'create' ? 'Create Policy' : 'Edit Policy'}
          </h3>
        </div>

        <form onSubmit={handleSubmit} className="p-6 space-y-6 max-h-[70vh] overflow-y-auto">
          {/* Basic Information */}
          <div className="grid grid-cols-2 gap-4">
            <div className="col-span-2">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Policy Name *
              </label>
              <input
                data-testid="policy-name-input"
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                required
                className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
              {errors.name && (
                <p className="text-red-500 text-sm mt-1">{errors.name}</p>
              )}
            </div>

            <div className="col-span-2">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Description
              </label>
              <textarea
                data-testid="policy-description-input"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                rows={2}
                className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Priority (1-100) *
              </label>
              <input
                data-testid="policy-priority-input"
                type="number"
                value={priority}
                onChange={(e) => setPriority(parseInt(e.target.value) || 1)}
                min={1}
                max={100}
                required
                className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
              {errors.priority && (
                <p className="text-red-500 text-sm mt-1">{errors.priority}</p>
              )}
              <p className="text-xs text-gray-500 mt-1">Lower numbers = higher priority</p>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Scope
              </label>
              <select
                data-testid="policy-scope-select"
                value={appliesTo}
                onChange={(e) => setAppliesTo(e.target.value as 'all' | 'users' | 'groups' | 'printers')}
                className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              >
                <option value="all">All Users</option>
                <option value="users">Specific Users</option>
                <option value="groups">Specific Groups</option>
                <option value="printers">Specific Printers</option>
              </select>
            </div>

            <div className="col-span-2 flex items-center gap-3">
              <input
                data-testid="policy-enabled-toggle"
                type="checkbox"
                id="policy-enabled"
                checked={isEnabled}
                onChange={(e) => setIsEnabled(e.target.checked)}
                className="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
              />
              <label
                htmlFor="policy-enabled"
                className="text-sm text-gray-700 dark:text-gray-300"
              >
                Enable this policy
              </label>
            </div>
          </div>

          {/* Conditions Section */}
          <div className="border-t border-gray-200 dark:border-gray-700 pt-4">
            <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-4">
              Conditions
            </h4>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Max Pages per Job
                </label>
                <input
                  data-testid="max-pages-input"
                  type="number"
                  value={maxPages || ''}
                  onChange={(e) => setMaxPages(parseInt(e.target.value) || 0)}
                  min={0}
                  className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  placeholder="0 = unlimited"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Max Pages per Month
                </label>
                <input
                  type="number"
                  value={maxPagesPerMonth || ''}
                  onChange={(e) => setMaxPagesPerMonth(parseInt(e.target.value) || 0)}
                  min={0}
                  className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  placeholder="0 = unlimited"
                />
              </div>
            </div>
          </div>

          {/* Actions Section */}
          <div className="border-t border-gray-200 dark:border-gray-700 pt-4">
            <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-4">
              Actions
            </h4>
            <div className="space-y-3">
              <label className="flex items-center gap-3">
                <input
                  data-testid="force-duplex-checkbox"
                  type="checkbox"
                  checked={forceDuplex}
                  onChange={(e) => setForceDuplex(e.target.checked)}
                  className="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
                />
                <span className="text-sm text-gray-700 dark:text-gray-300">
                  Force duplex (double-sided) printing
                </span>
              </label>
              <label className="flex items-center gap-3">
                <input
                  data-testid="force-grayscale-checkbox"
                  type="checkbox"
                  checked={forceGrayscale}
                  onChange={(e) => setForceGrayscale(e.target.checked)}
                  className="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
                />
                <span className="text-sm text-gray-700 dark:text-gray-300">
                  Force grayscale (black & white only)
                </span>
              </label>
              <label className="flex items-center gap-3">
                <input
                  data-testid="require-approval-checkbox"
                  type="checkbox"
                  checked={requireApproval}
                  onChange={(e) => setRequireApproval(e.target.checked)}
                  className="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
                />
                <span className="text-sm text-gray-700 dark:text-gray-300">
                  Require manual approval for print jobs
                </span>
              </label>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Max Copies
                </label>
                <input
                  data-testid="max-copies-input"
                  type="number"
                  value={maxCopies}
                  onChange={(e) => setMaxCopies(parseInt(e.target.value) || 1)}
                  min={1}
                  className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
              </div>
            </div>
          </div>

          {/* Form Actions */}
          <div className="flex gap-3 pt-4 border-t border-gray-200 dark:border-gray-700">
            <button
              type="button"
              data-testid="cancel-policy-button"
              onClick={onClose}
              className="flex-1 px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
            >
              Cancel
            </button>
            <button
              type="submit"
              data-testid="save-policy-button"
              disabled={isLoading}
              className="flex-1 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {isLoading ? 'Saving...' : mode === 'create' ? 'Create' : 'Update'} Policy
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

export default PolicyForm;

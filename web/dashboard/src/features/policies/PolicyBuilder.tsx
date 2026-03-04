import { type FC, useState } from 'react';
import { useCreatePolicy } from './usePolicies';
import type {
  PolicyCondition,
  ConditionType,
  ConditionOperator,
  PolicyAction,
  ActionType,
} from './types';

interface PolicyBuilderProps {
  templateId?: string;
  onClose?: () => void;
  onSave?: () => void;
}

export const PolicyBuilder: FC<PolicyBuilderProps> = ({
  // templateId is not currently used but kept for future implementation
  onClose,
  onSave,
}) => {
  const createMutation = useCreatePolicy();

  const [policyName, setPolicyName] = useState('');
  const [policyDescription, setPolicyDescription] = useState('');
  const [priority, setPriority] = useState(10);
  const [conditions, setConditions] = useState<PolicyCondition[]>([]);
  const [actions, setActions] = useState<PolicyAction[]>([]);

  const conditionTypes: { value: ConditionType; label: string }[] = [
    { value: 'userRole', label: 'User Role' },
    { value: 'group', label: 'Group' },
    { value: 'printer', label: 'Printer' },
    { value: 'time', label: 'Time Range' },
    { value: 'documentType', label: 'Document Type' },
    { value: 'pageCount', label: 'Page Count' },
    { value: 'fileSize', label: 'File Size' },
    { value: 'colorMode', label: 'Color Mode' },
    { value: 'always', label: 'Always' },
  ];

  const getOperatorsForType = (type: ConditionType): ConditionOperator[] => {
    switch (type) {
      case 'pageCount':
      case 'fileSize':
        return ['equals', 'notEquals', 'greaterThan', 'lessThan', 'between'];
      case 'userRole':
      case 'group':
      case 'printer':
      case 'documentType':
        return ['equals', 'notEquals', 'in', 'notIn'];
      case 'time':
        return ['equals', 'between'];
      case 'colorMode':
        return ['equals', 'notEquals'];
      case 'always':
        return ['equals'];
      default:
        return ['equals'];
    }
  };

  const actionTypes: { value: ActionType; label: string; hasParameter?: boolean }[] = [
    { value: 'allow', label: 'Allow' },
    { value: 'deny', label: 'Deny' },
    { value: 'forceDuplex', label: 'Force Duplex' },
    { value: 'forceGrayscale', label: 'Force Grayscale' },
    { value: 'requireApproval', label: 'Require Approval' },
    { value: 'setCopiesLimit', label: 'Set Copies Limit', hasParameter: true },
    { value: 'routeToPrinter', label: 'Route to Printer', hasParameter: true },
    { value: 'notify', label: 'Send Notification', hasParameter: true },
    { value: 'blockJob', label: 'Block Job' },
  ];

  const addCondition = () => {
    const newCondition: PolicyCondition = {
      id: `cond-${Date.now()}`,
      type: 'pageCount',
      operator: 'greaterThan',
      value: '',
    };
    setConditions([...conditions, newCondition]);
  };

  const updateCondition = (id: string, updates: Partial<PolicyCondition>) => {
    setConditions(conditions.map((c) => (c.id === id ? { ...c, ...updates } : c)));
  };

  const removeCondition = (id: string) => {
    setConditions(conditions.filter((c) => c.id !== id));
  };

  const addAction = () => {
    const newAction: PolicyAction = {
      id: `action-${Date.now()}`,
      type: 'allow',
    };
    setActions([...actions, newAction]);
  };

  const updateAction = (id: string, updates: Partial<PolicyAction>) => {
    setActions(actions.map((a) => (a.id === id ? { ...a, ...updates } : a)));
  };

  const removeAction = (id: string) => {
    setActions(actions.filter((a) => a.id !== id));
  };

  const handleSubmit = async () => {
    if (!policyName.trim()) {
      alert('Please enter a policy name');
      return;
    }

    if (conditions.length === 0) {
      alert('Please add at least one condition');
      return;
    }

    if (actions.length === 0) {
      alert('Please add at least one action');
      return;
    }

    // Convert to API format
    const data = {
      name: policyName.trim(),
      description: policyDescription.trim(),
      priority,
      conditions: {
        // Map conditions to the API format
        maxPagesPerJob: conditions.find((c) => c.type === 'pageCount')
          ? parseInt(conditions.find((c) => c.type === 'pageCount')!.value) || undefined
          : undefined,
        requireApproval: actions.some((a) => a.type === 'requireApproval'),
      },
      actions: {
        forceDuplex: actions.some((a) => a.type === 'forceDuplex'),
        forceColor: actions.some((a) => a.type === 'forceGrayscale') ? 'grayscale' : undefined,
        maxCopies: actions.find((a) => a.type === 'setCopiesLimit')
          ? parseInt(actions.find((a) => a.type === 'setCopiesLimit')!.value as string) || undefined
          : undefined,
      },
      appliesTo: 'all' as const,
    };

    try {
      await createMutation.mutateAsync(data as any);
      onSave?.();
      onClose?.();
    } catch (error) {
      console.error('Failed to create policy:', error);
    }
  };

  return (
    <div className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700">
      <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-6">
        Policy Builder
      </h2>

      <div className="space-y-6">
        {/* Basic Info */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Policy Name
            </label>
            <input
              type="text"
              value={policyName}
              onChange={(e) => setPolicyName(e.target.value)}
              placeholder="Enter policy name"
              className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Priority
            </label>
            <select
              value={priority}
              onChange={(e) => setPriority(parseInt(e.target.value))}
              className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
            >
              <option value={1}>1 - Critical</option>
              <option value={5}>5 - High</option>
              <option value={10}>10 - Medium</option>
              <option value={50}>50 - Low</option>
            </select>
          </div>
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            Description
          </label>
          <textarea
            value={policyDescription}
            onChange={(e) => setPolicyDescription(e.target.value)}
            rows={2}
            placeholder="Describe what this policy does"
            className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
        </div>

        {/* Conditions Builder */}
        <div data-testid="conditions-section">
          <div className="flex items-center justify-between mb-3">
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
              Conditions
            </label>
            <button
              type="button"
              onClick={addCondition}
              className="text-sm text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300"
            >
              + Add Condition
            </button>
          </div>
          <div className="p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg space-y-3">
            {conditions.length === 0 ? (
              <p className="text-sm text-gray-500 dark:text-gray-400 text-center py-4">
                Add conditions to define when this policy should apply
              </p>
            ) : (
              conditions.map((condition) => (
                <div
                  key={condition.id}
                  data-testid="condition-row"
                  className="flex items-center gap-3 p-3 bg-white dark:bg-gray-800 rounded-lg"
                >
                  <select
                    value={condition.type}
                    onChange={(e) =>
                      updateCondition(condition.id, {
                        type: e.target.value as ConditionType,
                        operator: getOperatorsForType(e.target.value as ConditionType)[0],
                      })
                    }
                    className="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 text-sm"
                  >
                    {conditionTypes.map((type) => (
                      <option key={type.value} value={type.value}>
                        {type.label}
                      </option>
                    ))}
                  </select>
                  <select
                    value={condition.operator}
                    onChange={(e) =>
                      updateCondition(condition.id, { operator: e.target.value as ConditionOperator })
                    }
                    className="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 text-sm"
                  >
                    {getOperatorsForType(condition.type).map((op) => (
                      <option key={op} value={op}>
                        {op}
                      </option>
                    ))}
                  </select>
                  <input
                    type="text"
                    value={condition.value}
                    onChange={(e) => updateCondition(condition.id, { value: e.target.value })}
                    placeholder="Value"
                    className="flex-1 px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 text-sm"
                  />
                  <button
                    type="button"
                    onClick={() => removeCondition(condition.id)}
                    data-testid="remove-condition"
                    className="p-2 text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-lg"
                  >
                    <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                    </svg>
                  </button>
                </div>
              ))
            )}
          </div>
        </div>

        {/* Actions Builder */}
        <div data-testid="actions-section">
          <div className="flex items-center justify-between mb-3">
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
              Actions
            </label>
            <button
              type="button"
              onClick={addAction}
              className="text-sm text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300"
            >
              + Add Action
            </button>
          </div>
          <div className="p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg space-y-3">
            {actions.length === 0 ? (
              <p className="text-sm text-gray-500 dark:text-gray-400 text-center py-4">
                Define what actions to take when conditions are met
              </p>
            ) : (
              actions.map((action) => (
                <div
                  key={action.id}
                  data-testid="action-row"
                  className="flex items-center gap-3 p-3 bg-white dark:bg-gray-800 rounded-lg"
                >
                  <select
                    value={action.type}
                    onChange={(e) =>
                      updateAction(action.id, { type: e.target.value as ActionType })
                    }
                    className="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 text-sm"
                  >
                    {actionTypes.map((type) => (
                      <option key={type.value} value={type.value}>
                        {type.label}
                      </option>
                    ))}
                  </select>
                  {actionTypes.find((t) => t.value === action.type)?.hasParameter && (
                    <input
                      type="text"
                      value={action.parameter || action.value?.toString() || ''}
                      onChange={(e) => updateAction(action.id, { parameter: e.target.value })}
                      placeholder="Parameter"
                      className="flex-1 px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 text-sm"
                    />
                  )}
                  <button
                    type="button"
                    onClick={() => removeAction(action.id)}
                    data-testid="remove-action"
                    className="p-2 text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-lg"
                  >
                    <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                    </svg>
                  </button>
                </div>
              ))
            )}
          </div>
        </div>

        {/* Actions */}
        <div className="flex items-center gap-3 pt-4 border-t border-gray-200 dark:border-gray-700">
          <button
            type="button"
            onClick={handleSubmit}
            disabled={createMutation.isPending}
            className="px-6 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-medium transition-colors disabled:opacity-50"
          >
            {createMutation.isPending ? 'Creating...' : 'Create Policy'}
          </button>
          <button
            type="button"
            onClick={onClose}
            className="px-6 py-2 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700 rounded-lg font-medium transition-colors"
          >
            Cancel
          </button>
        </div>
      </div>
    </div>
  );
};

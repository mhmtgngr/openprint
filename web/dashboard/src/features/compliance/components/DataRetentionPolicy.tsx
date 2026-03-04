/**
 * DataRetentionPolicy Component
 * Manages data retention policy settings
 */

import { useState } from 'react';
import type { DataRetentionPolicy as RetentionPolicyType } from '../types';

export interface DataRetentionPolicyProps {
  policy?: RetentionPolicyType;
  isLoading?: boolean;
  onUpdate?: (policy: Partial<RetentionPolicyType>) => void | Promise<void>;
}

export const DataRetentionPolicy = ({
  policy,
  isLoading = false,
  onUpdate,
}: DataRetentionPolicyProps) => {
  const [isEnabled, setIsEnabled] = useState(policy?.enabled ?? false);
  const [periodDays, setPeriodDays] = useState(policy?.period_days ?? 90);
  const [periodUnit, setPeriodUnit] = useState<'days' | 'months' | 'years'>(
    policy?.period_unit ?? 'days'
  );
  const [isSaving, setIsSaving] = useState(false);
  const [saveMessage, setSaveMessage] = useState<string | null>(null);

  const handleSave = async () => {
    if (!onUpdate) return;
    setIsSaving(true);
    setSaveMessage(null);

    try {
      await onUpdate({
        enabled: isEnabled,
        period_days: periodDays,
        period_unit: periodUnit,
      });
      setSaveMessage('Data retention policy saved successfully');
      setTimeout(() => setSaveMessage(null), 3000);
    } catch (error) {
      setSaveMessage('Failed to save data retention policy');
      setTimeout(() => setSaveMessage(null), 3000);
    } finally {
      setIsSaving(false);
    }
  };

  const calculateRetentionDate = () => {
    const date = new Date();
    switch (periodUnit) {
      case 'days':
        date.setDate(date.getDate() - periodDays);
        break;
      case 'months':
        date.setMonth(date.getMonth() - periodDays);
        break;
      case 'years':
        date.setFullYear(date.getFullYear() - periodDays);
        break;
    }
    return date;
  };

  return (
    <div
      className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700"
      data-testid="data-retention-section"
    >
      <div className="mb-4">
        <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
          Data Retention Policy
        </h2>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          Configure automatic deletion of old audit logs and compliance data
        </p>
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center py-8">
          <div className="animate-spin rounded-full h-8 w-8 border-2 border-blue-600 border-t-transparent" />
        </div>
      ) : (
        <div className="space-y-4">
          {/* Enable/Disable Toggle */}
          <div
            className="flex items-center justify-between p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg"
            data-testid="retention-toggle"
          >
            <div>
              <p className="font-medium text-gray-900 dark:text-gray-100">
                Enable Automatic Retention
              </p>
              <p className="text-sm text-gray-600 dark:text-gray-400">
                Automatically delete old audit logs after specified period
              </p>
            </div>
            <button
              onClick={() => setIsEnabled(!isEnabled)}
              className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                isEnabled ? 'bg-blue-600' : 'bg-gray-300 dark:bg-gray-600'
              }`}
              role="switch"
              aria-checked={isEnabled}
              data-testid="retention-enabled-switch"
            >
              <span
                className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                  isEnabled ? 'translate-x-6' : 'translate-x-1'
                }`}
              />
            </button>
          </div>

          {/* Retention Period Settings */}
          {isEnabled && (
            <div
              className="p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg space-y-4"
              data-testid="retention-period-settings"
            >
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  Retention Period
                </label>
                <div className="flex gap-2">
                  <input
                    type="number"
                    value={periodDays}
                    onChange={(e) => setPeriodDays(parseInt(e.target.value) || 0)}
                    min={1}
                    className="flex-1 px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
                    data-testid="retention-period-input"
                  />
                  <select
                    value={periodUnit}
                    onChange={(e) =>
                      setPeriodUnit(e.target.value as 'days' | 'months' | 'years')
                    }
                    className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
                    data-testid="retention-period-unit"
                  >
                    <option value="days">Days</option>
                    <option value="months">Months</option>
                    <option value="years">Years</option>
                  </select>
                </div>
              </div>

              {/* Retention Summary */}
              <div
                className="p-3 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg"
                data-testid="retention-summary"
              >
                <div className="flex items-start gap-2">
                  <svg
                    className="w-5 h-5 text-blue-600 dark:text-blue-400 flex-shrink-0 mt-0.5"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                    />
                  </svg>
                  <div className="text-sm">
                    <p className="font-medium text-blue-800 dark:text-blue-400">
                      Audit logs older than{' '}
                      <span className="font-semibold">
                        {periodDays} {periodUnit}
                      </span>{' '}
                      will be automatically deleted.
                    </p>
                    <p className="text-blue-700 dark:text-blue-300 mt-1">
                      Currently, logs from before{' '}
                      {calculateRetentionDate().toLocaleDateString()} would be
                      deleted.
                    </p>
                  </div>
                </div>
              </div>

              {/* Warning */}
              <div
                className="p-3 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg"
              >
                <div className="flex items-start gap-2">
                  <svg
                    className="w-5 h-5 text-amber-600 dark:text-amber-400 flex-shrink-0 mt-0.5"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
                    />
                  </svg>
                  <p className="text-sm text-amber-800 dark:text-amber-400">
                    <span className="font-medium">Warning:</span> This action
                    cannot be undone. Deleted audit logs will be permanently
                    removed.
                  </p>
                </div>
              </div>
            </div>
          )}

          {/* Save Button */}
          <div className="flex items-center justify-between">
            <div>
              {saveMessage && (
                <p
                  className={`text-sm ${
                    saveMessage.includes('Failed')
                      ? 'text-red-600 dark:text-red-400'
                      : 'text-green-600 dark:text-green-400'
                  }`}
                  data-testid="retention-save-message"
                >
                  {saveMessage}
                </p>
              )}
            </div>
            <button
              onClick={handleSave}
              disabled={isSaving || isLoading}
              className="px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-blue-400 text-white rounded-lg font-medium transition-colors"
              data-testid="save-retention-button"
            >
              {isSaving ? 'Saving...' : 'Save Policy'}
            </button>
          </div>
        </div>
      )}
    </div>
  );
};

export default DataRetentionPolicy;

/**
 * SecuritySettings Component
 * Manages security settings including encryption, 2FA, session timeout, and IP whitelist
 */

import { useState } from 'react';
import type { SecuritySettings as SecuritySettingsType, IPWhitelistEntry } from '../types';

export interface SecuritySettingsProps {
  settings?: SecuritySettingsType;
  isLoading?: boolean;
  onUpdate?: (settings: Partial<SecuritySettingsType>) => void | Promise<void>;
  onAddIPWhitelist?: (entry: Omit<IPWhitelistEntry, 'id'>) => void | Promise<void>;
  onRemoveIPWhitelist?: (ip: string) => void | Promise<void>;
}

export const SecuritySettings = ({
  settings,
  isLoading = false,
  onUpdate,
  onAddIPWhitelist,
  onRemoveIPWhitelist,
}: SecuritySettingsProps) => {
  const [encryptionEnabled, setEncryptionEnabled] = useState(
    settings?.encryption_enabled ?? false
  );
  const [encryptionAlgorithm, setEncryptionAlgorithm] = useState(
    settings?.encryption_algorithm ?? 'AES-256'
  );
  const [twoFactorEnabled, setTwoFactorEnabled] = useState(
    settings?.two_factor_enabled ?? false
  );
  const [sessionTimeout, setSessionTimeout] = useState(
    settings?.session_timeout_minutes ?? 30
  );
  const [newIP, setNewIP] = useState('');
  const [newIPDescription, setNewIPDescription] = useState('');
  const [isSaving, setIsSaving] = useState(false);
  const [saveMessage, setSaveMessage] = useState<string | null>(null);
  const [ipWhitelist, setIPWhitelist] = useState<IPWhitelistEntry[]>(
    settings?.ip_whitelist ?? []
  );

  const handleSaveSecuritySettings = async () => {
    if (!onUpdate) return;
    setIsSaving(true);
    setSaveMessage(null);

    try {
      await onUpdate({
        encryption_enabled: encryptionEnabled,
        encryption_algorithm: encryptionAlgorithm,
        two_factor_enabled: twoFactorEnabled,
        session_timeout_minutes: sessionTimeout,
        ip_whitelist: ipWhitelist,
      });
      setSaveMessage('Security settings saved successfully');
      setTimeout(() => setSaveMessage(null), 3000);
    } catch (error) {
      setSaveMessage('Failed to save security settings');
      setTimeout(() => setSaveMessage(null), 3000);
    } finally {
      setIsSaving(false);
    }
  };

  const handleAddIP = async () => {
    if (!newIP.trim()) return;

    const entry: Omit<IPWhitelistEntry, 'id'> = {
      ip: newIP.trim(),
      description: newIPDescription.trim(),
    };

    if (onAddIPWhitelist) {
      try {
        await onAddIPWhitelist(entry);
        setIPWhitelist([...ipWhitelist, { ...entry, id: Date.now().toString() }]);
        setNewIP('');
        setNewIPDescription('');
        setSaveMessage('IP added to whitelist');
        setTimeout(() => setSaveMessage(null), 3000);
      } catch (error) {
        setSaveMessage('Failed to add IP to whitelist');
        setTimeout(() => setSaveMessage(null), 3000);
      }
    } else {
      setIPWhitelist([...ipWhitelist, { ...entry, id: Date.now().toString() }]);
      setNewIP('');
      setNewIPDescription('');
    }
  };

  const handleRemoveIP = async (ip: string) => {
    if (onRemoveIPWhitelist) {
      try {
        await onRemoveIPWhitelist(ip);
        setIPWhitelist(ipWhitelist.filter((entry) => entry.ip !== ip));
        setSaveMessage('IP removed from whitelist');
        setTimeout(() => setSaveMessage(null), 3000);
      } catch (error) {
        setSaveMessage('Failed to remove IP from whitelist');
        setTimeout(() => setSaveMessage(null), 3000);
      }
    } else {
      setIPWhitelist(ipWhitelist.filter((entry) => entry.ip !== ip));
    }
  };

  return (
    <div
      className="bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700"
      data-testid="security-settings-section"
    >
      <div className="mb-4">
        <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
          Security Settings
        </h2>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          Configure encryption, authentication, and access controls
        </p>
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center py-8">
          <div className="animate-spin rounded-full h-8 w-8 border-2 border-blue-600 border-t-transparent" />
        </div>
      ) : (
        <div className="space-y-6">
          {/* Data Encryption */}
          <div
            className="flex items-center justify-between p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg"
            data-testid="encryption-setting"
          >
            <div>
              <p className="font-medium text-gray-900 dark:text-gray-100">
                Data Encryption
              </p>
              <p className="text-sm text-gray-600 dark:text-gray-400">
                Encrypt sensitive data at rest
              </p>
            </div>
            <button
              onClick={() => setEncryptionEnabled(!encryptionEnabled)}
              className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                encryptionEnabled ? 'bg-blue-600' : 'bg-gray-300 dark:bg-gray-600'
              }`}
              role="switch"
              aria-checked={encryptionEnabled}
              data-testid="encryption-toggle"
            >
              <span
                className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                  encryptionEnabled ? 'translate-x-6' : 'translate-x-1'
                }`}
              />
            </button>
          </div>

          {encryptionEnabled && (
            <div className="ml-4 p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                Encryption Algorithm
              </label>
              <select
                value={encryptionAlgorithm}
                onChange={(e) =>
                  setEncryptionAlgorithm(
                    e.target.value as 'AES-256' | 'AES-128' | 'ChaCha20'
                  )
                }
                className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
                data-testid="encryption-algorithm"
              >
                <option>AES-256</option>
                <option>AES-128</option>
                <option>ChaCha20</option>
              </select>
            </div>
          )}

          {/* Two-Factor Authentication */}
          <div
            className="flex items-center justify-between p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg"
            data-testid="2fa-setting"
          >
            <div>
              <p className="font-medium text-gray-900 dark:text-gray-100">
                Two-Factor Authentication (2FA)
              </p>
              <p className="text-sm text-gray-600 dark:text-gray-400">
                Require 2FA for all admin users
              </p>
            </div>
            <button
              onClick={() => setTwoFactorEnabled(!twoFactorEnabled)}
              className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                twoFactorEnabled ? 'bg-blue-600' : 'bg-gray-300 dark:bg-gray-600'
              }`}
              role="switch"
              aria-checked={twoFactorEnabled}
              data-testid="2fa-toggle"
            >
              <span
                className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                  twoFactorEnabled ? 'translate-x-6' : 'translate-x-1'
                }`}
              />
            </button>
          </div>

          {/* Session Timeout */}
          <div
            className="p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg"
            data-testid="session-timeout-setting"
          >
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Session Timeout (minutes)
            </label>
            <input
              type="number"
              value={sessionTimeout}
              onChange={(e) => setSessionTimeout(parseInt(e.target.value) || 30)}
              min={5}
              max={480}
              className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
              data-testid="session-timeout-input"
            />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              Users will be logged out after {sessionTimeout} minutes of
              inactivity
            </p>
          </div>

          {/* IP Whitelist */}
          <div
            className="p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg"
            data-testid="ip-whitelist"
          >
            <p className="font-medium text-gray-900 dark:text-gray-100 mb-3">
              IP Whitelist
            </p>
            <p className="text-sm text-gray-600 dark:text-gray-400 mb-3">
              Restrict admin access to specific IP addresses
            </p>

            <div className="flex gap-2 mb-4">
              <input
                type="text"
                value={newIP}
                onChange={(e) => setNewIP(e.target.value)}
                placeholder="192.168.1.100"
                className="flex-1 px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
                data-testid="whitelist-ip-input"
              />
              <input
                type="text"
                value={newIPDescription}
                onChange={(e) => setNewIPDescription(e.target.value)}
                placeholder="Description (optional)"
                className="flex-1 px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
                data-testid="whitelist-description-input"
              />
              <button
                onClick={handleAddIP}
                disabled={!newIP.trim()}
                className="px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-400 text-white rounded-lg font-medium transition-colors"
                data-testid="add-whitelist-ip-button"
              >
                Add
              </button>
            </div>

            <div className="space-y-2">
              {ipWhitelist.length === 0 ? (
                <p className="text-sm text-gray-500 dark:text-gray-400 italic">
                  No IP addresses whitelisted
                </p>
              ) : (
                ipWhitelist.map((entry, index) => (
                  <div
                    key={entry.id || index}
                    className="flex items-center justify-between p-3 bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700"
                    data-testid="whitelist-item"
                  >
                    <div>
                      <p className="font-medium text-gray-900 dark:text-gray-100 font-mono">
                        {entry.ip}
                      </p>
                      {entry.description && (
                        <p className="text-sm text-gray-500 dark:text-gray-400">
                          {entry.description}
                        </p>
                      )}
                    </div>
                    <button
                      onClick={() => handleRemoveIP(entry.ip)}
                      className="p-2 text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-lg transition-colors"
                      data-testid={`remove-whitelist-${entry.ip}`}
                    >
                      <TrashIcon className="w-5 h-5" />
                    </button>
                  </div>
                ))
              )}
            </div>
          </div>

          {/* Save Button */}
          <div className="flex items-center justify-between pt-4 border-t border-gray-200 dark:border-gray-700">
            <div>
              {saveMessage && (
                <p
                  className={`text-sm ${
                    saveMessage.includes('Failed')
                      ? 'text-red-600 dark:text-red-400'
                      : 'text-green-600 dark:text-green-400'
                  }`}
                  data-testid="security-save-message"
                >
                  {saveMessage}
                </p>
              )}
            </div>
            <button
              onClick={handleSaveSecuritySettings}
              disabled={isSaving || isLoading}
              className="px-6 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-blue-400 text-white rounded-lg font-medium transition-colors"
              data-testid="save-security-button"
            >
              {isSaving ? 'Saving...' : 'Save Settings'}
            </button>
          </div>
        </div>
      )}
    </div>
  );
};

const TrashIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
    />
  </svg>
);

export default SecuritySettings;

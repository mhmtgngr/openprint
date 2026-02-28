import { useState } from 'react';
import { ProfileSettings } from './ProfileSettings';
import { SecuritySettings } from './SecuritySettings';
import { OrganizationSettings } from './OrganizationSettings';
import { Toast } from './Toast';
import type { SettingsTab } from './types';

interface SettingsProps {
  className?: string;
  defaultTab?: SettingsTab['value'];
}

const tabs: SettingsTab[] = [
  { value: 'profile', label: 'Profile' },
  { value: 'security', label: 'Security' },
  { value: 'organization', label: 'Organization' },
];

export const Settings = ({ className = '', defaultTab = 'profile' }: SettingsProps) => {
  const [activeTab, setActiveTab] = useState<SettingsTab['value']>(defaultTab);

  return (
    <div className={`space-y-6 ${className}`}>
      {/* Toast Notifications */}
      <Toast />

      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold text-gray-900 dark:text-gray-100">Settings</h1>
        <p className="text-gray-600 dark:text-gray-400 mt-1">
          Manage your account and organization settings
        </p>
      </div>

      {/* Tabs */}
      <div className="border-b border-gray-200 dark:border-gray-700">
        <nav className="flex gap-8 overflow-x-auto">
          {tabs.map((tab) => (
            <button
              key={tab.value}
              onClick={() => setActiveTab(tab.value)}
              className={`
                py-4 px-1 border-b-2 font-medium text-sm transition-colors whitespace-nowrap
                ${activeTab === tab.value
                  ? 'border-blue-500 text-blue-600 dark:text-blue-400'
                  : 'border-transparent text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300'
                }
              `}
              aria-current={activeTab === tab.value ? 'page' : undefined}
            >
              {tab.label}
            </button>
          ))}
        </nav>
      </div>

      {/* Tab Content */}
      <div className="min-h-[400px]">
        {activeTab === 'profile' && <ProfileSettings />}
        {activeTab === 'security' && <SecuritySettings />}
        {activeTab === 'organization' && <OrganizationSettings />}
      </div>
    </div>
  );
};

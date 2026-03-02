/**
 * OrganizationForm - Form for creating and editing organizations
 *
 * Features:
 * - Create new organization
 * - Edit existing organization details
 * - Configure quotas and limits
 * - Plan selection
 * - Organization settings
 */

import { useState, useEffect, FormEvent } from 'react';
import type {
  Organization,
  CreateOrganizationRequest,
  UpdateOrganizationRequest,
  OrganizationPlan,
} from '@/types';

interface OrganizationFormProps {
  mode: 'create' | 'edit';
  organization?: Organization;
  onSubmit: (data: CreateOrganizationRequest | UpdateOrganizationRequest) => Promise<void>;
  onCancel: () => void;
  isLoading?: boolean;
  className?: string;
}

const planOptions: Array<{ value: OrganizationPlan; label: string; description: string }> = [
  {
    value: 'free',
    label: 'Free',
    description: 'Up to 5 users, 2 printers, 10GB storage',
  },
  {
    value: 'pro',
    label: 'Pro',
    description: 'Up to 50 users, 20 printers, 100GB storage',
  },
  {
    value: 'enterprise',
    label: 'Enterprise',
    description: 'Unlimited users, printers, and storage',
  },
];

interface FormData {
  name: string;
  slug: string;
  displayName: string;
  plan: OrganizationPlan;
  ownerId: string;
  maxUsers: number;
  maxPrinters: number;
  maxStorageGB: number;
  maxJobsPerMonth: number;
  branding: {
    logoUrl: string;
    primaryColor: string;
    customDomain: string;
  };
  security: {
    requireMFA: boolean;
    passwordMinLength: number;
    sessionTimeoutMinutes: number;
  };
}

const generateSlug = (name: string): string => {
  return name
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '');
};

export const OrganizationForm = ({
  mode,
  organization,
  onSubmit,
  onCancel,
  isLoading = false,
  className = '',
}: OrganizationFormProps) => {
  const [formData, setFormData] = useState<FormData>({
    name: '',
    slug: '',
    displayName: '',
    plan: 'free',
    ownerId: '',
    maxUsers: 5,
    maxPrinters: 2,
    maxStorageGB: 10,
    maxJobsPerMonth: 1000,
    branding: {
      logoUrl: '',
      primaryColor: '#3b82f6',
      customDomain: '',
    },
    security: {
      requireMFA: false,
      passwordMinLength: 8,
      sessionTimeoutMinutes: 60,
    },
  });

  const [errors, setErrors] = useState<Partial<Record<keyof FormData, string>>>({});
  const [showAdvanced, setShowAdvanced] = useState(false);

  // Initialize form with organization data if editing
  useEffect(() => {
    if (organization && mode === 'edit') {
      const branding = (organization.settings as any)?.branding || {};
      const security = (organization.settings as any)?.security || {};
      setFormData({
        name: organization.name,
        slug: organization.slug,
        displayName: (organization as any).displayName || '',
        plan: organization.plan,
        ownerId: '',
        maxUsers: (organization as any).quotas?.maxUsers || 5,
        maxPrinters: (organization as any).quotas?.maxPrinters || 2,
        maxStorageGB: (organization as any).quotas?.maxStorageGB || 10,
        maxJobsPerMonth: (organization as any).quotas?.maxJobsPerMonth || 1000,
        branding: {
          logoUrl: branding.logoUrl || '',
          primaryColor: branding.primaryColor || '#3b82f6',
          customDomain: branding.customDomain || '',
        },
        security: {
          requireMFA: security.requireMFA || false,
          passwordMinLength: security.passwordMinLength || 8,
          sessionTimeoutMinutes: security.sessionTimeoutMinutes || 60,
        },
      });
    }
  }, [organization, mode]);

  // Auto-generate slug from name
  useEffect(() => {
    if (mode === 'create' && formData.name && !formData.slug) {
      setFormData(prev => ({ ...prev, slug: generateSlug(formData.name) }));
    }
  }, [formData.name, mode, formData.slug]);

  const updateField = <K extends keyof FormData>(field: K, value: FormData[K]) => {
    setFormData(prev => ({ ...prev, [field]: value }));
    // Clear error for this field
    if (errors[field]) {
      setErrors(prev => ({ ...prev, [field]: undefined }));
    }
  };

  const updateNestedField = <K extends keyof FormData>(
    parent: K,
    field: string,
    value: unknown
  ) => {
    setFormData(prev => ({
      ...prev,
      [parent]: { ...(prev[parent] as Record<string, unknown>), [field]: value },
    }));
  };

  const validate = (): boolean => {
    const newErrors: Partial<Record<keyof FormData, string>> = {};

    if (!formData.name.trim()) {
      newErrors.name = 'Organization name is required';
    }
    if (!formData.slug.trim()) {
      newErrors.slug = 'Slug is required';
    } else if (!/^[a-z0-9-]+$/.test(formData.slug)) {
      newErrors.slug = 'Slug can only contain lowercase letters, numbers, and hyphens';
    }
    if (mode === 'create' && !formData.ownerId.trim()) {
      newErrors.ownerId = 'Owner is required';
    }
    if (formData.maxUsers <= 0) {
      newErrors.maxUsers = 'Must be greater than 0';
    }
    if (formData.maxPrinters <= 0) {
      newErrors.maxPrinters = 'Must be greater than 0';
    }
    if (formData.maxStorageGB <= 0) {
      newErrors.maxStorageGB = 'Must be greater than 0';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();

    if (!validate()) return;

    if (mode === 'create') {
      const createData: CreateOrganizationRequest = {
        name: formData.name,
        slug: formData.slug,
        displayName: formData.displayName || undefined,
        plan: formData.plan,
        ownerId: formData.ownerId,
        quotas: {
          maxUsers: formData.maxUsers,
          maxPrinters: formData.maxPrinters,
          maxStorageGB: formData.maxStorageGB,
          maxJobsPerMonth: formData.maxJobsPerMonth,
        },
        settings: {
          branding: formData.branding,
          security: formData.security,
        },
      };
      await onSubmit(createData);
    } else {
      const updateData: UpdateOrganizationRequest = {
        name: formData.name,
        displayName: formData.displayName || undefined,
        plan: formData.plan,
        settings: {
          branding: formData.branding,
          security: formData.security,
        },
      };
      await onSubmit(updateData);
    }
  };

  const getDefaultQuotas = (plan: OrganizationPlan) => {
    switch (plan) {
      case 'free':
        return { maxUsers: 5, maxPrinters: 2, maxStorageGB: 10, maxJobsPerMonth: 1000 };
      case 'pro':
        return { maxUsers: 50, maxPrinters: 20, maxStorageGB: 100, maxJobsPerMonth: 10000 };
      case 'enterprise':
        return { maxUsers: -1, maxPrinters: -1, maxStorageGB: -1, maxJobsPerMonth: -1 };
    }
  };

  // Update quotas when plan changes
  const handlePlanChange = (plan: OrganizationPlan) => {
    const defaults = getDefaultQuotas(plan);
    setFormData(prev => ({
      ...prev,
      plan,
      ...defaults,
    }));
  };

  return (
    <form onSubmit={handleSubmit} className={`space-y-6 ${className}`}>
      {/* Basic Information */}
      <div className="space-y-4">
        <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100">
          Basic Information
        </h3>

        {/* Organization Name */}
        <div>
          <label
            htmlFor="name"
            className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
          >
            Organization Name <span className="text-red-500">*</span>
          </label>
          <input
            id="name"
            type="text"
            value={formData.name}
            onChange={e => updateField('name', e.target.value)}
            placeholder="Acme Corporation"
            className={`w-full px-4 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-gray-100 ${
              errors.name ? 'border-red-500' : 'border-gray-300 dark:border-gray-600'
            }`}
            disabled={isLoading}
          />
          {errors.name && <p className="text-sm text-red-500 mt-1">{errors.name}</p>}
        </div>

        {/* Slug */}
        <div>
          <label
            htmlFor="slug"
            className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
          >
            Slug <span className="text-red-500">*</span>
          </label>
          <div className="flex items-center gap-2">
            <span className="text-gray-500 dark:text-gray-400 text-sm">openprint.cloud/</span>
            <input
              id="slug"
              type="text"
              value={formData.slug}
              onChange={e => updateField('slug', e.target.value)}
              placeholder="acme-corporation"
              className={`flex-1 px-4 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-gray-100 ${
                errors.slug ? 'border-red-500' : 'border-gray-300 dark:border-gray-600'
              }`}
              disabled={isLoading || mode === 'edit'}
            />
          </div>
          {errors.slug && <p className="text-sm text-red-500 mt-1">{errors.slug}</p>}
          <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
            Used for URLs and organization identification
          </p>
        </div>

        {/* Display Name */}
        <div>
          <label
            htmlFor="displayName"
            className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
          >
            Display Name
          </label>
          <input
            id="displayName"
            type="text"
            value={formData.displayName}
            onChange={e => updateField('displayName', e.target.value)}
            placeholder="Acme Corp"
            className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-gray-100"
            disabled={isLoading}
          />
          <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
            Optional friendly name for display purposes
          </p>
        </div>
      </div>

      {/* Plan Selection */}
      <div className="space-y-4">
        <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100">
          Plan Selection
        </h3>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {planOptions.map(plan => (
            <label
              key={plan.value}
              className={`relative flex flex-col p-4 border-2 rounded-lg cursor-pointer transition-all ${
                formData.plan === plan.value
                  ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
                  : 'border-gray-200 dark:border-gray-700 hover:border-gray-300 dark:hover:border-gray-600'
              }`}
            >
              <input
                type="radio"
                name="plan"
                value={plan.value}
                checked={formData.plan === plan.value}
                onChange={e => handlePlanChange(e.target.value as OrganizationPlan)}
                className="sr-only"
                disabled={isLoading}
              />
              <div className="flex items-center justify-between mb-2">
                <span className="font-semibold text-gray-900 dark:text-gray-100">{plan.label}</span>
                {formData.plan === plan.value && (
                  <svg className="w-5 h-5 text-blue-500" fill="currentColor" viewBox="0 0 20 20">
                    <path
                      fillRule="evenodd"
                      d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
                      clipRule="evenodd"
                    />
                  </svg>
                )}
              </div>
              <p className="text-sm text-gray-600 dark:text-gray-400">{plan.description}</p>
            </label>
          ))}
        </div>
      </div>

      {/* Resource Quotas */}
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100">
            Resource Quotas
          </h3>
          <button
            type="button"
            onClick={() => setShowAdvanced(!showAdvanced)}
            className="text-sm text-blue-600 dark:text-blue-400 hover:underline"
          >
            {showAdvanced ? 'Hide' : 'Show'} Advanced
          </button>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {/* Max Users */}
          <div>
            <label
              htmlFor="maxUsers"
              className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
            >
              Max Users
            </label>
            <div className="relative">
              <input
                id="maxUsers"
                type="number"
                value={formData.maxUsers}
                onChange={e => updateField('maxUsers', parseInt(e.target.value) || 0)}
                className={`w-full px-4 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-gray-100 ${
                  errors.maxUsers ? 'border-red-500' : 'border-gray-300 dark:border-gray-600'
                }`}
                disabled={isLoading || formData.plan === 'enterprise'}
              />
              {formData.plan === 'enterprise' && (
                <span className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-500 text-sm">
                  Unlimited
                </span>
              )}
            </div>
          </div>

          {/* Max Printers */}
          <div>
            <label
              htmlFor="maxPrinters"
              className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
            >
              Max Printers
            </label>
            <div className="relative">
              <input
                id="maxPrinters"
                type="number"
                value={formData.maxPrinters}
                onChange={e => updateField('maxPrinters', parseInt(e.target.value) || 0)}
                className={`w-full px-4 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-gray-100 ${
                  errors.maxPrinters ? 'border-red-500' : 'border-gray-300 dark:border-gray-600'
                }`}
                disabled={isLoading || formData.plan === 'enterprise'}
              />
              {formData.plan === 'enterprise' && (
                <span className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-500 text-sm">
                  Unlimited
                </span>
              )}
            </div>
          </div>

          {/* Max Storage */}
          <div>
            <label
              htmlFor="maxStorageGB"
              className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
            >
              Max Storage (GB)
            </label>
            <div className="relative">
              <input
                id="maxStorageGB"
                type="number"
                value={formData.maxStorageGB}
                onChange={e => updateField('maxStorageGB', parseInt(e.target.value) || 0)}
                className={`w-full px-4 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-gray-100 ${
                  errors.maxStorageGB ? 'border-red-500' : 'border-gray-300 dark:border-gray-600'
                }`}
                disabled={isLoading || formData.plan === 'enterprise'}
              />
              {formData.plan === 'enterprise' && (
                <span className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-500 text-sm">
                  Unlimited
                </span>
              )}
            </div>
          </div>

          {/* Max Jobs Per Month */}
          {showAdvanced && (
            <div>
              <label
                htmlFor="maxJobsPerMonth"
                className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
              >
                Monthly Job Limit
              </label>
              <div className="relative">
                <input
                  id="maxJobsPerMonth"
                  type="number"
                  value={formData.maxJobsPerMonth}
                  onChange={e => updateField('maxJobsPerMonth', parseInt(e.target.value) || 0)}
                  className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-gray-100"
                  disabled={isLoading || formData.plan === 'enterprise'}
                />
                {formData.plan === 'enterprise' && (
                  <span className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-500 text-sm">
                    Unlimited
                  </span>
                )}
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Owner Selection (create mode only) */}
      {mode === 'create' && (
        <div className="space-y-4">
          <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100">
            Owner
          </h3>
          <div>
            <label
              htmlFor="ownerId"
              className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
            >
              Owner User ID <span className="text-red-500">*</span>
            </label>
            <input
              id="ownerId"
              type="text"
              value={formData.ownerId}
              onChange={e => updateField('ownerId', e.target.value)}
              placeholder="user_abc123"
              className={`w-full px-4 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-gray-100 ${
                errors.ownerId ? 'border-red-500' : 'border-gray-300 dark:border-gray-600'
              }`}
              disabled={isLoading}
            />
            {errors.ownerId && <p className="text-sm text-red-500 mt-1">{errors.ownerId}</p>}
          </div>
        </div>
      )}

      {/* Branding Settings */}
      {showAdvanced && (
        <div className="space-y-4">
          <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100">
            Branding
          </h3>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div>
              <label
                htmlFor="logoUrl"
                className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
              >
                Logo URL
              </label>
              <input
                id="logoUrl"
                type="url"
                value={formData.branding.logoUrl}
                onChange={e => updateNestedField('branding', 'logoUrl', e.target.value)}
                placeholder="https://example.com/logo.png"
                className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-gray-100"
                disabled={isLoading}
              />
            </div>
            <div>
              <label
                htmlFor="primaryColor"
                className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
              >
                Primary Color
              </label>
              <div className="flex items-center gap-2">
                <input
                  id="primaryColor"
                  type="color"
                  value={formData.branding.primaryColor}
                  onChange={e => updateNestedField('branding', 'primaryColor', e.target.value)}
                  className="w-12 h-10 border border-gray-300 dark:border-gray-600 rounded cursor-pointer"
                  disabled={isLoading}
                />
                <input
                  type="text"
                  value={formData.branding.primaryColor}
                  onChange={e => updateNestedField('branding', 'primaryColor', e.target.value)}
                  className="flex-1 px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-gray-100"
                  disabled={isLoading}
                />
              </div>
            </div>
            <div>
              <label
                htmlFor="customDomain"
                className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
              >
                Custom Domain
              </label>
              <input
                id="customDomain"
                type="text"
                value={formData.branding.customDomain}
                onChange={e => updateNestedField('branding', 'customDomain', e.target.value)}
                placeholder="print.acme.com"
                className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-gray-100"
                disabled={isLoading}
              />
            </div>
          </div>
        </div>
      )}

      {/* Security Settings */}
      {showAdvanced && (
        <div className="space-y-4">
          <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100">
            Security
          </h3>
          <div className="space-y-3">
            <label className="flex items-center gap-3">
              <input
                type="checkbox"
                checked={formData.security.requireMFA}
                onChange={e => updateNestedField('security', 'requireMFA', e.target.checked)}
                className="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
                disabled={isLoading}
              />
              <span className="text-sm text-gray-700 dark:text-gray-300">Require multi-factor authentication</span>
            </label>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label
                  htmlFor="passwordMinLength"
                  className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
                >
                  Minimum Password Length
                </label>
                <input
                  id="passwordMinLength"
                  type="number"
                  min="8"
                  max="64"
                  value={formData.security.passwordMinLength}
                  onChange={e => updateNestedField('security', 'passwordMinLength', parseInt(e.target.value) || 8)}
                  className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-gray-100"
                  disabled={isLoading}
                />
              </div>
              <div>
                <label
                  htmlFor="sessionTimeout"
                  className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
                >
                  Session Timeout (minutes)
                </label>
                <input
                  id="sessionTimeout"
                  type="number"
                  min="5"
                  max="1440"
                  value={formData.security.sessionTimeoutMinutes}
                  onChange={e => updateNestedField('security', 'sessionTimeoutMinutes', parseInt(e.target.value) || 60)}
                  className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent dark:bg-gray-700 dark:text-gray-100"
                  disabled={isLoading}
                />
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Form Actions */}
      <div className="flex items-center justify-end gap-3 pt-4 border-t border-gray-200 dark:border-gray-700">
        <button
          type="button"
          onClick={onCancel}
          disabled={isLoading}
          className="px-4 py-2 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors font-medium disabled:opacity-50"
        >
          Cancel
        </button>
        <button
          type="submit"
          disabled={isLoading}
          className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors font-medium disabled:opacity-50 flex items-center gap-2"
        >
          {isLoading ? (
            <>
              <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                <circle
                  className="opacity-25"
                  cx="12"
                  cy="12"
                  r="10"
                  stroke="currentColor"
                  strokeWidth="4"
                />
                <path
                  className="opacity-75"
                  fill="currentColor"
                  d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                />
              </svg>
              Saving...
            </>
          ) : mode === 'create' ? (
            'Create Organization'
          ) : (
            'Save Changes'
          )}
        </button>
      </div>
    </form>
  );
};

export default OrganizationForm;

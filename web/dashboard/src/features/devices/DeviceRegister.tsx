/**
 * DeviceRegister Component - Form for registering new printers and agents
 */

import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type {
  RegisterPrinterFormData,
  RegisterAgentFormData,
  RegisterPrinterFormErrors,
  RegisterAgentFormErrors,
  PrinterType,
} from './types';
import { devicesApi } from './api';
import { agentsApi } from '@/services/api';
import type { Agent } from '@/types';

type RegisterMode = 'printer' | 'agent';
type FormStep = 'select' | 'details';

interface DeviceRegisterProps {
  onSuccess?: () => void;
  onCancel?: () => void;
  defaultMode?: RegisterMode;
}

export const DeviceRegister = ({
  onSuccess,
  onCancel,
  defaultMode = 'printer',
}: DeviceRegisterProps) => {
  const queryClient = useQueryClient();
  const [mode, setMode] = useState<RegisterMode>(defaultMode);
  const [step, setStep] = useState<FormStep>('select');

  // Fetch agents for printer registration
  const { data: agents = [] } = useQuery({
    queryKey: ['agents'],
    queryFn: () => agentsApi.list(),
    enabled: mode === 'printer',
  });

  // Register printer mutation
  const registerPrinterMutation = useMutation({
    mutationFn: (data: RegisterPrinterFormData) => devicesApi.registerPrinter(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['devices'] });
      queryClient.invalidateQueries({ queryKey: ['printers'] });
      onSuccess?.();
    },
  });

  // Register agent mutation
  const registerAgentMutation = useMutation({
    mutationFn: (data: RegisterAgentFormData) => devicesApi.registerAgent(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['devices'] });
      queryClient.invalidateQueries({ queryKey: ['agents'] });
      onSuccess?.();
    },
  });

  const handleModeSelect = (selectedMode: RegisterMode) => {
    setMode(selectedMode);
    setStep('details');
  };

  const handleBack = () => {
    setStep('select');
  };

  if (step === 'select') {
    return <ModeSelection onSelect={handleModeSelect} onCancel={onCancel} />;
  }

  return mode === 'printer' ? (
    <PrinterRegisterForm
      agents={agents}
      onSubmit={(data) => registerPrinterMutation.mutate(data)}
      onCancel={onCancel}
      onBack={handleBack}
      isSubmitting={registerPrinterMutation.isPending}
      error={registerPrinterMutation.error}
    />
  ) : (
    <AgentRegisterForm
      onSubmit={(data) => registerAgentMutation.mutate(data)}
      onCancel={onCancel}
      onBack={handleBack}
      isSubmitting={registerAgentMutation.isPending}
      error={registerAgentMutation.error}
    />
  );
};

// Mode Selection Component
interface ModeSelectionProps {
  onSelect: (mode: RegisterMode) => void;
  onCancel?: () => void;
}

const ModeSelection = ({ onSelect, onCancel }: ModeSelectionProps) => {
  return (
    <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
      <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-2">
        Register New Device
      </h2>
      <p className="text-gray-600 dark:text-gray-400 mb-6">
        Choose the type of device you want to register
      </p>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
        <ModeCard
          title="Printer"
          description="Add a new printer to your organization"
          icon="printer"
          onClick={() => onSelect('printer')}
        />
        <ModeCard
          title="Agent"
          description="Register a new print agent"
          icon="agent"
          onClick={() => onSelect('agent')}
        />
      </div>

      {onCancel && (
        <div className="flex justify-end">
          <button
            onClick={onCancel}
            className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
          >
            Cancel
          </button>
        </div>
      )}
    </div>
  );
};

interface ModeCardProps {
  title: string;
  description: string;
  icon: string;
  onClick: () => void;
}

const ModeCard = ({ title, description, icon, onClick }: ModeCardProps) => (
  <button
    onClick={onClick}
    className="p-6 border border-gray-200 dark:border-gray-700 rounded-lg hover:border-blue-500 dark:hover:border-blue-500 hover:bg-blue-50 dark:hover:bg-blue-900/20 transition-all text-left group"
  >
    <div className="w-12 h-12 rounded-lg bg-gray-100 dark:bg-gray-700 group-hover:bg-blue-100 dark:group-hover:bg-blue-900/30 flex items-center justify-center mb-4 transition-colors">
      {icon === 'printer' ? (
        <PrinterIcon className="w-6 h-6 text-gray-600 dark:text-gray-400 group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors" />
      ) : (
        <AgentIcon className="w-6 h-6 text-gray-600 dark:text-gray-400 group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors" />
      )}
    </div>
    <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-1">
      {title}
    </h3>
    <p className="text-sm text-gray-600 dark:text-gray-400">{description}</p>
  </button>
);

// Printer Register Form Component
interface PrinterRegisterFormProps {
  agents: Agent[];
  onSubmit: (data: RegisterPrinterFormData) => void;
  onCancel?: () => void;
  onBack?: () => void;
  isSubmitting?: boolean;
  error?: Error | null;
}

const PrinterRegisterForm = ({
  agents,
  onSubmit,
  onCancel,
  onBack,
  isSubmitting = false,
  error = null,
}: PrinterRegisterFormProps) => {
  const [formData, setFormData] = useState<RegisterPrinterFormData>({
    name: '',
    type: 'network',
    agentId: '',
    capabilities: {
      supportsColor: true,
      supportsDuplex: true,
      supportedPaperSizes: ['A4', 'Letter'],
      resolution: '600 dpi',
    },
  });
  const [errors, setErrors] = useState<RegisterPrinterFormErrors>({});

  const validate = (): boolean => {
    const newErrors: RegisterPrinterFormErrors = {};

    if (!formData.name.trim()) {
      newErrors.name = 'Printer name is required';
    }
    if (!formData.agentId) {
      newErrors.agentId = 'Please select an agent';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (validate()) {
      onSubmit(formData);
    }
  };

  const updateCapability = <K extends keyof RegisterPrinterFormData['capabilities']>(
    key: K,
    value: RegisterPrinterFormData['capabilities'][K]
  ) => {
    setFormData((prev) => ({
      ...prev,
      capabilities: { ...prev.capabilities, [key]: value },
    }));
  };

  const togglePaperSize = (size: string) => {
    setFormData((prev) => ({
      ...prev,
      capabilities: {
        ...prev.capabilities,
        supportedPaperSizes: prev.capabilities.supportedPaperSizes.includes(size)
          ? prev.capabilities.supportedPaperSizes.filter((s) => s !== size)
          : [...prev.capabilities.supportedPaperSizes, size],
      },
    }));
  };

  return (
    <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100">
            Register Printer
          </h2>
          <p className="text-sm text-gray-600 dark:text-gray-400">
            Add a new printer to your organization
          </p>
        </div>
        {onBack && (
          <button
            onClick={onBack}
            className="text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-100"
          >
            <ArrowLeftIcon className="w-5 h-5" />
          </button>
        )}
      </div>

      {error && (
        <div className="mb-4 p-3 bg-red-100 dark:bg-red-900/30 border border-red-200 dark:border-red-800 rounded-lg">
          <p className="text-sm text-red-700 dark:text-red-300">
            {error.message || 'Failed to register printer. Please try again.'}
          </p>
        </div>
      )}

      <form onSubmit={handleSubmit} className="space-y-6">
        {/* Name */}
        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            Printer Name *
          </label>
          <input
            type="text"
            value={formData.name}
            onChange={(e) => setFormData({ ...formData, name: e.target.value })}
            className={`w-full px-3 py-2 border rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 ${
              errors.name
                ? 'border-red-300 dark:border-red-700'
                : 'border-gray-300 dark:border-gray-600'
            }`}
            placeholder="e.g., Office HP LaserJet"
          />
          {errors.name && (
            <p className="mt-1 text-sm text-red-600 dark:text-red-400">{errors.name}</p>
          )}
        </div>

        {/* Type */}
        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            Connection Type
          </label>
          <div className="flex gap-4">
            {(['usb', 'network', 'virtual'] as PrinterType[]).map((type) => (
              <label
                key={type}
                className={`flex items-center gap-2 px-4 py-2 border rounded-lg cursor-pointer transition-colors ${
                  formData.type === type
                    ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
                    : 'border-gray-300 dark:border-gray-600 hover:border-gray-400 dark:hover:border-gray-500'
                }`}
              >
                <input
                  type="radio"
                  name="type"
                  value={type}
                  checked={formData.type === type}
                  onChange={(e) =>
                    setFormData({ ...formData, type: e.target.value as PrinterType })
                  }
                  className="sr-only"
                />
                <span className="capitalize text-gray-900 dark:text-gray-100">{type}</span>
              </label>
            ))}
          </div>
        </div>

        {/* Agent */}
        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            Agent *
          </label>
          <select
            value={formData.agentId}
            onChange={(e) => setFormData({ ...formData, agentId: e.target.value })}
            className={`w-full px-3 py-2 border rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 ${
              errors.agentId
                ? 'border-red-300 dark:border-red-700'
                : 'border-gray-300 dark:border-gray-600'
            }`}
          >
            <option value="">Select an agent...</option>
            {agents
              .filter((a) => a.status === 'online')
              .map((agent) => (
                <option key={agent.id} value={agent.id}>
                  {agent.name} ({agent.platform})
                </option>
              ))}
          </select>
          {errors.agentId && (
            <p className="mt-1 text-sm text-red-600 dark:text-red-400">{errors.agentId}</p>
          )}
        </div>

        {/* Capabilities */}
        <div className="border border-gray-200 dark:border-gray-700 rounded-lg p-4">
          <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">
            Capabilities
          </h3>

          <div className="space-y-3">
            <label className="flex items-center gap-2">
              <input
                type="checkbox"
                checked={formData.capabilities.supportsColor}
                onChange={(e) => updateCapability('supportsColor', e.target.checked)}
                className="w-4 h-4 text-blue-600 rounded border-gray-300 focus:ring-2 focus:ring-blue-500"
              />
              <span className="text-sm text-gray-700 dark:text-gray-300">Supports Color</span>
            </label>

            <label className="flex items-center gap-2">
              <input
                type="checkbox"
                checked={formData.capabilities.supportsDuplex}
                onChange={(e) => updateCapability('supportsDuplex', e.target.checked)}
                className="w-4 h-4 text-blue-600 rounded border-gray-300 focus:ring-2 focus:ring-blue-500"
              />
              <span className="text-sm text-gray-700 dark:text-gray-300">Supports Duplex</span>
            </label>

            <div>
              <label className="block text-sm text-gray-700 dark:text-gray-300 mb-1">
                Paper Sizes
              </label>
              <div className="flex flex-wrap gap-2">
                {['A4', 'A3', 'Letter', 'Legal', 'Tabloid'].map((size) => (
                  <button
                    key={size}
                    type="button"
                    onClick={() => togglePaperSize(size)}
                    className={`px-3 py-1 text-sm rounded-lg transition-colors ${
                      formData.capabilities.supportedPaperSizes.includes(size)
                        ? 'bg-blue-600 text-white'
                        : 'bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-600'
                    }`}
                  >
                    {size}
                  </button>
                ))}
              </div>
            </div>

            <div>
              <label className="block text-sm text-gray-700 dark:text-gray-300 mb-1">
                Resolution
              </label>
              <select
                value={formData.capabilities.resolution}
                onChange={(e) => updateCapability('resolution', e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500"
              >
                <option value="300 dpi">300 dpi</option>
                <option value="600 dpi">600 dpi</option>
                <option value="1200 dpi">1200 dpi</option>
                <option value="2400 dpi">2400 dpi</option>
              </select>
            </div>
          </div>
        </div>

        {/* Actions */}
        <div className="flex justify-end gap-3">
          {onCancel && (
            <button
              type="button"
              onClick={onCancel}
              className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
            >
              Cancel
            </button>
          )}
          <button
            type="submit"
            disabled={isSubmitting}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isSubmitting ? 'Registering...' : 'Register Printer'}
          </button>
        </div>
      </form>
    </div>
  );
};

// Agent Register Form Component
interface AgentRegisterFormProps {
  onSubmit: (data: RegisterAgentFormData) => void;
  onCancel?: () => void;
  onBack?: () => void;
  isSubmitting?: boolean;
  error?: Error | null;
}

const AgentRegisterForm = ({
  onSubmit,
  onCancel,
  onBack,
  isSubmitting = false,
  error = null,
}: AgentRegisterFormProps) => {
  const [formData, setFormData] = useState<RegisterAgentFormData>({
    name: '',
    platform: 'windows',
    ipAddress: '',
  });
  const [errors, setErrors] = useState<RegisterAgentFormErrors>({});

  const validate = (): boolean => {
    const newErrors: RegisterAgentFormErrors = {};

    if (!formData.name.trim()) {
      newErrors.name = 'Agent name is required';
    }
    if (!formData.platform) {
      newErrors.platform = 'Platform is required';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (validate()) {
      onSubmit(formData);
    }
  };

  return (
    <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100">
            Register Agent
          </h2>
          <p className="text-sm text-gray-600 dark:text-gray-400">
            Register a new print agent manually
          </p>
        </div>
        {onBack && (
          <button
            onClick={onBack}
            className="text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-100"
          >
            <ArrowLeftIcon className="w-5 h-5" />
          </button>
        )}
      </div>

      {error && (
        <div className="mb-4 p-3 bg-red-100 dark:bg-red-900/30 border border-red-200 dark:border-red-800 rounded-lg">
          <p className="text-sm text-red-700 dark:text-red-300">
            {error.message || 'Failed to register agent. Please try again.'}
          </p>
        </div>
      )}

      <form onSubmit={handleSubmit} className="space-y-6">
        {/* Name */}
        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            Agent Name *
          </label>
          <input
            type="text"
            value={formData.name}
            onChange={(e) => setFormData({ ...formData, name: e.target.value })}
            className={`w-full px-3 py-2 border rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 ${
              errors.name
                ? 'border-red-300 dark:border-red-700'
                : 'border-gray-300 dark:border-gray-600'
            }`}
            placeholder="e.g., Office Front Desk"
          />
          {errors.name && (
            <p className="mt-1 text-sm text-red-600 dark:text-red-400">{errors.name}</p>
          )}
        </div>

        {/* Platform */}
        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            Platform *
          </label>
          <div className="grid grid-cols-3 gap-4">
            {(['windows', 'macos', 'linux'] as const).map((platform) => (
              <label
                key={platform}
                className={`flex items-center justify-center gap-2 px-4 py-3 border rounded-lg cursor-pointer transition-colors ${
                  formData.platform === platform
                    ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
                    : 'border-gray-300 dark:border-gray-600 hover:border-gray-400 dark:hover:border-gray-500'
                }`}
              >
                <input
                  type="radio"
                  name="platform"
                  value={platform}
                  checked={formData.platform === platform}
                  onChange={(e) => setFormData({ ...formData, platform: e.target.value })}
                  className="sr-only"
                />
                <PlatformIcon platform={platform} className="w-5 h-5" />
                <span className="capitalize text-gray-900 dark:text-gray-100">{platform}</span>
              </label>
            ))}
          </div>
          {errors.platform && (
            <p className="mt-1 text-sm text-red-600 dark:text-red-400">{errors.platform}</p>
          )}
        </div>

        {/* IP Address */}
        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            IP Address (optional)
          </label>
          <input
            type="text"
            value={formData.ipAddress}
            onChange={(e) => setFormData({ ...formData, ipAddress: e.target.value })}
            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500"
            placeholder="e.g., 192.168.1.100"
          />
          <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
            The agent will update this automatically when it connects
          </p>
        </div>

        {/* Actions */}
        <div className="flex justify-end gap-3">
          {onCancel && (
            <button
              type="button"
              onClick={onCancel}
              className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
            >
              Cancel
            </button>
          )}
          <button
            type="submit"
            disabled={isSubmitting}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isSubmitting ? 'Registering...' : 'Register Agent'}
          </button>
        </div>
      </form>
    </div>
  );
};

// Icons
const PrinterIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2h2m2 4h6a2 2 0 002-2v-4a2 2 0 00-2-2H9a2 2 0 00-2 2v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z"
    />
  </svg>
);

const AgentIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01"
    />
  </svg>
);

const ArrowLeftIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 19l-7-7m0 0l7-7m-7 7h18" />
  </svg>
);

const PlatformIcon = ({ platform, className }: { platform: string; className?: string }) => {
  if (platform === 'windows') {
    return (
      <svg className={className} viewBox="0 0 24 24" fill="currentColor">
        <path d="M0 3.449L9.75 2.1v9.451H0V3.449zm10.949-1.606L24 0v11.4H10.949V1.843zM0 12.6h9.75v9.451L0 20.699V12.6zm10.949 0H24V24l-12.9-1.801L10.949 12.6z" />
      </svg>
    );
  }
  if (platform === 'macos') {
    return (
      <svg className={className} viewBox="0 0 24 24" fill="currentColor">
        <path d="M18.71 19.5c-.83 1.24-1.71 2.45-3.05 2.47-1.34.03-1.77-.79-3.29-.79-1.53 0-2 .77-3.27.82-1.31.05-2.3-1.32-3.14-2.53C4.25 17 2.94 12.45 4.7 9.39c.87-1.52 2.43-2.48 4.12-2.51 1.28-.02 2.5.87 3.29.87.78 0 2.26-1.07 3.93-.91 1.32.05 2.31.53 2.94 1.37-2.58 1.42-2.14 5.27.66 6.38-.34.96-.83 1.89-1.43 2.91zM13 3.5c.68-.83 1.14-1.99 1.01-3.15-1.13.05-2.43.79-3.12 1.71-.66.88-1.16 2.09-1.01 3.15 1.25.09 2.44-.71 3.12-1.71z" />
      </svg>
    );
  }
  // linux
  return (
    <svg className={className} viewBox="0 0 24 24" fill="currentColor">
      <path d="M3.5 13c0 1.58.75 2.98 1.91 3.89-.04.21-.07.43-.07.65 0 1.76 1.43 3.19 3.19 3.19.22 0 .44-.03.65-.07.91 1.16 2.31 1.91 3.89 1.91s2.98-.75 3.89-1.91c.21.04.43.07.65.07 1.76 0 3.19-1.43 3.19-3.19 0-.22-.03-.44-.07-.65 1.16-.91 1.91-2.31 1.91-3.89s-.75-2.98-1.91-3.89c.04-.21.07-.43.07-.65 0-1.76-1.43-3.19-3.19-3.19-.22 0-.44.03-.65.07C14.48 4.17 13.08 3.42 11.5 3.42s-2.98.75-3.89 1.91c-.21-.04-.43-.07-.65-.07-1.76 0-3.19 1.43-3.19 3.19 0 .22.03.44.07.65C2.42 9.48 1.67 10.88 1.67 12.46c0 .18.01.36.03.54H3.5zM11.5 5.42c1.21 0 2.3.52 3.06 1.35-.05.02-.11.03-.16.05-.78.28-1.41.83-1.8 1.53-.16-.07-.33-.12-.5-.17-.23-.06-.47-.1-.71-.1-.18 0-.36.02-.53.06-.16.04-.32.09-.47.16-.39-.7-1.02-1.25-1.8-1.53-.05-.02-.11-.03-.16-.05.76-.83 1.85-1.35 3.06-1.35zm-5.09 3.4c.07 0 .14.01.21.03.16.04.31.11.44.2.13.09.24.2.33.33.09.13.16.28.2.44.04.16.06.32.05.49-.01.17-.05.33-.12.48-.07.15-.17.29-.29.4-.12.11-.26.21-.41.27-.15.07-.32.11-.49.12-.17.01-.33-.01-.49-.05-.16-.04-.31-.11-.44-.2-.13-.09-.24-.2-.33-.33-.09-.13-.16-.28-.2-.44-.04-.16-.06-.32-.05-.49.01-.17.05-.33.12-.48.07-.15.17-.29.29-.4.12-.11.26-.21.41-.27.15-.07.32-.11.49-.12.01 0 .03 0 .04 0zm10.18 0c.07 0 .14.01.21.03.16.04.31.11.44.2.13.09.24.2.33.33.09.13.16.28.2.44.04.16.06.32.05.49-.01.17-.05.33-.12.48-.07.15-.17.29-.29.4-.12.11-.26.21-.41.27-.15.07-.32.11-.49.12-.17.01-.33-.01-.49-.05-.16-.04-.31-.11-.44-.2-.13-.09-.24-.2-.33-.33-.09-.13-.16-.28-.2-.44-.04-.16-.06-.32-.05-.49.01-.17.05-.33.12-.48.07-.15.17-.29.29-.4.12-.11.26-.21.41-.27.15-.07.32-.11.49-.12.01 0 .03 0 .04 0z" />
    </svg>
  );
};

/**
 * DeviceDetail Component - Detailed view of a device (agent or printer) with status and configuration
 */

import { formatDistanceToNow } from 'date-fns';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { DeviceAgent, DevicePrinter } from './types';
import { DEVICE_STATUS_CONFIG } from './types';
import { devicesApi, isAgent } from './api';

interface DeviceDetailProps {
  deviceId: string;
  deviceType: 'agent' | 'printer';
  onEdit?: () => void;
  onDelete?: () => void;
  onBack?: () => void;
}

export const DeviceDetail = ({
  deviceId,
  deviceType,
  onEdit,
  onDelete,
  onBack,
}: DeviceDetailProps) => {
  const queryClient = useQueryClient();

  // Fetch device details
  const { data: device, isLoading, error } = useQuery({
    queryKey: ['device', deviceId, deviceType],
    queryFn: () => devicesApi.get(deviceId, deviceType),
    refetchInterval: (query) => {
      const data = query.state.data as DeviceAgent | DevicePrinter | undefined;
      // Poll every 10 seconds if device is online
      if (data) {
        const isOnline = isAgent(data)
          ? data.status === 'online'
          : data.isOnline;
        return isOnline ? 10000 : false;
      }
      return false;
    },
  });

  // Delete device mutation
  const deleteMutation = useMutation({
    mutationFn: () => devicesApi.delete(deviceId, deviceType),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['devices'] });
      onDelete?.();
    },
  });

  // Toggle printer active status mutation
  const toggleMutation = useMutation({
    mutationFn: () =>
      devicesApi.setPrinterActive(deviceId, !(device as DevicePrinter)?.isActive),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['device', deviceId, deviceType] });
    },
  });

  // Fetch agent's printers if this is an agent
  const { data: agentPrinters = [] } = useQuery({
    queryKey: ['agent-printers', deviceId],
    queryFn: () => devicesApi.getAgentPrinters(deviceId),
    enabled: deviceType === 'agent',
  });

  const handleDelete = () => {
    if (
      confirm(
        `Are you sure you want to delete this ${deviceType}? This action cannot be undone.`
      )
    ) {
      deleteMutation.mutate();
    }
  };

  const handleToggle = () => {
    if (device && !isAgent(device)) {
      toggleMutation.mutate();
    }
  };

  // Loading state
  if (isLoading) {
    return (
      <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
        <div className="animate-pulse space-y-4">
          <div className="h-8 w-48 bg-gray-200 dark:bg-gray-700 rounded" />
          <div className="h-4 w-32 bg-gray-200 dark:bg-gray-700 rounded" />
          <div className="h-32 bg-gray-100 dark:bg-gray-700 rounded mt-4" />
        </div>
      </div>
    );
  }

  // Error state
  if (error || !device) {
    return (
      <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
        <div className="text-center py-8">
          <svg
            className="mx-auto h-12 w-12 text-red-400"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
            />
          </svg>
          <h3 className="mt-2 text-sm font-medium text-gray-900 dark:text-gray-100">
            Error loading device
          </h3>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {(error as Error)?.message || 'Unable to load device details.'}
          </p>
        </div>
      </div>
    );
  }

  const isAgentDevice = isAgent(device);
  const status = isAgentDevice
    ? (device.status as 'online' | 'offline' | 'error')
    : device.isOnline
      ? 'online'
      : 'offline';
  const statusConfig = DEVICE_STATUS_CONFIG[status];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          {onBack && (
            <button
              onClick={onBack}
              className="p-2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-200 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
            >
              <ArrowLeftIcon className="w-5 h-5" />
            </button>
          )}
          <div
            className={`w-12 h-12 rounded-lg flex items-center justify-center ${statusConfig.bgColor}`}
          >
            {isAgentDevice ? (
              <AgentIcon className={`w-6 h-6 ${statusConfig.textColor}`} />
            ) : (
              <PrinterIcon className={`w-6 h-6 ${statusConfig.textColor}`} />
            )}
          </div>
          <div>
            <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">
              {device.name}
            </h1>
            <p className={`text-sm flex items-center gap-1.5 ${statusConfig.textColor}`}>
              <span className={`w-2 h-2 rounded-full ${statusConfig.dotColor}`} />
              {statusConfig.label}
            </p>
          </div>
        </div>

        <div className="flex items-center gap-2">
          {!isAgentDevice && (device as DevicePrinter).isActive !== undefined && (
            <button
              onClick={handleToggle}
              disabled={toggleMutation.isPending}
              className={`
                px-4 py-2 rounded-lg font-medium transition-colors
                ${(device as DevicePrinter).isActive
                  ? 'bg-orange-100 dark:bg-orange-900/30 text-orange-700 dark:text-orange-300 hover:bg-orange-200 dark:hover:bg-orange-900/50'
                  : 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300 hover:bg-green-200 dark:hover:bg-green-900/50'
                }
                disabled:opacity-50 disabled:cursor-not-allowed
              `}
            >
              {(device as DevicePrinter).isActive ? 'Disable' : 'Enable'}
            </button>
          )}
          {onEdit && (
            <button
              onClick={onEdit}
              className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-medium transition-colors"
            >
              Edit
            </button>
          )}
          <button
            onClick={handleDelete}
            disabled={deleteMutation.isPending}
            className="p-2 text-gray-400 hover:text-red-600 dark:hover:text-red-400 rounded-lg hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors disabled:opacity-50"
          >
            <TrashIcon className="w-5 h-5" />
          </button>
        </div>
      </div>

      {/* Status Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <StatusCard
          title="Status"
          value={statusConfig.label}
          color={status === 'online' ? 'green' : status === 'error' ? 'red' : 'gray'}
        />
        <StatusCard
          title="Last Heartbeat"
          value={
            device.uptime ||
            formatDistanceToNow(
              new Date((isAgentDevice ? undefined : (device as DevicePrinter).lastSeen) || device.createdAt),
              { addSuffix: true }
            )
          }
          color="blue"
        />
        <StatusCard
          title="Created"
          value={formatDistanceToNow(new Date(device.createdAt), { addSuffix: true })}
          color="blue"
        />
      </div>

      {/* Details Section */}
      <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 overflow-hidden">
        <div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
            Device Information
          </h2>
        </div>
        <div className="p-6">
          <dl className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {/* Common fields */}
            <DetailItem label="ID" value={device.id} />
            <DetailItem label="Name" value={device.name} />
            <DetailItem
              label="Type"
              value={isAgentDevice ? 'Agent' : `Printer (${(device as DevicePrinter).type})`}
            />
            <DetailItem
              label="Organization ID"
              value={device.orgId}
            />

            {/* Agent-specific fields */}
            {isAgentDevice && (
              <>
                <DetailItem label="Platform" value={(device as DeviceAgent).platform} />
                <DetailItem
                  label="Platform Version"
                  value={(device as DeviceAgent).platformVersion || 'Unknown'}
                />
                <DetailItem
                  label="Agent Version"
                  value={(device as DeviceAgent).agentVersion || 'Unknown'}
                />
                <DetailItem
                  label="IP Address"
                  value={(device as DeviceAgent).ipAddress || 'Unknown'}
                />
                <DetailItem
                  label="Max Job Size"
                  value={`${(device as DeviceAgent).capabilities?.maxJobSize ? `${(device as DeviceAgent).capabilities.maxJobSize / 1024 / 1024} MB` : 'Unknown'}`}
                />
                <DetailItem
                  label="Supports Color"
                  value={(device as DeviceAgent).capabilities?.supportsColor ? 'Yes' : 'No'}
                />
                <DetailItem
                  label="Supports Duplex"
                  value={(device as DeviceAgent).capabilities?.supportsDuplex ? 'Yes' : 'No'}
                />
                <DetailItem
                  label="Supported Formats"
                  value={(device as DeviceAgent).capabilities?.supportedFormats?.join(', ') || 'Unknown'}
                />
              </>
            )}

            {/* Printer-specific fields */}
            {!isAgentDevice && (
              <>
                <DetailItem
                  label="Agent"
                  value={(device as DevicePrinter).agentName || 'Unknown'}
                />
                <DetailItem
                  label="Driver"
                  value={(device as DevicePrinter).driver || 'Default'}
                />
                <DetailItem
                  label="Port"
                  value={(device as DevicePrinter).port || 'Default'}
                />
                <DetailItem
                  label="Active"
                  value={(device as DevicePrinter).isActive ? 'Yes' : 'No'}
                />
                <DetailItem
                  label="Supports Color"
                  value={(device as DevicePrinter).capabilities?.supportsColor ? 'Yes' : 'No'}
                />
                <DetailItem
                  label="Supports Duplex"
                  value={(device as DevicePrinter).capabilities?.supportsDuplex ? 'Yes' : 'No'}
                />
                <DetailItem
                  label="Resolution"
                  value={(device as DevicePrinter).capabilities?.resolution || 'Unknown'}
                />
                <DetailItem
                  label="Paper Sizes"
                  value={(device as DevicePrinter).capabilities?.supportedPaperSizes?.join(', ') || 'Unknown'}
                />
              </>
            )}
          </dl>
        </div>
      </div>

      {/* Agent Printers Section */}
      {isAgentDevice && agentPrinters.length > 0 && (
        <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 overflow-hidden">
          <div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
              Connected Printers ({agentPrinters.length})
            </h2>
          </div>
          <div className="divide-y divide-gray-200 dark:divide-gray-700">
            {agentPrinters.map((printer) => (
              <div
                key={printer.id}
                className="px-6 py-4 flex items-center justify-between hover:bg-gray-50 dark:hover:bg-gray-700/50"
              >
                <div className="flex items-center gap-3">
                  <div
                    className={`w-8 h-8 rounded-lg flex items-center justify-center ${
                      printer.isOnline
                        ? 'bg-green-100 dark:bg-green-900/30'
                        : 'bg-gray-100 dark:bg-gray-700'
                    }`}
                  >
                    <PrinterIcon
                      className={`w-4 h-4 ${
                        printer.isOnline
                          ? 'text-green-600 dark:text-green-400'
                          : 'text-gray-400'
                      }`}
                    />
                  </div>
                  <div>
                    <p className="text-sm font-medium text-gray-900 dark:text-gray-100">
                      {printer.name}
                    </p>
                    <p className="text-xs text-gray-500 dark:text-gray-400">
                      {printer.type} · {printer.driver || 'Default driver'}
                    </p>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <span
                    className={`px-2 py-1 text-xs font-medium rounded-full ${
                      printer.isOnline
                        ? 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300'
                        : 'bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400'
                    }`}
                  >
                    {printer.isOnline ? 'Online' : 'Offline'}
                  </span>
                  {!printer.isActive && (
                    <span className="px-2 py-1 text-xs font-medium rounded-full bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400">
                      Disabled
                    </span>
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Activity Timeline */}
      <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 overflow-hidden">
        <div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
            Recent Activity
          </h2>
        </div>
        <div className="p-6">
          <div className="space-y-4">
            <TimelineItem
              event="Heartbeat received"
              time={
                device.uptime ||
                formatDistanceToNow(
                  new Date((isAgentDevice ? undefined : (device as DevicePrinter).lastSeen) || device.createdAt),
                  { addSuffix: true }
                )
              }
              status="success"
            />
            <TimelineItem
              event="Device registered"
              time={formatDistanceToNow(new Date(device.createdAt), { addSuffix: true })}
              status="info"
            />
          </div>
        </div>
      </div>
    </div>
  );
};

// Status Card Component
interface StatusCardProps {
  title: string;
  value: string;
  color: 'green' | 'red' | 'blue' | 'gray';
}

const StatusCard = ({ title, value, color }: StatusCardProps) => {
  const colorClasses = {
    green: 'border-green-200 dark:border-green-800 bg-green-50 dark:bg-green-900/20',
    red: 'border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-900/20',
    blue: 'border-blue-200 dark:border-blue-800 bg-blue-50 dark:bg-blue-900/20',
    gray: 'border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800',
  };

  return (
    <div className={`border rounded-lg p-4 ${colorClasses[color]}`}>
      <p className="text-sm font-medium text-gray-600 dark:text-gray-400">{title}</p>
      <p className="text-lg font-semibold text-gray-900 dark:text-gray-100 mt-1">{value}</p>
    </div>
  );
};

// Detail Item Component
interface DetailItemProps {
  label: string;
  value: string | number | undefined;
}

const DetailItem = ({ label, value }: DetailItemProps) => (
  <div>
    <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">{label}</dt>
    <dd className="mt-1 text-sm text-gray-900 dark:text-gray-100 font-mono">
      {value ?? 'N/A'}
    </dd>
  </div>
);

// Timeline Item Component
interface TimelineItemProps {
  event: string;
  time: string;
  status: 'success' | 'error' | 'info' | 'warning';
}

const TimelineItem = ({ event, time, status }: TimelineItemProps) => {
  const statusConfig = {
    success: { bg: 'bg-green-500', icon: CheckIcon },
    error: { bg: 'bg-red-500', icon: XIcon },
    info: { bg: 'bg-blue-500', icon: InfoIcon },
    warning: { bg: 'bg-yellow-500', icon: WarningIcon },
  };

  const { bg } = statusConfig[status];

  return (
    <div className="flex items-start gap-3">
      <div className={`w-2 h-2 rounded-full ${bg} mt-1.5`} />
      <div className="flex-1 min-w-0">
        <p className="text-sm font-medium text-gray-900 dark:text-gray-100">{event}</p>
        <p className="text-xs text-gray-500 dark:text-gray-400">{time}</p>
      </div>
    </div>
  );
};

// Icons
const ArrowLeftIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 19l-7-7m0 0l7-7m-7 7h18" />
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

const CheckIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
  </svg>
);

const XIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
  </svg>
);

const InfoIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
    />
  </svg>
);

const WarningIcon = ({ className }: { className?: string }) => (
  <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
    />
  </svg>
);

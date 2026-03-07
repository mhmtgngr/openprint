/**
 * DeviceCard Component - Displays a single device (agent or printer) with status indicator
 */

import type { DeviceAgent, DevicePrinter } from './types';
import { DEVICE_STATUS_CONFIG } from './types';
import { isAgent } from './api';

interface DeviceCardProps {
  device: DeviceAgent | DevicePrinter;
  onClick?: () => void;
  onToggleStatus?: () => void;
  onDelete?: () => void;
  isToggling?: boolean;
}

export const DeviceCard = ({
  device,
  onClick,
  onToggleStatus,
  onDelete,
  isToggling = false,
}: DeviceCardProps) => {
  const isAgentDevice = isAgent(device);
  const status = isAgentDevice
    ? (device.status as 'online' | 'offline' | 'error')
    : device.isOnline
      ? 'online'
      : 'offline';
  const statusConfig = DEVICE_STATUS_CONFIG[status];

  const canToggle = !isAgentDevice && onToggleStatus;
  const isActive = isAgentDevice ? status === 'online' : (device as DevicePrinter).isActive;

  return (
    <div
      className={`
        bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700
        p-4 hover:shadow-md dark:hover:shadow-gray-900/50 transition-all
        ${onClick ? 'cursor-pointer' : ''}
      `}
      onClick={onClick}
    >
      {/* Header */}
      <div className="flex items-start justify-between mb-3">
        <div className="flex items-center gap-3">
          {/* Icon */}
          <div
            className={`
              w-10 h-10 rounded-lg flex items-center justify-center
              ${statusConfig.bgColor}
            `}
          >
            {isAgentDevice ? (
              <AgentIcon className={`w-5 h-5 ${statusConfig.textColor}`} />
            ) : (
              <PrinterIcon className={`w-5 h-5 ${statusConfig.textColor}`} />
            )}
          </div>

          {/* Name and status */}
          <div>
            <h3 className="font-medium text-gray-900 dark:text-gray-100">
              {device.name}
            </h3>
            <p className={`text-xs ${statusConfig.textColor} flex items-center gap-1`}>
              <span className={`w-2 h-2 rounded-full ${statusConfig.dotColor}`} />
              {statusConfig.label}
              {isAgentDevice && ` · ${(device as DeviceAgent).platform}`}
            </p>
          </div>
        </div>

        {/* Actions */}
        <div
          className="flex items-center gap-1"
          onClick={(e) => e.stopPropagation()}
        >
          {canToggle && (
            <button
              onClick={onToggleStatus}
              disabled={isToggling}
              className={`
                relative inline-flex h-6 w-11 items-center rounded-full transition-colors
                focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2
                ${isActive
                  ? 'bg-blue-600 dark:bg-blue-500'
                  : 'bg-gray-200 dark:bg-gray-700'
                }
                disabled:opacity-50 disabled:cursor-not-allowed
              `}
            >
              <span
                className={`
                  inline-block h-4 w-4 transform rounded-full bg-white transition-transform
                  ${isActive ? 'translate-x-6' : 'translate-x-1'}
                `}
              />
            </button>
          )}
          {onDelete && (
            <button
              onClick={onDelete}
              className="p-1.5 text-gray-400 hover:text-red-600 dark:hover:text-red-400 rounded-md hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
              title="Delete device"
            >
              <TrashIcon className="w-4 h-4" />
            </button>
          )}
        </div>
      </div>

      {/* Details */}
      <div className="space-y-2 text-sm">
        {/* Agent-specific details */}
        {isAgentDevice ? (
          <>
            <div className="flex justify-between text-gray-600 dark:text-gray-400">
              <span>Version</span>
              <span className="text-gray-900 dark:text-gray-100 font-mono text-xs">
                {(device as DeviceAgent).agentVersion || 'Unknown'}
              </span>
            </div>
            <div className="flex justify-between text-gray-600 dark:text-gray-400">
              <span>Printers</span>
              <span className="text-gray-900 dark:text-gray-100">
                {(device as DeviceAgent).printerCount || 0}
              </span>
            </div>
            {(device as DeviceAgent).ipAddress && (
              <div className="flex justify-between text-gray-600 dark:text-gray-400">
                <span>IP Address</span>
                <span className="text-gray-900 dark:text-gray-100 font-mono text-xs">
                  {(device as DeviceAgent).ipAddress}
                </span>
              </div>
            )}
          </>
        ) : (
          <>
            {/* Printer-specific details */}
            <div className="flex justify-between text-gray-600 dark:text-gray-400">
              <span>Type</span>
              <span className="text-gray-900 dark:text-gray-100 capitalize">
                {(device as DevicePrinter).type}
              </span>
            </div>
            <div className="flex justify-between text-gray-600 dark:text-gray-400">
              <span>Agent</span>
              <span className="text-gray-900 dark:text-gray-100">
                {(device as DevicePrinter).agentName || 'Unknown'}
              </span>
            </div>
            <div className="flex items-center gap-3 text-gray-600 dark:text-gray-400">
              <span>Capabilities</span>
              <div className="flex gap-2">
                {(device as DevicePrinter).capabilities?.supportsColor && (
                  <span
                    className="px-2 py-0.5 bg-purple-100 dark:bg-purple-900/30 text-purple-700 dark:text-purple-300 rounded text-xs"
                  >
                    Color
                  </span>
                )}
                {(device as DevicePrinter).capabilities?.supportsDuplex && (
                  <span
                    className="px-2 py-0.5 bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300 rounded text-xs"
                  >
                    Duplex
                  </span>
                )}
              </div>
            </div>
          </>
        )}
      </div>

      {/* Footer with last seen */}
      <div className="mt-3 pt-3 border-t border-gray-100 dark:border-gray-700">
        <div className="flex items-center justify-between text-xs text-gray-500 dark:text-gray-400">
          <span>Last seen</span>
          <span>
            {isAgentDevice
              ? (device as DeviceAgent).uptime || 'Unknown'
              : (device as DevicePrinter).uptime || 'Unknown'}
          </span>
        </div>
      </div>
    </div>
  );
};

// Icons
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

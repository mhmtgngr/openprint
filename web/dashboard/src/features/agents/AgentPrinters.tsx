/**
 * AgentPrinters Component
 * Shows printers discovered by a specific agent with capabilities and status
 */

import { formatDistanceToNow } from 'date-fns';
import { useAgentPrinters, useSetDefaultPrinter, useDeleteDiscoveredPrinter } from './useAgents';

interface AgentPrintersProps {
  agentId: string;
}

export const AgentPrinters = ({ agentId }: AgentPrintersProps) => {
  const { data: printers, isLoading, error } = useAgentPrinters(agentId);
  const setDefaultMutation = useSetDefaultPrinter();
  const deletePrinterMutation = useDeleteDiscoveredPrinter();

  const handleSetDefault = async (printerId: string) => {
    await setDefaultMutation.mutateAsync(printerId);
  };

  const handleDelete = async (printerId: string, printerName: string) => {
    if (
      confirm(
        `Are you sure you want to remove "${printerName}" from discovered printers?`
      )
    ) {
      await deletePrinterMutation.mutateAsync(printerId);
    }
  };

  if (isLoading) {
    return (
      <div className="space-y-3">
        {[...Array(3)].map((_, i) => (
          <div
            key={i}
            className="bg-gray-100 dark:bg-gray-800 rounded-lg h-24 animate-pulse"
          />
        ))}
      </div>
    );
  }

  if (error) {
    return (
      <div className="text-center py-12">
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
          Error loading printers
        </h3>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          {(error as Error).message}
        </p>
      </div>
    );
  }

  if (!printers || printers.length === 0) {
    return (
      <div className="text-center py-12">
        <svg
          className="mx-auto h-12 w-12 text-gray-400"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2h2m2 4h6a2 2 0 002-2v-4a2 2 0 00-2-2H9a2 2 0 00-2 2v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z"
          />
        </svg>
        <h3 className="mt-2 text-sm font-medium text-gray-900 dark:text-gray-100">
          No printers discovered
        </h3>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          This agent hasn't discovered any printers yet. Trigger a printer discovery to scan
          for available printers.
        </p>
      </div>
    );
  }

  const getPrinterTypeIcon = (type: string) => {
    switch (type) {
      case 'local':
        return (
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z"
            />
          </svg>
        );
      case 'network':
        return (
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M8.111 16.404a5.5 5.5 0 017.778 0M12 20h.01m-7.08-7.071c3.904-3.905 10.236-3.905 14.141 0M1.394 9.393c5.857-5.857 15.355-5.857 21.213 0"
            />
          </svg>
        );
      case 'shared':
        return (
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"
            />
          </svg>
        );
      default:
        return (
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2h2m2 4h6a2 2 0 002-2v-4a2 2 0 00-2-2H9a2 2 0 00-2 2v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z"
            />
          </svg>
        );
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'available':
        return 'bg-green-500';
      case 'offline':
        return 'bg-gray-400';
      case 'error':
        return 'bg-red-500';
      default:
        return 'bg-gray-400';
    }
  };

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {printers.map((printer) => (
          <div
            key={printer.id}
            className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4 hover:shadow-md transition-shadow"
          >
            <div className="flex items-start justify-between">
              <div className="flex items-center gap-3">
                <div className="text-gray-500">{getPrinterTypeIcon(printer.type)}</div>
                <div className="flex-1 min-w-0">
                  <h4 className="text-sm font-medium text-gray-900 dark:text-gray-100 truncate flex items-center gap-2">
                    {printer.name}
                    {printer.isDefault && (
                      <span className="px-1.5 py-0.5 text-xs font-medium bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300 rounded">
                        Default
                      </span>
                    )}
                  </h4>
                  <p className="text-xs text-gray-600 dark:text-gray-400">{printer.driver}</p>
                </div>
              </div>
              <div className="flex items-center gap-2">
                <span
                  className={`w-2 h-2 rounded-full ${getStatusColor(printer.status)}`}
                  title={printer.status}
                />
              </div>
            </div>

            <div className="mt-4 space-y-2">
              <div className="flex items-center justify-between text-xs">
                <span className="text-gray-600 dark:text-gray-400">Type</span>
                <span className="font-medium text-gray-900 dark:text-gray-100 capitalize">
                  {printer.type}
                </span>
              </div>
              <div className="flex items-center justify-between text-xs">
                <span className="text-gray-600 dark:text-gray-400">Port</span>
                <span className="font-medium text-gray-900 dark:text-gray-100">
                  {printer.port}
                </span>
              </div>
              <div className="flex items-center justify-between text-xs">
                <span className="text-gray-600 dark:text-gray-400">Resolution</span>
                <span className="font-medium text-gray-900 dark:text-gray-100">
                  {printer.capabilities.resolution}
                </span>
              </div>

              {/* Capabilities badges */}
              <div className="flex flex-wrap gap-1 mt-3">
                {printer.capabilities.supportsColor && (
                  <span className="px-2 py-0.5 text-xs font-medium bg-purple-50 text-purple-700 dark:bg-purple-900/20 dark:text-purple-300 rounded">
                    Color
                  </span>
                )}
                {printer.capabilities.supportsDuplex && (
                  <span className="px-2 py-0.5 text-xs font-medium bg-indigo-50 text-indigo-700 dark:bg-indigo-900/20 dark:text-indigo-300 rounded">
                    Duplex
                  </span>
                )}
                {printer.capabilities.supportedPaperSizes.length > 0 && (
                  <span className="px-2 py-0.5 text-xs font-medium bg-gray-50 text-gray-700 dark:bg-gray-700 dark:text-gray-300 rounded">
                    {printer.capabilities.supportedPaperSizes[0]}
                    {printer.capabilities.supportedPaperSizes.length > 1 && ` +${printer.capabilities.supportedPaperSizes.length - 1}`}
                  </span>
                )}
              </div>

              <div className="flex items-center justify-between text-xs text-gray-500 dark:text-gray-400 pt-2 border-t border-gray-100 dark:border-gray-700">
                <span>
                  Last seen:{' '}
                  {formatDistanceToNow(new Date(printer.lastSeen), { addSuffix: true })}
                </span>
              </div>
            </div>

            {/* Actions */}
            <div className="mt-4 flex items-center justify-between">
              {!printer.isDefault && printer.status === 'available' && (
                <button
                  onClick={() => handleSetDefault(printer.id)}
                  disabled={setDefaultMutation.isPending}
                  className="text-xs font-medium text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300 disabled:opacity-50"
                >
                  Set as Default
                </button>
              )}
              <button
                onClick={() => handleDelete(printer.id, printer.name)}
                disabled={deletePrinterMutation.isPending}
                className="text-xs font-medium text-red-600 hover:text-red-700 dark:text-red-400 dark:hover:text-red-300 disabled:opacity-50 ml-auto"
              >
                Remove
              </button>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};
